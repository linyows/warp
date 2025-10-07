package warp

import (
	"bytes"
	"testing"
	"time"
)

// TestRFCViolation tests behavior with RFC 5321 violating SMTP commands and email addresses
// References:
// - RFC 5321 Section 2.4: Command verbs are case-insensitive
// - RFC 5321 Section 3.3: "spaces are not permitted on either side of the colon"
// - https://www.docomo.ne.jp/service/docomo_mail/rfc_add/
// - https://www.sonoko.co.jp/user_data/oshirase10.php
//
// The proxy accepts RFC-violating commands to collect metadata, but logs warnings for spacing violations.
func TestRFCViolation(t *testing.T) {
	tests := []struct {
		name          string
		command       []byte
		expectAddr    []byte
		expectServer  []byte
		shouldMatch   bool
		expectWarning bool
		description   string
	}{
		// === RFC Compliant Cases ===
		{
			name:          "RFC compliant: MAIL FROM:<address>",
			command:       []byte("MAIL FROM:<alice@example.com>\r\n"),
			expectAddr:    []byte("alice@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Standard RFC-compliant MAIL FROM",
		},
		{
			name:          "RFC compliant: RCPT TO:<address>",
			command:       []byte("RCPT TO:<bob@example.com>\r\n"),
			expectAddr:    []byte("bob@example.com"),
			expectServer:  []byte("example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Standard RFC-compliant RCPT TO",
		},

		// === RFC Violation: Command Spacing ===
		{
			name:          "RFC violation: MAIL FROM: <address> (space after colon)",
			command:       []byte("MAIL FROM: <alice@example.com>\r\n"),
			expectAddr:    []byte("alice@example.com"),
			shouldMatch:   true,
			expectWarning: true,
			description:   "Space after colon violates RFC 5321 Section 3.3",
		},
		{
			name:          "RFC violation: MAIL FROM : <address> (spaces around colon)",
			command:       []byte("MAIL FROM : <alice@example.com>\r\n"),
			expectAddr:    []byte("alice@example.com"),
			shouldMatch:   true,
			expectWarning: true,
			description:   "Spaces before and after colon",
		},
		{
			name:          "RFC violation: MAIL  FROM:<address> (double space)",
			command:       []byte("MAIL  FROM:<alice@example.com>\r\n"),
			expectAddr:    []byte("alice@example.com"),
			shouldMatch:   true,
			expectWarning: true,
			description:   "Double space between MAIL and FROM",
		},
		{
			name:          "RFC violation: RCPT TO: <address> (space after colon)",
			command:       []byte("RCPT TO: <bob@example.com>\r\n"),
			expectAddr:    []byte("bob@example.com"),
			expectServer:  []byte("example.com"),
			shouldMatch:   true,
			expectWarning: true,
			description:   "Space after colon in RCPT TO",
		},

		// === RFC Violation: Email Address Local Part (Carrier Patterns) ===
		{
			name:          "Carrier pattern: consecutive dots",
			command:       []byte("MAIL FROM:<user..name@example.com>\r\n"),
			expectAddr:    []byte("user..name@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Two consecutive dots in local part",
		},
		{
			name:          "Carrier pattern: triple consecutive dots",
			command:       []byte("MAIL FROM:<user...name@example.com>\r\n"),
			expectAddr:    []byte("user...name@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Three consecutive dots in local part",
		},
		{
			name:          "Carrier pattern: dot before @",
			command:       []byte("MAIL FROM:<username.@example.com>\r\n"),
			expectAddr:    []byte("username.@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Dot immediately before @ symbol",
		},
		{
			name:          "Carrier pattern: hyphen at start",
			command:       []byte("MAIL FROM:<-username@example.com>\r\n"),
			expectAddr:    []byte("-username@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Hyphen at the start of local part",
		},
		{
			name:          "Carrier pattern: dot at start",
			command:       []byte("MAIL FROM:<.username@example.com>\r\n"),
			expectAddr:    []byte(".username@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Dot at the start of local part",
		},
		{
			name:          "Carrier pattern: consecutive hyphens",
			command:       []byte("MAIL FROM:<user--name@example.com>\r\n"),
			expectAddr:    []byte("user--name@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Two consecutive hyphens in local part",
		},
		{
			name:          "Carrier pattern: multiple violations",
			command:       []byte("MAIL FROM:<-user..name.@example.com>\r\n"),
			expectAddr:    []byte("-user..name.@example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Hyphen at start, consecutive dots, dot before @",
		},

		// === RCPT TO with Carrier Patterns ===
		{
			name:          "RCPT TO: carrier pattern consecutive dots",
			command:       []byte("RCPT TO:<user..name@example.com>\r\n"),
			expectAddr:    []byte("user..name@example.com"),
			expectServer:  []byte("example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Consecutive dots in RCPT TO",
		},
		{
			name:          "RCPT TO: carrier pattern hyphen at start",
			command:       []byte("RCPT TO:<-username@example.com>\r\n"),
			expectAddr:    []byte("-username@example.com"),
			expectServer:  []byte("example.com"),
			shouldMatch:   true,
			expectWarning: false,
			description:   "Hyphen at start in RCPT TO",
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

			// Determine command type and execute appropriate function
			isRCPT := len(tt.expectServer) > 0
			if isRCPT {
				// Test RCPT TO
				pipe.setReceiverMailAddressAndServerName(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.rMailAddr) {
						t.Errorf("%s: expected address %q, got %q", tt.description, tt.expectAddr, pipe.rMailAddr)
					}
					if string(tt.expectServer) != string(pipe.rServerName) {
						t.Errorf("%s: expected server %q, got %q", tt.description, tt.expectServer, pipe.rServerName)
					}
				} else {
					if len(pipe.rMailAddr) > 0 {
						t.Errorf("%s: expected no match, but got address %q", tt.description, pipe.rMailAddr)
					}
				}
			} else {
				// Test MAIL FROM
				pipe.setSenderMailAddress(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.sMailAddr) {
						t.Errorf("%s: expected address %q, got %q", tt.description, tt.expectAddr, pipe.sMailAddr)
					}
				} else {
					if len(pipe.sMailAddr) > 0 {
						t.Errorf("%s: expected no match, but got address %q", tt.description, pipe.sMailAddr)
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

			// Check if RFC violation warning was logged as expected
			if tt.expectWarning && !warningLogged {
				t.Errorf("%s: expected RFC violation warning to be logged, but none was found", tt.description)
			}
			if !tt.expectWarning && warningLogged {
				t.Errorf("%s: unexpected RFC violation warning logged", tt.description)
			}
		})
	}
}
