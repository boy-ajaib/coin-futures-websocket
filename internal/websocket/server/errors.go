package server

import "errors"

var (
	ErrConnectionLimitReached = errors.New("connection limit reached for user")
	ErrAlreadySubscribed      = errors.New("already subscribed to channel")
	ErrNotSubscribed          = errors.New("not subscribed to channel")
	ErrChannelNotFound        = errors.New("channel not found")
	ErrInvalidMessage         = errors.New("invalid message format")
	ErrClientBufferFull       = errors.New("client send buffer full")
)
