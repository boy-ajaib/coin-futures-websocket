package protocol

import (
	"encoding/json"
	"time"
)

// Message types for client-server communication
const (
	// Client -> Server commands
	TypeSubscribe   = "subscribe"
	TypeUnsubscribe = "unsubscribe"
	TypePing        = "ping"

	// Server -> Client responses
	TypeConnected    = "connected"
	TypeSubscribed   = "subscribed"
	TypeUnsubscribed = "unsubscribed"
	TypePong         = "pong"
	TypePublication  = "publication"
	TypeError        = "error"
	TypeDisconnect   = "disconnect"
)

// Error codes for WebSocket communication
const (
	CodeBadRequest        = 4000
	CodeChannelNotFound   = 4001
	CodeAlreadySubscribed = 4002
	CodeNotSubscribed     = 4003
	CodeSubscriptionLimit = 4004
	CodeUnauthorized      = 4100
	CodeConnectionLimit   = 4200
	CodeInternalError     = 4500
)

// Message represents a WebSocket protocol message
type Message struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Success   bool            `json:"success,omitempty"`
	Code      int             `json:"code,omitempty"`
	Message   string          `json:"message,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// SubscribeMessage is sent by client to subscribe to a channel
type SubscribeMessage struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

// UnsubscribeMessage is sent by client to unsubscribe from a channel
type UnsubscribeMessage struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

// NewSubscribedMessage creates a subscribed response message
func NewSubscribedMessage(requestID, channel string) *Message {
	return &Message{
		ID:        requestID,
		Type:      TypeSubscribed,
		Channel:   channel,
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewUnsubscribedMessage creates an unsubscribed response message
func NewUnsubscribedMessage(requestID, channel string) *Message {
	return &Message{
		ID:        requestID,
		Type:      TypeUnsubscribed,
		Channel:   channel,
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewPublicationMessage creates a publication message for channel data
func NewPublicationMessage(channel string, data json.RawMessage) *Message {
	return &Message{
		Type:      TypePublication,
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewPongMessage creates a pong response message
func NewPongMessage(requestID string) *Message {
	return &Message{
		ID:        requestID,
		Type:      TypePong,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewConnectedMessage creates a connected greeting message sent after successful WebSocket upgrade
func NewConnectedMessage(clientID, ajaibID string) *Message {
	dataPayload := map[string]string{
		"client_id": clientID,
		"ajaib_id":  ajaibID,
	}
	dataBytes, _ := json.Marshal(dataPayload)

	return &Message{
		Type:      TypeConnected,
		Data:      json.RawMessage(dataBytes),
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewErrorMessage creates an error response message
func NewErrorMessage(requestID string, code int, message string) *Message {
	return &Message{
		ID:        requestID,
		Type:      TypeError,
		Code:      code,
		Message:   message,
		Success:   false,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewDisconnectMessage creates a disconnect message
func NewDisconnectMessage(code int, reason string) *Message {
	return &Message{
		Type:      TypeDisconnect,
		Code:      code,
		Message:   reason,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ParseMessage parses a raw JSON message into a Message struct
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// Encode encodes a message to JSON bytes
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}
