package subscription

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/centrifugal/centrifuge-go"
)

// Manager manages channel subscriptions and their handlers
type Manager struct {
	client   *centrifuge.Client
	logger   *slog.Logger
	handlers map[string]MessageHandler
	subs     map[string]*centrifuge.Subscription
	mu       sync.RWMutex
}

// NewManager creates a new subscription manager
func NewManager(client *centrifuge.Client, logger *slog.Logger) *Manager {
	return &Manager{
		client:   client,
		logger:   logger,
		handlers: make(map[string]MessageHandler),
		subs:     make(map[string]*centrifuge.Subscription),
	}
}

// RegisterHandler registers a message handler for a channel
func (m *Manager) RegisterHandler(handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	channel := handler.Channel()
	m.handlers[channel] = handler

	m.logger.Info("message handler registered", "channel", channel)
	return nil
}

// Subscribe subscribes to a channel using its registered handler
func (m *Manager) Subscribe(channel string) error {
	m.mu.RLock()
	handler, exists := m.handlers[channel]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for channel: %s", channel)
	}

	sub, err := m.client.NewSubscription(channel, centrifuge.SubscriptionConfig{
		Recoverable: true,
		JoinLeave:   false,
	})
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	// Handle subscription events
	sub.OnSubscribing(func(e centrifuge.SubscribingEvent) {
		m.logger.Debug("subscribing to channel", "channel", channel, "code", e.Code, "reason", e.Reason)
	})

	sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		m.logger.Info("subscribed to channel", "channel", channel, "recoverable", e.Recoverable)
	})

	sub.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
		m.logger.Warn("unsubscribed from channel", "channel", channel, "code", e.Code, "reason", e.Reason)
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		m.logger.Error("subscription error", "channel", channel, "error", e.Error.Error())
	})

	// Handle publications
	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		if err := handler.Handle(e.Data); err != nil {
			m.logger.Error("handler error", "channel", channel, "error", err)
		}
	})

	if err := sub.Subscribe(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	m.mu.Lock()
	m.subs[channel] = sub
	m.mu.Unlock()

	return nil
}

// Unsubscribe unsubscribes from a channel
func (m *Manager) Unsubscribe(channel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subs[channel]
	if !exists {
		return fmt.Errorf("not subscribed to channel: %s", channel)
	}

	sub.Unsubscribe()
	delete(m.subs, channel)

	m.logger.Info("unsubscribed from channel", "channel", channel)
	return nil
}

// UnsubscribeAll unsubscribes from all channels
func (m *Manager) UnsubscribeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for channel, sub := range m.subs {
		m.logger.Debug("unsubscribing from channel", "channel", channel)
		sub.Unsubscribe()
	}
	m.subs = make(map[string]*centrifuge.Subscription)
}

// IsSubscribed returns whether the manager is subscribed to a channel
func (m *Manager) IsSubscribed(channel string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.subs[channel]
	return exists
}

// GetChannels returns a list of all subscribed channels
func (m *Manager) GetChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channels := make([]string, 0, len(m.subs))
	for channel := range m.subs {
		channels = append(channels, channel)
	}
	return channels
}
