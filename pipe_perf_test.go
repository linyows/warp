package warp

import (
	"bytes"
	"testing"
)

// newPerfPipe constructs a Pipe with a no-op afterCommHook and minimal state
// so that mediator functions can be called in isolation by benchmarks/tests.
func newPerfPipe() *Pipe {
	return &Pipe{
		bufferSize:    10 * 1024 * 1024,
		afterCommHook: func(b Data, to Direction) {},
		afterConnHook: func() {},
	}
}

// buildEHLOResponseChunk returns a typical EHLO response advertising STARTTLS.
func buildEHLOResponseChunk() []byte {
	return []byte(
		"250-mail.example.com\r\n" +
			"250-PIPELINING\r\n" +
			"250-SIZE 10240000\r\n" +
			"250-STARTTLS\r\n" +
			"250 8BITMIME\r\n",
	)
}

// buildMailBodyChunk returns a synthetic mail body chunk of the given size.
// It deliberately embeds substrings that mimic real-world false positives for
// the SMTP-response classifiers:
//   - "250" (in a Received header)
//   - "STARTTLS" (as a literal word in documentation-like body text)
//   - the exact sequence "250-STARTTLS\r\n" that the misfired
//     removeStartTLSCommand attempts to bytes.Replace
//
// The result is padded with non-matching text up to size bytes.
func buildMailBodyChunk(size int) []byte {
	header := "Received: from sender.example (sender.example [192.0.2.250])\r\n" +
		"\tby mx.example.com with ESMTP id 250abc; deployment notes\r\n" +
		"From: alice@example.test\r\n" +
		"To: bob@example.local\r\n" +
		"Subject: Notes on TLS deployment\r\n" +
		"\r\n" +
		"Our migration plan mentions 250-STARTTLS\r\n" +
		"as an EHLO continuation marker.\r\n"
	var buf bytes.Buffer
	buf.WriteString(header)
	for buf.Len() < size {
		buf.WriteString("Lorem ipsum dolor sit amet. ")
	}
	body := buf.Bytes()
	if len(body) > size {
		body = body[:size]
	}
	return body
}

// ─────────────────────────────────────────────────────────────────────
// Layer A: Function-level benchmarks for the hot paths called out in
// PERFORMANCE_INVESTIGATION.md (bytes.Split / bytes.Contains / Replace).
//
// Each benchmark records B/op and allocs/op via b.ReportAllocs so that the
// effect of upcoming fixes can be measured numerically. b.SetBytes is set to
// the input chunk size so `MB/s` throughput is also reported.
// ─────────────────────────────────────────────────────────────────────

// BenchmarkHasResponseCode_LargeBuffer measures bytes.Split inside
// hasResponseCode against a large buffer.
func BenchmarkHasResponseCode_LargeBuffer(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.hasResponseCode(chunk, codeStartingMailInput)
	}
}

// BenchmarkIsResponseOfEHLOWithStartTLS_MailBody shows the cost of the two
// bytes.Contains scans over a large buffer.
func BenchmarkIsResponseOfEHLOWithStartTLS_MailBody(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.isResponseOfEHLOWithStartTLS(chunk)
	}
}

// BenchmarkMediateOnDownstream_LargeBuffer measures the per-chunk cost of
// mediateOnDownstream on a 1 MiB chunk. p.tls is forced to true so that the
// removeStartTLSCommand path is suppressed; this isolates the scan-only cost
// of the always-on classifiers and the inline 354 detection.
func BenchmarkMediateOnDownstream_LargeBuffer(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	p.tls = true
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnDownstream(buf, len(buf))
	}
}

// BenchmarkMediateOnDownstream_LargeBuffer_WithMisfire enables the
// removeStartTLSCommand misfire path (p.tls=false) so the per-iteration cost
// includes bytes.Replace allocations over the full buffer when the chunk
// happens to contain the literal "250-STARTTLS\r\n" sequence.
//
// This benchmark numerically captures the cost of PERFORMANCE_INVESTIGATION.md
// "発見 A" (1.5 GB bytes.Replace alloc traffic in production).
func BenchmarkMediateOnDownstream_LargeBuffer_WithMisfire(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnDownstream(buf, len(buf))
	}
}

// BenchmarkMediateOnUpstream_DataBody_PlainSMTP measures upstream mediation
// when the receiver mail address is already known (RCPT TO already processed):
// the regex-based extraction is skipped per the existing guard.
func BenchmarkMediateOnUpstream_DataBody_PlainSMTP(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	p.rMailAddr = []byte("bob@example.local")
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnUpstream(buf, len(buf))
	}
}

// BenchmarkMediateOnUpstream_InDataPhase represents the post-fix steady state
// of upstream mediation for a non-filter connection: the proxy has observed
// the server's 354 reply downstream, set inDataPhase=true, and every
// subsequent upstream chunk is mail body bytes. The fast path bypasses all
// command/regex scans and only looks for the end-of-data terminator.
func BenchmarkMediateOnUpstream_InDataPhase(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	p.inDataPhase = true
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnUpstream(buf, len(buf))
	}
}

// BenchmarkMediateOnUpstream_DataBody_NoRcptSet measures upstream mediation
// when rMailAddr has not been set yet: every chunk runs the regex-based
// sender/receiver extraction over the full buffer.
func BenchmarkMediateOnUpstream_DataBody_NoRcptSet(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnUpstream(buf, len(buf))
	}
}

// BenchmarkMediateOnDownstream_EHLOResponse is a baseline for comparison:
// a real EHLO response is small (~100 B), so the per-chunk scan cost should
// be negligible compared to the large-buffer benchmarks above.
func BenchmarkMediateOnDownstream_EHLOResponse(b *testing.B) {
	chunk := buildEHLOResponseChunk()
	p := newPerfPipe()
	p.tls = true // suppress STARTTLS path; we want the scan cost only
	buf := make([]byte, 0, len(chunk))
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf[:0], chunk...)
		_, _, _ = p.mediateOnDownstream(buf, len(buf))
	}
}

// ─────────────────────────────────────────────────────────────────────
// Layer B: Regression test documenting the removeStartTLSCommand misfire.
// ─────────────────────────────────────────────────────────────────────

// TestMediateOnDownstream_RemoveStartTLSNoMisfireAfterEHLO verifies the fix
// for PERFORMANCE_INVESTIGATION.md "発見 A": once the first EHLO response
// has been handled, downstream chunks that happen to contain "250" and
// "STARTTLS" — e.g. a mail body where "250-STARTTLS\r\n" appears in
// documentation text — must NOT retrigger removeStartTLSCommand, must NOT
// allocate a full-buffer bytes.Replace copy, and must NOT flip p.readytls.
func TestMediateOnDownstream_RemoveStartTLSNoMisfireAfterEHLO(t *testing.T) {
	p := &Pipe{
		afterCommHook: func(b Data, to Direction) {},
		afterConnHook: func() {},
	}

	// First, process a legitimate EHLO response so the ehloResponseHandled
	// latch is engaged. The buffer is a complete "250-...\r\n...250 ...\r\n"
	// response advertising STARTTLS — the real classifier target.
	ehlo := buildEHLOResponseChunk()
	if !p.isResponseOfEHLOWithStartTLS(ehlo) {
		t.Fatalf("precondition: classifier should fire on a real EHLO response")
	}
	ehloBuf := make([]byte, len(ehlo))
	copy(ehloBuf, ehlo)
	_, _, _ = p.mediateOnDownstream(ehloBuf, len(ehloBuf))
	if !p.readytls {
		t.Fatalf("expected p.readytls=true after handling a real EHLO STARTTLS response, got false")
	}
	if !p.ehloResponseHandled {
		t.Fatalf("expected ehloResponseHandled=true after first EHLO response, got false")
	}

	// Now simulate the dangerous case: a downstream chunk that mimics a
	// mail body containing both "250" and "STARTTLS" (including the literal
	// "250-STARTTLS\r\n" sequence). The classifier must return false and
	// removeStartTLSCommand must NOT be invoked.
	body := []byte("From: alice@example.test\r\n" +
		"To: bob@example.local\r\n" +
		"Subject: TLS notes\r\n" +
		"\r\n" +
		"Our EHLO continuation looks like 250-STARTTLS\r\n" +
		"...and that is the issue.\r\n")

	if p.isResponseOfEHLOWithStartTLS(body) {
		t.Fatalf("expected classifier to suppress misfire on mail-body chunk after EHLO was handled, got true")
	}

	// Reset p.readytls so we can assert mediateOnDownstream does NOT set it.
	p.readytls = false
	bodyBuf := make([]byte, len(body))
	copy(bodyBuf, body)
	_, _, _ = p.mediateOnDownstream(bodyBuf, len(bodyBuf))

	if p.readytls {
		t.Fatalf("expected p.readytls=false after mail-body chunk; got true (misfire regression)")
	}
}
