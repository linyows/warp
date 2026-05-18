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

// BenchmarkSetTimeAtDataStarting_LargeBuffer measures the cost of the
// bytes.Split + [][]byte allocation that runs on EVERY downstream chunk,
// including chunks that can never contain a 354 response code.
func BenchmarkSetTimeAtDataStarting_LargeBuffer(b *testing.B) {
	const size = 1024 * 1024
	chunk := buildMailBodyChunk(size)
	p := newPerfPipe()
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.setTimeAtDataStarting(chunk)
	}
}

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
// of the always-on classifiers and setTimeAtDataStarting.
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

// TestMediateOnDownstream_RemoveStartTLSMisfire reproduces the bug described
// in PERFORMANCE_INVESTIGATION.md "発見 A": when a downstream chunk happens
// to contain both "250" and "STARTTLS" (here, the literal sequence
// "250-STARTTLS\r\n" embedded in a relayed mail body), the classifier
// returns true, removeStartTLSCommand runs bytes.Replace over the chunk,
// and p.readytls is wrongly flipped to true.
//
// This test asserts the CURRENT (buggy) behavior so the misfire is locked in
// as a regression. After the fix lands (e.g. an ehloResponseHandled latch on
// the Pipe), this test should be updated to assert that p.readytls remains
// false and that removeStartTLSCommand is NOT invoked.
func TestMediateOnDownstream_RemoveStartTLSMisfire(t *testing.T) {
	body := []byte("From: alice@example.test\r\n" +
		"To: bob@example.local\r\n" +
		"Subject: TLS notes\r\n" +
		"\r\n" +
		"Our EHLO continuation looks like 250-STARTTLS\r\n" +
		"...and that is the issue.\r\n")

	p := &Pipe{
		afterCommHook: func(b Data, to Direction) {},
		afterConnHook: func() {},
	}

	if !p.isResponseOfEHLOWithStartTLS(body) {
		t.Fatalf("precondition: classifier should misfire on the mail-body chunk, got false")
	}

	buf := make([]byte, len(body))
	copy(buf, body)
	_, _, _ = p.mediateOnDownstream(buf, len(buf))

	if !p.readytls {
		t.Fatalf("expected p.readytls=true after the misfired removeStartTLSCommand; got false. " +
			"If this trips, the misfire was likely fixed upstream — invert this assertion " +
			"(and rename the test) to lock in the fix.")
	}
}
