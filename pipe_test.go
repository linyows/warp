package warp

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestSetSenderServerName(t *testing.T) {
	var tests = []struct {
		arg                []byte
		expectSenderServer []byte
	}{
		{
			arg:                []byte("EHLO mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
		{
			arg:                []byte("HELO mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
		{
			// Case-insensitive: lowercase
			arg:                []byte("ehlo mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
		{
			// Case-insensitive: lowercase
			arg:                []byte("helo mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
		{
			// Case-insensitive: mixed case
			arg:                []byte("Ehlo mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
		{
			// Case-insensitive: mixed case
			arg:                []byte("Helo mx.example.local\r\n"),
			expectSenderServer: []byte("mx.example.local"),
		},
	}
	for _, v := range tests {
		pipe := &Pipe{afterCommHook: func(b Data, to Direction) {}}
		pipe.setSenderServerName(v.arg)
		if string(v.expectSenderServer) != string(pipe.sServerName) {
			t.Errorf("sender server name expected %s, but got %s", v.expectSenderServer, pipe.sServerName)
		}
	}
}

func TestSetSenderMailAddress(t *testing.T) {
	var tests = []struct {
		arg              []byte
		expectSenderAddr []byte
	}{
		{
			arg:              []byte("MAIL FROM:<b-ob+foo@e-xample.local> SIZE=4095\r\n"),
			expectSenderAddr: []byte("b-ob+foo@e-xample.local"),
		},
		{
			// Sender Rewriting Scheme
			arg:              []byte("MAIL FROM:<SRS0=x/Eg=D3=example.test=alice@example.com> SIZE=4095\r\n"),
			expectSenderAddr: []byte("SRS0=x/Eg=D3=example.test=alice@example.com"),
		},
		{
			// Pipelining
			arg:              []byte("MAIL FROM:<bob@example.local> SIZE=4095\r\nRCPT TO:<alice@example.com> ORCPT=rfc822;bob@example.local\r\nDATA\r\n"),
			expectSenderAddr: []byte("bob@example.local"),
		},
		{
			// Case-insensitive: lowercase
			arg:              []byte("mail from:<alice@example.test> SIZE=4095\r\n"),
			expectSenderAddr: []byte("alice@example.test"),
		},
		{
			// Case-insensitive: mixed case
			arg:              []byte("Mail From:<charlie@example.net> SIZE=4095\r\n"),
			expectSenderAddr: []byte("charlie@example.net"),
		},
	}
	for _, v := range tests {
		pipe := &Pipe{afterCommHook: func(b Data, to Direction) {}}
		pipe.setSenderMailAddress(v.arg)
		if string(v.expectSenderAddr) != string(pipe.sMailAddr) {
			t.Errorf("sender email address expected %s, but got %s", v.expectSenderAddr, pipe.sMailAddr)
		}
	}
}

func TestSetReceiverMailAddressAndServerName(t *testing.T) {
	var tests = []struct {
		arg                  []byte
		expectReceiverServer []byte
		expectReceiverAddr   []byte
	}{
		{
			arg:                  []byte("RCPT TO:<alice@example.com>\r\n"),
			expectReceiverServer: []byte("example.com"),
			expectReceiverAddr:   []byte("alice@example.com"),
		},
		{
			// Pipelining
			arg:                  []byte("MAIL FROM:<bob@example.local> SIZE=4095\r\nRCPT TO:<alice@example.com> ORCPT=rfc822;bob@example.local\r\nDATA\r\n"),
			expectReceiverServer: []byte("example.com"),
			expectReceiverAddr:   []byte("alice@example.com"),
		},
		{
			// Case-insensitive: lowercase
			arg:                  []byte("rcpt to:<bob@example.org>\r\n"),
			expectReceiverServer: []byte("example.org"),
			expectReceiverAddr:   []byte("bob@example.org"),
		},
		{
			// Case-insensitive: mixed case
			arg:                  []byte("Rcpt To:<charlie@example.net>\r\n"),
			expectReceiverServer: []byte("example.net"),
			expectReceiverAddr:   []byte("charlie@example.net"),
		},
	}
	for _, v := range tests {
		pipe := &Pipe{afterCommHook: func(b Data, to Direction) {}}
		pipe.setReceiverMailAddressAndServerName(v.arg)
		if string(v.expectReceiverServer) != string(pipe.rServerName) {
			t.Errorf("receiver server name expected %s, but got %s", v.expectReceiverServer, pipe.rServerName)
		}
		if string(v.expectReceiverAddr) != string(pipe.rMailAddr) {
			t.Errorf("receiver email address expected %s, but got %s", v.expectReceiverAddr, pipe.rMailAddr)
		}
	}
}

func TestIsResponseOfEHLOWithStartTLS(t *testing.T) {
	pipe := &Pipe{
		tls:    false,
		locked: false,
	}
	if !pipe.isResponseOfEHLOWithStartTLS([]byte("250-example.test\r\n250-PIPELINING\r\n250-8BITMIME\r\n250-SIZE 41943040\r\n250 STARTTLS\r\n")) {
		t.Errorf("expected true, but got false")
	}
}

func TestIsResponseOfEHLOWithoutStartTLS(t *testing.T) {
	pipe := &Pipe{
		tls:    false,
		locked: false,
	}
	if !pipe.isResponseOfEHLOWithoutStartTLS([]byte("250-example.test\r\n250-PIPELINING\r\n250-8BITMIME\r\n250 SIZE 41943040\r\n")) {
		t.Errorf("expected true, but got false")
	}
}

func TestIsResponseOfReadyToStartTLS(t *testing.T) {
	pipe := &Pipe{
		tls:    false,
		locked: true,
	}
	if !pipe.isResponseOfReadyToStartTLS([]byte("220 2.0.0 SMTP server ready\r\n")) {
		t.Errorf("expected true, but got false")
	}
}

func TestRemoveStartTLSCommand(t *testing.T) {
	var tests = []struct {
		ehloResp []byte
		ehloSize int
		expeResp []byte
		expeSize int
		expeTLS  bool
	}{
		{
			ehloResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-VRFY\r\n250-ETRN\r\n250-STARTTLS\r\n250-ENHANCEDSTATUSCODES\r\n250-8BITMIME\r\n250-DSN\r\n250-SMTPUTF8\r\n250 CHUNKING\r\n"),
			ehloSize: 174,
			expeResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-VRFY\r\n250-ETRN\r\n250-ENHANCEDSTATUSCODES\r\n250-8BITMIME\r\n250-DSN\r\n250-SMTPUTF8\r\n250 CHUNKING\r\n"),
			expeSize: 160,
			expeTLS:  true,
		},
		{
			ehloResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-8BITMIME\r\n250-SIZE 41943040\r\n250 STARTTLS\r\n"),
			ehloSize: 92,
			expeResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-8BITMIME\r\n250 SIZE 41943040\r\n"),
			expeSize: 78,
			expeTLS:  true,
		},
		{
			ehloResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-8BITMIME\r\n250 SIZE 41943040\r\n"),
			ehloSize: 78,
			expeResp: []byte("250-recipient@example.local\r\n250-PIPELINING\r\n250-8BITMIME\r\n250 SIZE 41943040\r\n"),
			expeSize: 78,
			expeTLS:  false,
		},
	}

	for _, v := range tests {
		pipe := &Pipe{readytls: false, afterCommHook: func(b Data, to Direction) {}}
		gotResp, gotSize := pipe.removeStartTLSCommand(v.ehloResp, v.ehloSize)
		if string(v.expeResp) != string(gotResp) {
			t.Errorf("response\nexpected:\n%sgot:\n%s", v.expeResp, gotResp)
		}
		if v.expeSize != gotSize {
			t.Errorf("size expected %#v got %#v", v.expeSize, gotSize)
		}
		if v.expeTLS != pipe.readytls {
			t.Errorf("tls expected %#v got %#v", v.expeTLS, pipe.readytls)
		}
	}
}

func TestElapseString(t *testing.T) {
	var tests = []struct {
		elapse Elapse
		expect string
	}{
		{
			elapse: 2147483647,
			expect: "2147483647 msec",
		},
	}

	for _, v := range tests {
		got := v.elapse.String()
		if got != v.expect {
			t.Errorf("expected %s got %s", v.expect, got)
		}
	}
}

func TestElapse(t *testing.T) {
	var tests = []struct {
		start  time.Time
		stop   time.Time
		expect Elapse
	}{
		{
			start:  time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC),
			stop:   time.Date(2023, time.August, 16, 14, 48, 20, 0, time.UTC),
			expect: 20000,
		},
		{
			start:  time.Time{},
			stop:   time.Date(2023, time.August, 16, 14, 48, 20, 0, time.UTC),
			expect: -1,
		},
		{
			start:  time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC),
			stop:   time.Time{},
			expect: -2,
		},
	}

	for _, v := range tests {
		p := &Pipe{
			timeAtConnected:    v.start,
			timeAtDataStarting: v.stop,
		}
		got := p.elapse()
		if got != v.expect {
			t.Errorf("expected %#v got %#v", v.expect, got)
		}
	}
}

func TestIsDataCommand(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect bool
	}{
		{"uppercase", []byte("DATA\r\n"), true},
		{"lowercase", []byte("data\r\n"), true},
		{"mixed case", []byte("DaTa\r\n"), true},
		{"with spaces", []byte("  DATA  \r\n"), true},
		{"bare", []byte("DATA"), true},
		{"not DATA command", []byte("MAIL FROM:<a@b.c>\r\n"), false},
		{"DATA prefix", []byte("DATA extra\r\n"), false},
		{"empty", []byte(""), false},
		{"partial", []byte("DAT\r\n"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pipe{}
			got := p.isDataCommand(tt.input)
			if got != tt.expect {
				t.Errorf("isDataCommand(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestIsActionCompletedResponse(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect bool
	}{
		{"250 OK", []byte("250 2.0.0 Ok: queued\r\n"), true},
		{"250 plain", []byte("250 Ok\r\n"), true},
		{"354 response", []byte("354 End data with <CR><LF>.<CR><LF>\r\n"), false},
		{"550 error", []byte("550 5.7.1 Spam detected\r\n"), false},
		{"empty", []byte(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pipe{}
			got := p.isActionCompletedResponse(tt.input)
			if got != tt.expect {
				t.Errorf("isActionCompletedResponse(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

// newTestPipeWithConns creates a Pipe with net.Pipe connections for testing.
// Returns the pipe and the "remote" ends of the connections (sRemote writes to sConn, rRemote reads from rConn).
func newTestPipeWithConns(t *testing.T) (*Pipe, net.Conn, net.Conn) {
	t.Helper()
	sClient, sServer := net.Pipe()
	rClient, rServer := net.Pipe()
	p := &Pipe{
		id:            "test-conn",
		sConn:         sServer,
		rConn:         rClient,
		bufferSize:    10240000,
		afterCommHook: func(b Data, to Direction) {},
		afterConnHook: func() {},
	}
	t.Cleanup(func() {
		_ = sClient.Close()
		_ = sServer.Close()
		_ = rClient.Close()
		_ = rServer.Close()
	})
	return p, sClient, rServer
}

func TestHandleDataPhaseUpstream_Relay(t *testing.T) {
	p, _, rRemote := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}
	p.sMailAddr = []byte("sender@example.test")
	p.rMailAddr = []byte("rcpt@example.local")
	p.senderIP = "192.168.1.1"
	p.sServerName = []byte("mx.example.test")

	var hookData *BeforeRelayData
	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		hookData = data
		return &FilterResult{Action: FilterRelay}
	}

	// Read from rRemote in background to avoid blocking
	var rBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := rRemote.Read(buf)
			if n > 0 {
				rBuf.Write(buf[:n])
			}
			// Stop when we've received the terminator
			if bytes.Contains(rBuf.Bytes(), []byte(".\r\n")) || err != nil {
				break
			}
		}
	}()

	message := []byte("From: sender@example.test\r\nSubject: Test\r\n\r\nHello\r\n.\r\n")
	buf := make([]byte, len(message))
	copy(buf, message)

	_, _, isContinue := p.handleDataPhaseUpstream(buf, len(message))

	// Wait for rRemote to receive data
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true for Relay action")
	}
	if p.inDataPhase {
		t.Error("expected inDataPhase=false after handling")
	}
	if p.dataBuffer != nil {
		t.Error("expected dataBuffer=nil after handling")
	}

	// Verify hook received correct data
	if hookData == nil {
		t.Fatal("BeforeRelay hook was not called")
	}
	if hookData.ConnID != "test-conn" {
		t.Errorf("ConnID = %q, want %q", hookData.ConnID, "test-conn")
	}
	if string(hookData.MailFrom) != "sender@example.test" {
		t.Errorf("MailFrom = %q, want %q", hookData.MailFrom, "sender@example.test")
	}
	if string(hookData.MailTo) != "rcpt@example.local" {
		t.Errorf("MailTo = %q, want %q", hookData.MailTo, "rcpt@example.local")
	}
	if hookData.SenderIP != "192.168.1.1" {
		t.Errorf("SenderIP = %q, want %q", hookData.SenderIP, "192.168.1.1")
	}
	if string(hookData.Helo) != "mx.example.test" {
		t.Errorf("Helo = %q, want %q", hookData.Helo, "mx.example.test")
	}

	// Verify message was relayed to rConn
	relayed := rBuf.String()
	if !bytes.Contains([]byte(relayed), []byte("Subject: Test")) {
		t.Errorf("relayed data missing Subject header: %q", relayed)
	}
	if !bytes.HasSuffix([]byte(relayed), []byte(".\r\n")) {
		t.Errorf("relayed data should end with terminator: %q", relayed)
	}
}

func TestHandleDataPhaseUpstream_Reject(t *testing.T) {
	p, _, rRemote := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}

	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		return &FilterResult{
			Action: FilterReject,
			Reply:  "550 5.7.1 Spam detected",
		}
	}

	// Read rRemote in background
	var rBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := rRemote.Read(buf)
			if n > 0 {
				rBuf.Write(buf[:n])
			}
			if bytes.Contains(rBuf.Bytes(), dataTerminator) || err != nil {
				break
			}
		}
	}()

	message := []byte("From: spam@evil.test\r\n\r\nBuy now!\r\n.\r\n")
	buf := make([]byte, len(message))
	copy(buf, message)

	_, _, isContinue := p.handleDataPhaseUpstream(buf, len(message))
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true for Reject action")
	}
	if p.filterRejectReply != "550 5.7.1 Spam detected" {
		t.Errorf("filterRejectReply = %q, want %q", p.filterRejectReply, "550 5.7.1 Spam detected")
	}

	// Verify empty terminator was sent to rConn
	if rBuf.String() != "\r\n.\r\n" {
		t.Errorf("expected empty terminator sent to rConn, got %q", rBuf.String())
	}
}

func TestHandleDataPhaseUpstream_AddHeader(t *testing.T) {
	p, _, rRemote := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}

	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		modified := append([]byte("X-Spam-Score: 0.1\r\n"), data.Message...)
		return &FilterResult{
			Action:  FilterAddHeader,
			Message: modified,
		}
	}

	var rBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := rRemote.Read(buf)
			if n > 0 {
				rBuf.Write(buf[:n])
			}
			if bytes.Contains(rBuf.Bytes(), []byte(".\r\n")) || err != nil {
				break
			}
		}
	}()

	message := []byte("From: test@example.test\r\nSubject: Hi\r\n\r\nBody\r\n.\r\n")
	buf := make([]byte, len(message))
	copy(buf, message)

	_, _, isContinue := p.handleDataPhaseUpstream(buf, len(message))
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true for AddHeader action")
	}

	relayed := rBuf.String()
	if !bytes.Contains([]byte(relayed), []byte("X-Spam-Score: 0.1")) {
		t.Errorf("relayed data missing added header: %q", relayed)
	}
	if !bytes.Contains([]byte(relayed), []byte("Subject: Hi")) {
		t.Errorf("relayed data missing original header: %q", relayed)
	}
	if !bytes.HasSuffix([]byte(relayed), []byte(".\r\n")) {
		t.Errorf("relayed data should end with terminator: %q", relayed)
	}
}

func TestHandleDataPhaseUpstream_Buffering(t *testing.T) {
	p, _, _ := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}
	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		return &FilterResult{Action: FilterRelay}
	}

	// First chunk: incomplete message (no terminator)
	chunk1 := []byte("From: test@example.test\r\nSubject: Hi\r\n\r\nHello ")
	buf := make([]byte, len(chunk1))
	copy(buf, chunk1)

	_, _, isContinue := p.handleDataPhaseUpstream(buf, len(chunk1))
	if !isContinue {
		t.Error("expected isContinue=true for incomplete data")
	}
	if !p.inDataPhase {
		t.Error("should still be in DATA phase after incomplete chunk")
	}
	if p.dataBuffer == nil {
		t.Error("dataBuffer should not be nil while buffering")
	}
	if p.dataBuffer.Len() != len(chunk1) {
		t.Errorf("dataBuffer length = %d, want %d", p.dataBuffer.Len(), len(chunk1))
	}
}

func TestHandleDataPhaseUpstream_BufferOverflow(t *testing.T) {
	p, sRemote, rRemote := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}
	p.dataBufferSize = 50 // Small limit
	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		t.Fatal("filter hook should not be called on buffer overflow")
		return nil
	}

	// Read responses from both connections in background
	var sBuf bytes.Buffer
	var rBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		n, _ := sRemote.Read(buf)
		if n > 0 {
			sBuf.Write(buf[:n])
		}
	}()
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		n, _ := rRemote.Read(buf)
		if n > 0 {
			rBuf.Write(buf[:n])
		}
	}()

	// Write enough to exceed the buffer limit
	p.dataBuffer.Write([]byte("Already 40 bytes of data in the buffer!"))
	overflow := []byte("This additional data exceeds the limit\r\n.\r\n")
	buf := make([]byte, len(overflow))
	copy(buf, overflow)

	_, _, isContinue := p.handleDataPhaseUpstream(buf, len(overflow))
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true on buffer overflow")
	}
	if p.inDataPhase {
		t.Error("expected inDataPhase=false after overflow")
	}
	if p.dataBuffer != nil {
		t.Error("expected dataBuffer=nil after overflow")
	}

	// Verify 552 error sent to client
	if !bytes.Contains(sBuf.Bytes(), []byte("552")) {
		t.Errorf("expected 552 error sent to client, got %q", sBuf.String())
	}

	// Verify terminator sent to server
	if !bytes.Contains(rBuf.Bytes(), dataTerminator) {
		t.Errorf("expected terminator sent to server, got %q", rBuf.String())
	}
}

func TestMediateOnUpstream_FilterHookDataCommand(t *testing.T) {
	p := &Pipe{
		afterCommHook: func(b Data, to Direction) {},
		beforeRelayHook: func(data *BeforeRelayData) *FilterResult {
			return &FilterResult{Action: FilterRelay}
		},
	}

	data := []byte("DATA\r\n")
	buf := make([]byte, 1024)
	copy(buf, data)

	_, _, isContinue := p.mediateOnUpstream(buf, len(data))

	if isContinue {
		t.Error("DATA command should be relayed (isContinue=false)")
	}
	if !p.dataCommandSent {
		t.Error("dataCommandSent should be true after DATA command")
	}
	if p.dataBuffer == nil {
		t.Error("dataBuffer should be initialized after DATA command")
	}
}

func TestMediateOnUpstream_NoFilterHookBypass(t *testing.T) {
	p := &Pipe{
		afterCommHook:   func(b Data, to Direction) {},
		beforeRelayHook: nil, // No filter hook
	}

	data := []byte("DATA\r\n")
	buf := make([]byte, 1024)
	copy(buf, data)

	_, _, isContinue := p.mediateOnUpstream(buf, len(data))

	if isContinue {
		t.Error("without filter hook, should not suppress relay")
	}
	if p.dataCommandSent {
		t.Error("dataCommandSent should remain false without filter hook")
	}
}

func TestMediateOnUpstream_DelegatesInDataPhase(t *testing.T) {
	p, _, rRemote := newTestPipeWithConns(t)
	p.inDataPhase = true
	p.dataBuffer = &bytes.Buffer{}

	hookCalled := false
	p.beforeRelayHook = func(data *BeforeRelayData) *FilterResult {
		hookCalled = true
		return &FilterResult{Action: FilterRelay}
	}

	// Read rRemote in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.ReadAll(rRemote)
	}()

	message := []byte("Subject: test\r\n\r\nBody\r\n.\r\n")
	buf := make([]byte, 1024)
	copy(buf, message)

	_, _, isContinue := p.mediateOnUpstream(buf, len(message))
	// Close to unblock reader
	_ = p.rConn.Close()
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true in DATA phase")
	}
	if !hookCalled {
		t.Error("expected BeforeRelay hook to be called")
	}
}

func TestMediateOnDownstream_354EntersDataPhase(t *testing.T) {
	p := &Pipe{
		afterCommHook: func(b Data, to Direction) {},
		beforeRelayHook: func(data *BeforeRelayData) *FilterResult {
			return &FilterResult{Action: FilterRelay}
		},
		dataCommandSent: true,
	}

	resp := []byte("354 End data with <CR><LF>.<CR><LF>\r\n")
	buf := make([]byte, 1024)
	copy(buf, resp)

	_, _, _ = p.mediateOnDownstream(buf, len(resp))

	if !p.inDataPhase {
		t.Error("expected inDataPhase=true after 354 response")
	}
	if p.dataCommandSent {
		t.Error("expected dataCommandSent=false after 354 response")
	}
}

func TestMediateOnDownstream_354IgnoredWithoutFilterHook(t *testing.T) {
	p := &Pipe{
		afterCommHook:   func(b Data, to Direction) {},
		beforeRelayHook: nil,
		dataCommandSent: true,
	}

	resp := []byte("354 End data with <CR><LF>.<CR><LF>\r\n")
	buf := make([]byte, 1024)
	copy(buf, resp)

	_, _, _ = p.mediateOnDownstream(buf, len(resp))

	if p.inDataPhase {
		t.Error("inDataPhase should remain false without filter hook")
	}
}

func TestMediateOnDownstream_RejectReplySubstitution(t *testing.T) {
	p, sRemote, _ := newTestPipeWithConns(t)
	p.filterRejectReply = "550 5.7.1 Spam detected"

	// Read reject reply from sRemote
	var sBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		n, _ := sRemote.Read(buf)
		if n > 0 {
			sBuf.Write(buf[:n])
		}
	}()

	resp := []byte("250 2.0.0 Ok: queued\r\n")
	buf := make([]byte, 1024)
	copy(buf, resp)

	_, _, isContinue := p.mediateOnDownstream(buf, len(resp))
	wg.Wait()

	if !isContinue {
		t.Error("expected isContinue=true to suppress server's 250 OK")
	}
	if p.filterRejectReply != "" {
		t.Error("filterRejectReply should be cleared after substitution")
	}
	if !bytes.Contains(sBuf.Bytes(), []byte("550 5.7.1 Spam detected")) {
		t.Errorf("expected reject reply sent to client, got %q", sBuf.String())
	}
}

func TestMediateOnDownstream_NoSubstitutionWithoutRejectReply(t *testing.T) {
	p := &Pipe{
		afterCommHook:     func(b Data, to Direction) {},
		filterRejectReply: "",
	}

	resp := []byte("250 2.0.0 Ok: queued\r\n")
	buf := make([]byte, 1024)
	copy(buf, resp)

	_, _, isContinue := p.mediateOnDownstream(buf, len(resp))

	if isContinue {
		t.Error("should not suppress 250 OK when no reject reply is set")
	}
}

func TestMediateOnUpstream_MetadataExtractionWithFilterHook(t *testing.T) {
	p := &Pipe{
		afterCommHook: func(b Data, to Direction) {},
		beforeRelayHook: func(data *BeforeRelayData) *FilterResult {
			return &FilterResult{Action: FilterRelay}
		},
	}

	// MAIL FROM should still be extracted even with filter hook
	data := []byte("MAIL FROM:<alice@example.test> SIZE=4095\r\n")
	buf := make([]byte, 1024)
	copy(buf, data)

	p.mediateOnUpstream(buf, len(data))

	if string(p.sMailAddr) != "alice@example.test" {
		t.Errorf("sMailAddr = %q, want %q", p.sMailAddr, "alice@example.test")
	}
}
