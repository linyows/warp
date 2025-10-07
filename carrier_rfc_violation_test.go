package warp

import (
	"testing"
)

// TestCarrierRFCViolation tests RFC-violating email addresses patterns
// References:
// - https://www.docomo.ne.jp/service/docomo_mail/rfc_add/
// - https://www.sonoko.co.jp/user_data/oshirase10.php
//
// These patterns violate RFC 5321 but are actually used in production by some carriers.
func TestCarrierRFCViolation(t *testing.T) {
	tests := []struct {
		name        string
		command     []byte
		expectAddr  []byte
		shouldMatch bool
		description string
	}{
		// Carrier pattern: consecutive dots
		{
			name:        "RFC violation: consecutive dots",
			command:     []byte("MAIL FROM:<user..name@example.com>\r\n"),
			expectAddr:  []byte("user..name@example.com"),
			shouldMatch: true,
			description: "Two consecutive dots in local part",
		},
		{
			name:        "RFC violation: triple consecutive dots",
			command:     []byte("MAIL FROM:<user...name@example.com>\r\n"),
			expectAddr:  []byte("user...name@example.com"),
			shouldMatch: true,
			description: "Three consecutive dots in local part",
		},

		// Carrier pattern: dot before @
		{
			name:        "RFC violation: dot before @",
			command:     []byte("MAIL FROM:<username.@example.com>\r\n"),
			expectAddr:  []byte("username.@example.com"),
			shouldMatch: true,
			description: "Dot immediately before @ symbol",
		},

		// Carrier pattern: hyphen at start
		{
			name:        "RFC violation: hyphen at start",
			command:     []byte("MAIL FROM:<-username@example.com>\r\n"),
			expectAddr:  []byte("-username@example.com"),
			shouldMatch: true,
			description: "Hyphen at the start of local part",
		},

		// Dot at start
		{
			name:        "RFC violation: dot at start",
			command:     []byte("MAIL FROM:<.username@example.com>\r\n"),
			expectAddr:  []byte(".username@example.com"),
			shouldMatch: true,
			description: "Dot at the start of local part",
		},

		// RCPT TO cases
		{
			name:        "RCPT TO: RFC violation consecutive dots",
			command:     []byte("RCPT TO:<user..name@example.com>\r\n"),
			expectAddr:  []byte("user..name@example.com"),
			shouldMatch: true,
			description: "Consecutive dots in RCPT TO",
		},
		{
			name:        "RCPT TO: RFC violation hyphen at start",
			command:     []byte("RCPT TO:<-username@example.com>\r\n"),
			expectAddr:  []byte("-username@example.com"),
			shouldMatch: true,
			description: "Hyphen at start in RCPT TO",
		},

		// Consecutive hyphens
		{
			name:        "RFC violation: consecutive hyphens",
			command:     []byte("MAIL FROM:<user--name@example.com>\r\n"),
			expectAddr:  []byte("user--name@example.com"),
			shouldMatch: true,
			description: "Two consecutive hyphens in local part",
		},

		// Mixed violations
		{
			name:        "RFC violation: multiple issues",
			command:     []byte("MAIL FROM:<-user..name.@example.com>\r\n"),
			expectAddr:  []byte("-user..name.@example.com"),
			shouldMatch: true,
			description: "Hyphen at start, consecutive dots, dot before @",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipe := &Pipe{afterCommHook: func(b Data, to Direction) {}}

			if containsFold(tt.command, []byte("MAIL FROM")) {
				pipe.setSenderMailAddress(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.sMailAddr) {
						t.Errorf("%s: expected address %q, got %q", tt.description, tt.expectAddr, pipe.sMailAddr)
					} else {
						t.Logf("✓ Successfully extracted: %q (%s)", pipe.sMailAddr, tt.description)
					}
				} else {
					if len(pipe.sMailAddr) > 0 {
						t.Errorf("%s: expected no match, but got address %q", tt.description, pipe.sMailAddr)
					}
				}
			} else if containsFold(tt.command, []byte("RCPT TO")) {
				pipe.setReceiverMailAddressAndServerName(tt.command)
				if tt.shouldMatch {
					if string(tt.expectAddr) != string(pipe.rMailAddr) {
						t.Errorf("%s: expected address %q, got %q", tt.description, tt.expectAddr, pipe.rMailAddr)
					} else {
						t.Logf("✓ Successfully extracted: %q (%s)", pipe.rMailAddr, tt.description)
					}
				} else {
					if len(pipe.rMailAddr) > 0 {
						t.Errorf("%s: expected no match, but got address %q", tt.description, pipe.rMailAddr)
					}
				}
			}
		})
	}
}
