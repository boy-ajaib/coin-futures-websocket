package subscription

// MessageHandler defines the interface for handling channel messages
type MessageHandler interface {
	// Handle processes a message from the channel
	Handle(data []byte) error

	// Channel returns the channel name this handler is for
	Channel() string
}
