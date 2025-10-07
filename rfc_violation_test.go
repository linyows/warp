package warp

import (
	"bytes"
	"testing"
	"time"
)

// TestRFCViolation tests behavior with RFC 5321 violating SMTP commands
// RFC 5321 Section 3.3 states: "spaces are not permitted on either side of
// the colon following FROM in the MAIL command or TO in the RCPT command"
//
// The proxy now accepts RFC-violating commands to collect metadata, but logs warnings.
func TestRFCViolation(t *testing.T) {
	tests := []struct {
		name          string
		command       []byte
		expectAddr    []byte
		expectServer  []byte
		shouldMatch   bool
		isRFCViolation bool
	}{
		{
			name:           "RFC compliant: MAIL FROM:<address>",
			command:        []byte("MAIL FROM:<alice@example.com>\r\n"),
			expectAddr:     []byte("alice@example.com"),
			shouldMatch:    true,
			isRFCViolation: false,
		},
		{
			name:           "RFC violation: MAIL FROM: <address> (space after colon)",
			command:        []byte("MAIL FROM: <alice@example.com>\r\n"),
			expectAddr:     []byte("alice@example.com"),
			shouldMatch:    true,
			isRFCViolation: true,
		},
		{
			name:           "RFC violation: MAIL FROM : <address> (space before and after colon)",
			command:        []byte("MAIL FROM : <alice@example.com>\r\n"),
			expectAddr:     []byte("alice@example.com"),
			shouldMatch:    true,
			isRFCViolation: true,
		},
		{
			name:           "RFC violation: MAIL  FROM:<address> (double space)",
			command:        []byte("MAIL  FROM:<alice@example.com>\r\n"),
			expectAddr:     []byte("alice@example.com"),
			shouldMatch:    true,
			isRFCViolation: true,
		},
		{
			name:           "RFC compliant: RCPT TO:<address>",
			command:        []byte("RCPT TO:<bob@example.com>\r\n"),
			expectAddr:     []byte("bob@example.com"),
			expectServer:   []byte("example.com"),
			shouldMatch:    true,
			isRFCViolation: false,
		},
		{
			name:           "RFC violation: RCPT TO: <address> (space after colon)",
			command:        []byte("RCPT TO: <bob@example.com>\r\n"),
			expectAddr:     []byte("bob@example.com"),
			expectServer:   []byte("example.com"),
			shouldMatch:    true,
			isRFCViolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggedMsg := make(chan []byte, 1)
			pipe := &Pipe{afterCommHook: func(b Data, to Direction) {
				// Send logged message to channel (non-blocking)
				select {
				case loggedMsg <- b:
				default:
				}
			}}

			if len(tt.expectServer) > 0 {
				// Test RCPT TO
				pipe.setReceiverMailAddressAndServerName(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.rMailAddr) {
						t.Errorf("expected address %q, got %q", tt.expectAddr, pipe.rMailAddr)
					}
					if string(tt.expectServer) != string(pipe.rServerName) {
						t.Errorf("expected server %q, got %q", tt.expectServer, pipe.rServerName)
					}
				} else {
					if len(pipe.rMailAddr) > 0 {
						t.Errorf("expected no match, but got address %q", pipe.rMailAddr)
					}
				}
			} else {
				// Test MAIL FROM
				pipe.setSenderMailAddress(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.sMailAddr) {
						t.Errorf("expected address %q, got %q", tt.expectAddr, pipe.sMailAddr)
					}
				} else {
					if len(pipe.sMailAddr) > 0 {
						t.Errorf("expected no match, but got address %q", pipe.sMailAddr)
					}
				}
			}

			// Wait for goroutine to log (with timeout)
			var warningLogged bool
			select {
			case msg := <-loggedMsg:
				if bytes.Contains(msg, []byte("RFC 5321 violation")) {
					warningLogged = true
				}
			case <-time.After(50 * time.Millisecond):
				// Timeout - no message received
			}

			// Check if RFC violation warning was logged
			if tt.isRFCViolation && !warningLogged {
				t.Errorf("expected RFC violation warning to be logged, but none was found")
			}
			if !tt.isRFCViolation && warningLogged {
				t.Errorf("unexpected RFC violation warning logged for compliant command")
			}
		})
	}
}
