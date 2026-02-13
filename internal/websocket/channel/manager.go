package channel

import (
	"log/slog"
	"sync"
)

// Manager manages channel subscriptions and validation
type Manager struct {
	clientSubscriptions map[string]map[string]bool // clientID -> channel -> bool

	logger *slog.Logger
	mu     sync.RWMutex
}

// NewManager creates a new channel manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		clientSubscriptions: make(map[string]map[string]bool),
		logger:              logger,
	}
}

// ValidateSubscription validates the channel format
func (m *Manager) ValidateSubscription(clientID, channel string) (*ChannelInfo, error) {
	return ParseChannel(channel)
}

// TrackSubscription records a client subscription
func (m *Manager) TrackSubscription(clientID, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clientSubscriptions[clientID] == nil {
		m.clientSubscriptions[clientID] = make(map[string]bool)
	}
	m.clientSubscriptions[clientID][channel] = true

	m.logger.Debug("subscription tracked",
		"client_id", clientID,
		"channel", channel,
		"total", len(m.clientSubscriptions[clientID]))
}

// UntrackSubscription removes a client subscription record
func (m *Manager) UntrackSubscription(clientID, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subs, exists := m.clientSubscriptions[clientID]; exists {
		delete(subs, channel)
		if len(subs) == 0 {
			delete(m.clientSubscriptions, clientID)
		}
	}

	m.logger.Debug("subscription untracked",
		"client_id", clientID,
		"channel", channel)
}

// UntrackClient removes all subscription records for a client
func (m *Manager) UntrackClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clientSubscriptions, clientID)
	m.logger.Debug("client untracked", "client_id", clientID)
}

// GetClientSubscriptions returns all subscriptions for a client
func (m *Manager) GetClientSubscriptions(clientID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs := m.clientSubscriptions[clientID]
	if subs == nil {
		return nil
	}

	channels := make([]string, 0, len(subs))
	for ch := range subs {
		channels = append(channels, ch)
	}
	return channels
}

// IsSubscribed checks if a client is subscribed to a channel
func (m *Manager) IsSubscribed(clientID, channel string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if subs, exists := m.clientSubscriptions[clientID]; exists {
		return subs[channel]
	}
	return false
}
