package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// HeartbeatMessage represents a heartbeat message from CFX
type HeartbeatMessage struct {
	Timestamp int64         `json:"timestamp"`
	Method    string        `json:"method"`
	Channel   string        `json:"channel"`
	Code      int           `json:"code"`
	Data      HeartbeatData `json:"data"`
}

// HeartbeatData is the inner data of a heartbeat message
type HeartbeatData struct {
	Alive bool `json:"alive"`
}

// HeartbeatHandler handles heartbeat channel messages
type HeartbeatHandler struct {
	logger  *slog.Logger
	onMsg   func(HeartbeatMessage)
	mu      sync.RWMutex
	channel string
}

// NewHeartbeatHandler creates a new heartbeat handler
func NewHeartbeatHandler(logger *slog.Logger) *HeartbeatHandler {
	return &HeartbeatHandler{
		logger:  logger,
		channel: "heartbeat",
	}
}

// Channel returns the channel name this handler is for
func (h *HeartbeatHandler) Channel() string {
	return h.channel
}

// Handle processes a heartbeat message
func (h *HeartbeatHandler) Handle(data []byte) error {
	var msg HeartbeatMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal heartbeat message: %w", err)
	}

	h.logger.Info("heartbeat received",
		"timestamp", msg.Timestamp,
		"alive", msg.Data.Alive)

	// Call registered callback if any
	h.mu.RLock()
	callback := h.onMsg
	h.mu.RUnlock()

	if callback != nil {
		callback(msg)
	}

	return nil
}

// OnMessage registers a callback for heartbeat messages
func (h *HeartbeatHandler) OnMessage(f func(HeartbeatMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMsg = f
}
