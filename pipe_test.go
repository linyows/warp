package warp

import (
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
