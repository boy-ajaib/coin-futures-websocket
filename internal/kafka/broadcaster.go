package kafka

import (
	"encoding/json"
	"log/slog"
	"sync"

	"coin-futures-websocket/internal/types"
	"coin-futures-websocket/internal/websocket/server"
)

// Transformer defines the interface for transforming Kafka message data
type Transformer interface {
	TransformUserMargin(data []byte, cfxUserID string) ([]byte, error)
	TransformUserPosition(data []byte, cfxUserID string) ([]byte, error)
}

// Broadcaster handles broadcasting Kafka messages to WebSocket clients
type Broadcaster struct {
	hub         *server.Hub
	transformer Transformer
	logger      *slog.Logger
	activeUsers map[string]string // Map cfx_user_id -> ajaib_id
	mu          sync.RWMutex
}

// NewBroadcaster creates a new Kafka broadcaster
func NewBroadcaster(hub *server.Hub, transformer Transformer, logger *slog.Logger) *Broadcaster {
	return &Broadcaster{
		hub:         hub,
		transformer: transformer,
		logger:      logger,
		activeUsers: make(map[string]string),
	}
}

// HandleMessage is the Kafka message handler that routes messages to WebSocket clients
func (b *Broadcaster) HandleMessage(topic string, key []byte, value []byte) error {
	b.logger.Debug("kafka message received",
		"topic", topic,
		"key", string(key),
		"value", json.RawMessage(value))

	switch topic {
	case types.TopicUserMargin:
		return b.handleUserMargin(value)
	case types.TopicUserPosition:
		return b.handleUserPosition(value)
	default:
		b.logger.Warn("unknown kafka topic", "topic", topic)
		return nil
	}
}

// handleUserMargin processes UserMargin messages and broadcasts to relevant WebSocket clients
func (b *Broadcaster) handleUserMargin(data []byte) error {
	var margin types.UserMargin
	if err := json.Unmarshal(data, &margin); err != nil {
		b.logger.Error("failed to unmarshal UserMargin", "error", err)
		return err
	}

	b.logger.Debug("received user margin", "margin", margin)

	cfxUserID := margin.GetCFXUserID()
	ajaibID := b.getAjaibID(cfxUserID)
	if ajaibID == "" {
		// No active subscribers, skip broadcast
		return nil
	}

	var dataToBroadcast []byte = data
	if b.transformer != nil {
		transformedData, err := b.transformer.TransformUserMargin(data, cfxUserID)
		if err != nil {
			b.logger.Error("failed to transform user margin", "error", err)
			return nil
		} else {
			dataToBroadcast = transformedData
		}
	}

	channel := "user:" + ajaibID + ":" + types.ChannelMarginSuffix
	b.hub.Broadcast(channel, dataToBroadcast)

	b.logger.Debug("broadcasted user margin",
		"cfx_user_id", cfxUserID,
		"ajaib_id", ajaibID,
		"channel", channel,
		"asset", margin.Asset,
		"margin_balance", margin.MarginBalance)

	return nil
}

// handleUserPosition processes UserPosition messages and broadcasts to relevant WebSocket clients
func (b *Broadcaster) handleUserPosition(data []byte) error {
	var position types.UserPosition
	if err := json.Unmarshal(data, &position); err != nil {
		b.logger.Error("failed to unmarshal UserPosition", "error", err)
		return err
	}

	b.logger.Debug("received user position", "position", position)

	cfxUserID := position.GetCFXUserID()
	ajaibID := b.getAjaibID(cfxUserID)
	if ajaibID == "" {
		// No active subscribers, skip broadcast
		return nil
	}

	var dataToBroadcast []byte = data
	if b.transformer != nil {
		transformedData, err := b.transformer.TransformUserPosition(data, cfxUserID)
		if err != nil {
			b.logger.Error("failed to transform user position", "error", err)
			return nil
		} else {
			dataToBroadcast = transformedData
		}
	}

	channel := "user:" + ajaibID + ":" + types.ChannelPositionSuffix
	b.hub.Broadcast(channel, dataToBroadcast)

	b.logger.Debug("broadcasted user position",
		"cfx_user_id", cfxUserID,
		"ajaib_id", ajaibID,
		"channel", channel,
		"symbol", position.Symbol,
		"size", position.Size)

	return nil
}

// RegisterSubscription registers that a WebSocket client has subscribed to a user channel
func (b *Broadcaster) RegisterSubscription(cfxUserID, ajaibID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeUsers[cfxUserID] = ajaibID
	b.logger.Debug("registered kafka subscription", "cfx_user_id", cfxUserID, "ajaib_id", ajaibID)
}

// UnregisterSubscription removes a WebSocket client's subscription
func (b *Broadcaster) UnregisterSubscription(cfxUserID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.activeUsers, cfxUserID)
	b.logger.Debug("unregistered kafka subscription", "cfx_user_id", cfxUserID)
}

// getAjaibID returns the ajaib_id mapped to the given cfx_user_id, or empty string if not found
func (b *Broadcaster) getAjaibID(cfxUserID string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeUsers[cfxUserID]
}

// GetActiveUserCount returns the number of active user subscriptions
func (b *Broadcaster) GetActiveUserCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.activeUsers)
}

// GetActiveUsers returns a list of all active user IDs
func (b *Broadcaster) GetActiveUsers() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	users := make([]string, 0, len(b.activeUsers))
	for userID := range b.activeUsers {
		users = append(users, userID)
	}
	return users
}

// ClearAll removes all active subscriptions (useful for testing or shutdown)
func (b *Broadcaster) ClearAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeUsers = make(map[string]string)
}
