package warp

import (
	"testing"
)

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

	for _, v := range testcase {
		pipe := &Pipe{readytls: false}
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
