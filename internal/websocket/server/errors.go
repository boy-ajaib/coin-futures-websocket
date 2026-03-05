package server

import (
	"errors"

	"github.com/centrifugal/centrifuge"
)

var (
	// Internal errors for server logic
	ErrConnectionLimitReached = errors.New("connection limit reached for user")
	ErrAlreadySubscribed      = errors.New("already subscribed to channel")
	ErrNotSubscribed          = errors.New("not subscribed to channel")
	ErrChannelNotFound        = errors.New("channel not found")
	ErrInvalidMessage         = errors.New("invalid message format")
	ErrClientBufferFull       = errors.New("client send buffer full")
)

// Error codes for WebSocket communication.
// These codes are compatible with Centrifuge's disconnect and error codes.
// Centrifuge code ranges:
//
//	0-2999:     reserved for client-side and transport
//	3000-3499:  non-terminal, client should reconnect
//	3500-3999:  terminal, no auto-reconnect
//	4000-4499:  custom disconnects, reconnect (for library users)
//	4500-4999:  custom disconnects, terminal (for library users)
//	>=5000:     reserved by Centrifuge
const (
	// Client errors (4000-4499) - non-terminal, client should reconnect
	CodeBadRequest        = 4000 // Invalid request format
	CodeChannelNotFound   = 4001 // Channel not found or invalid format
	CodeAlreadySubscribed = 4002 // Already subscribed to channel
	CodeNotSubscribed     = 4003 // Not subscribed to channel
	CodeSubscriptionLimit = 4004 // Subscription limit exceeded

	// Authorization errors (4100-4199) - non-terminal
	CodeUnauthorized    = 4100 // Invalid or missing credentials
	CodeConnectionLimit = 4200 // Connection limit reached

	// Server errors (4500-4999) - terminal, no auto-reconnect
	CodeInternalError      = 4500 // Internal server error
	CodeServiceUnavailable = 4503 // Service unavailable (terminal)

	// Specific service unavailable codes
	CodeCfxUserResolution = 4501 // Failed to resolve CFX user ID (terminal)
	CodeUserPreference    = 4502 // Failed to fetch user preference (terminal)
)

// NewDisconnect creates a Disconnect from a custom error code.
func NewDisconnect(code uint32, reason string) centrifuge.Disconnect {
	return centrifuge.Disconnect{
		Code:   code,
		Reason: reason,
	}
}

// NewError creates an Error from a custom error code.
func NewError(code uint32, reason string) *centrifuge.Error {
	return &centrifuge.Error{
		Code:    code,
		Message: reason,
	}
}

// DisconnectReasons provides human-readable reasons for common disconnect scenarios.
var DisconnectReasons = disconnectReasons{}

type disconnectReasons struct{}

// Unauthorized returns the reason for unauthorized disconnect.
func (disconnectReasons) Unauthorized() string {
	return "unauthorized: invalid or missing credentials"
}

// ConnectionLimit returns the reason for connection limit disconnect.
func (disconnectReasons) ConnectionLimit() string {
	return "connection limit reached: too many connections for this user"
}

// ChannelNotFound returns the reason for channel not found disconnect.
func (disconnectReasons) ChannelNotFound() string {
	return "channel not found: invalid or unauthorized channel"
}

// ServiceUnavailable returns the reason for service unavailable disconnect.
func (disconnectReasons) ServiceUnavailable() string {
	return "service unavailable: please try again later"
}

// BadRequest returns the reason for bad request disconnect.
func (disconnectReasons) BadRequest() string {
	return "bad request: invalid request format"
}

// CfxUserResolutionError returns the reason for CFX user resolution failure.
func (disconnectReasons) CfxUserResolutionError() string {
	return "service unavailable: failed to resolve user identity"
}

// UserPreferenceError returns the reason for user preference fetch failure.
func (disconnectReasons) UserPreferenceError() string {
	return "service unavailable: failed to fetch user preferences"
}
