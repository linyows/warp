package warp

// FilterAction represents the action to take after filtering.
type FilterAction int

const (
	FilterRelay     FilterAction = iota // Relay as-is
	FilterReject                        // Reject with SMTP error response
	FilterAddHeader                     // Add/modify headers and relay
)

// BeforeRelayData contains the data available to filter hooks before relaying.
type BeforeRelayData struct {
	ConnID   string
	MailFrom []byte
	MailTo   []byte
	SenderIP string
	Helo     []byte
	// Message is the full message (headers + body) in SMTP DATA wire format.
	// It is dot-stuffed as received on the wire and does NOT include the
	// terminating "\r\n.\r\n" sequence.
	Message []byte
}

// FilterResult represents the result of a filter hook.
type FilterResult struct {
	Action FilterAction
	// Message is the modified message for FilterAddHeader. It MUST be in SMTP
	// DATA wire format (dot-stuffed) and MUST NOT contain the terminator
	// "\r\n.\r\n". The terminator is appended automatically after relay.
	Message []byte
	// Reply is the SMTP reply for FilterReject (e.g. "550 5.7.1 Spam detected").
	// Must be a single line without CR/LF characters; any CR/LF will be stripped.
	Reply string
}

// FilterHook extends Hook with a BeforeRelay method called during the DATA phase.
type FilterHook interface {
	Hook
	BeforeRelay(*BeforeRelayData) *FilterResult
}
