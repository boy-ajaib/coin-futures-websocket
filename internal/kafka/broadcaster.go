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
	TransformUserMargin(data []byte, cfxUserID string, quotePreference string) ([]byte, error)
	TransformUserPosition(data []byte, cfxUserID string, quotePreference string) ([]byte, error)
}

// subscribedUser holds the details of a user with an active WebSocket subscription.
type subscribedUser struct {
	ajaibID         string
	quotePreference string
}

// Broadcaster handles broadcasting Kafka messages to WebSocket clients
type Broadcaster struct {
	hub         *server.Hub
	transformer Transformer
	logger      *slog.Logger
	activeUsers map[string]subscribedUser // Map cfx_user_id -> subscribedUser
	mu          sync.RWMutex
}

// NewBroadcaster creates a new Kafka broadcaster
func NewBroadcaster(hub *server.Hub, transformer Transformer, logger *slog.Logger) *Broadcaster {
	return &Broadcaster{
		hub:         hub,
		transformer: transformer,
		logger:      logger,
		activeUsers: make(map[string]subscribedUser),
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
	user, ok := b.getSubscribedUser(cfxUserID)
	if !ok {
		// No active subscribers, skip broadcast
		return nil
	}

	var dataToBroadcast []byte = data
	if b.transformer != nil {
		transformedData, err := b.transformer.TransformUserMargin(data, cfxUserID, user.quotePreference)
		if err != nil {
			b.logger.Error("failed to transform user margin", "error", err)
			return nil
		} else {
			dataToBroadcast = transformedData
		}
	}

	channel := "user:" + user.ajaibID + ":" + types.ChannelMarginSuffix
	b.hub.Broadcast(channel, dataToBroadcast)

	b.logger.Debug("broadcasted user margin",
		"cfx_user_id", cfxUserID,
		"ajaib_id", user.ajaibID,
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
	user, ok := b.getSubscribedUser(cfxUserID)
	if !ok {
		// No active subscribers, skip broadcast
		return nil
	}

	var dataToBroadcast []byte = data
	if b.transformer != nil {
		transformedData, err := b.transformer.TransformUserPosition(data, cfxUserID, user.quotePreference)
		if err != nil {
			b.logger.Error("failed to transform user position", "error", err)
			return nil
		} else {
			dataToBroadcast = transformedData
		}
	}

	channel := "user:" + user.ajaibID + ":" + types.ChannelPositionSuffix
	b.hub.Broadcast(channel, dataToBroadcast)

	b.logger.Debug("broadcasted user position",
		"cfx_user_id", cfxUserID,
		"ajaib_id", user.ajaibID,
		"channel", channel,
		"symbol", position.Symbol,
		"size", position.Size)

	return nil
}

// RegisterSubscription registers that a WebSocket client has subscribed to a user channel
func (b *Broadcaster) RegisterSubscription(cfxUserID, ajaibID, quotePreference string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeUsers[cfxUserID] = subscribedUser{ajaibID: ajaibID, quotePreference: quotePreference}
	b.logger.Debug("registered kafka subscription", "cfx_user_id", cfxUserID, "ajaib_id", ajaibID, "quote_preference", quotePreference)
}

// UnregisterSubscription removes a WebSocket client's subscription
func (b *Broadcaster) UnregisterSubscription(cfxUserID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.activeUsers, cfxUserID)
	b.logger.Debug("unregistered kafka subscription", "cfx_user_id", cfxUserID)
}

// getSubscribedUser returns the subscribed user for the given cfx_user_id, or false if not found
func (b *Broadcaster) getSubscribedUser(cfxUserID string) (subscribedUser, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	user, ok := b.activeUsers[cfxUserID]
	return user, ok
}
