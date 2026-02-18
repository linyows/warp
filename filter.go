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
	Message  []byte // Full message (headers + body)
}

// FilterResult represents the result of a filter hook.
type FilterResult struct {
	Action  FilterAction
	Message []byte // Modified message for FilterAddHeader
	Reply   string // SMTP reply for FilterReject (e.g. "550 5.7.1 Spam detected")
}

// FilterHook extends Hook with a BeforeRelay method called during the DATA phase.
type FilterHook interface {
	Hook
	BeforeRelay(*BeforeRelayData) *FilterResult
}
