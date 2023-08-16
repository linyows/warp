package warp

import (
	"testing"
	"time"
)

func TestPairing(t *testing.T) {
	var tests = []struct {
		arg                  []byte
		expectSenderServer   []byte
		expectSenderAddr     []byte
		expectReceiverServer []byte
		expectReceiverAddr   []byte
	}{
		{
			arg:                  []byte("EHLO mx.example.local\r\n"),
			expectSenderServer:   []byte("mx.example.local"),
			expectSenderAddr:     nil,
			expectReceiverServer: nil,
			expectReceiverAddr:   nil,
		},
		{
			arg:                  []byte("HELO mx.example.local\r\n"),
			expectSenderServer:   []byte("mx.example.local"),
			expectSenderAddr:     nil,
			expectReceiverServer: nil,
			expectReceiverAddr:   nil,
		},
		{
			arg:                  []byte("MAIL FROM:<b-ob+foo@e-xample.local> SIZE=4095\r\n"),
			expectSenderServer:   nil,
			expectSenderAddr:     []byte("b-ob+foo@e-xample.local"),
			expectReceiverServer: nil,
			expectReceiverAddr:   nil,
		},
		{
			arg:                  []byte("RCPT TO:<alice@example.com>\r\n"),
			expectSenderServer:   nil,
			expectSenderAddr:     nil,
			expectReceiverServer: []byte("example.com"),
			expectReceiverAddr:   []byte("alice@example.com"),
		},
		{
			// Sender Rewriting Scheme
			arg:                  []byte("MAIL FROM:<SRS0=x/Eg=D3=example.test=alice@example.com> SIZE=4095\r\n"),
			expectSenderServer:   nil,
			expectSenderAddr:     []byte("SRS0=x/Eg=D3=example.test=alice@example.com"),
			expectReceiverServer: nil,
			expectReceiverAddr:   nil,
		},
		{
			// Pipelining
			arg:                  []byte("MAIL FROM:<bob@example.local> SIZE=4095\r\nRCPT TO:<alice@example.com> ORCPT=rfc822;bob@example.local\r\nDATA\r\n"),
			expectSenderServer:   nil,
			expectSenderAddr:     []byte("bob@example.local"),
			expectReceiverServer: []byte("example.com"),
			expectReceiverAddr:   []byte("alice@example.com"),
		},
	}
	for _, v := range tests {
		pipe := &Pipe{afterCommHook: func(b Data, to Direction) {}}
		pipe.pairing(v.arg)

		if v.expectSenderServer != nil && string(v.expectSenderServer) != string(pipe.sServerName) {
			t.Errorf("sender server name expected %s, but got %s", v.expectSenderServer, pipe.sServerName)
		}
		if v.expectSenderAddr != nil && string(v.expectSenderAddr) != string(pipe.sMailAddr) {
			t.Errorf("sender email address expected %s, but got %s", v.expectSenderAddr, pipe.sMailAddr)
		}
		if v.expectReceiverServer != nil && string(v.expectReceiverServer) != string(pipe.rServerName) {
			t.Errorf("receiver server name expected %s, but got %s", v.expectReceiverServer, pipe.rServerName)
		}
		if v.expectReceiverAddr != nil && string(v.expectReceiverAddr) != string(pipe.rMailAddr) {
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
			elapse: Elapse{Float64: float64(-9223372036854775808), Valid: true},
			expect: "-9223372036854775808sec",
		},
		{
			elapse: Elapse{Float64: 0, Valid: false},
			expect: "nil",
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
			start:  time.Date(2023, time.August, 16, 14, 48, 0, 0, time.Local),
			stop:   time.Date(2023, time.August, 16, 14, 48, 20, 0, time.Local),
			expect: Elapse{Float64: float64(20), Valid: true},
		},
		{
			start:  time.Time{},
			stop:   time.Date(2023, time.August, 16, 14, 48, 20, 0, time.Local),
			expect: Elapse{Float64: 0, Valid: false},
		},
		{
			start:  time.Date(2023, time.August, 16, 14, 48, 0, 0, time.Local),
			stop:   time.Time{},
			expect: Elapse{Float64: 0, Valid: false},
		},
	}

	for _, v := range tests {
		p := &Pipe{
			timeAtDataStarting: v.start,
			timeAtConnected:    v.stop,
		}
		got := p.elapse()
		if got != v.expect {
			t.Errorf("expected %#v got %#v", v.expect, got)
		}
	}
}
