package server

import (
	"encoding/json"
	"log/slog"
	"sync"

	"coin-futures-websocket/internal/websocket/protocol"
)

// Hub maintains the set of active clients and broadcasts messages to subscribed channels
type Hub struct {
	clients               map[*Client]bool
	userConnections       map[string]int
	channels              map[string]map[*Client]bool
	register              chan *Client
	unregister            chan *Client
	broadcast             chan *ChannelMessage
	maxConnectionsPerUser int
	logger                *slog.Logger
	mu                    sync.RWMutex
}

// ChannelMessage represents a message to broadcast to a channel
type ChannelMessage struct {
	Channel string
	Data    json.RawMessage
}

// NewHub creates a new Hub instance
func NewHub(maxConnectionsPerUser int, logger *slog.Logger) *Hub {
	return &Hub{
		clients:               make(map[*Client]bool),
		userConnections:       make(map[string]int),
		channels:              make(map[string]map[*Client]bool),
		register:              make(chan *Client),
		unregister:            make(chan *Client),
		broadcast:             make(chan *ChannelMessage, 256),
		maxConnectionsPerUser: maxConnectionsPerUser,
		logger:                logger,
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastToChannel(message)
		}
	}
}

// registerClient adds a client to the hub, enforcing per-user connection limits
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ajaibID := client.AjaibID()

	// Check connection limit for this user
	if ajaibID != "" && h.maxConnectionsPerUser > 0 {
		currentCount := h.userConnections[ajaibID]
		if currentCount >= h.maxConnectionsPerUser {
			h.logger.Warn("connection limit reached for user",
				"ajaib_id", ajaibID,
				"current_count", currentCount,
				"max_connections", h.maxConnectionsPerUser)
			return
		}
		h.userConnections[ajaibID] = currentCount + 1
	}

	h.clients[client] = true
	h.logger.Debug("client registered",
		"client_id", client.ID(),
		"ajaib_id", ajaibID,
		"user_connections", h.userConnections[ajaibID])
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)

	ajaibID := client.AjaibID()
	if ajaibID != "" {
		if currentCount, exists := h.userConnections[ajaibID]; exists {
			if currentCount > 0 {
				h.userConnections[ajaibID] = currentCount - 1
			}
			if h.userConnections[ajaibID] == 0 {
				delete(h.userConnections, ajaibID)
			}
		}
	}

	// Remove from all channels
	for channel := range client.subscriptions {
		clients := h.channels[channel]
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	close(client.send)

	h.logger.Debug("client unregistered",
		"client_id", client.ID(),
		"ajaib_id", ajaibID)
}

// broadcastToChannel sends a message to all clients subscribed to a channel
func (h *Hub) broadcastToChannel(message *ChannelMessage) {
	h.mu.RLock()
	clients, exists := h.channels[message.Channel]
	if !exists {
		h.mu.RUnlock()
		return
	}

	// Create a copy of clients to avoid holding lock during send
	clientList := make([]*Client, 0, len(clients))
	for client := range clients {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	// TODO: Uncomment for debugging purpose
	// message.Channel = "user:130010505:margin"

	msg := protocol.NewPublicationMessage(message.Channel, message.Data)
	data, err := msg.Encode()
	if err != nil {
		h.logger.Error("failed to encode broadcast message", "error", err)
		return
	}

	for _, client := range clientList {
		select {
		case client.send <- data:
			// Message sent
		default:
			h.logger.Warn("client send buffer full, dropping message",
				"client_id", client.id,
				"channel", message.Channel)
		}
	}
}

// SubscribeClient subscribes a client to a channel
func (h *Hub) SubscribeClient(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}

	h.channels[channel][client] = true
	client.subscriptions[channel] = true

	h.logger.Debug("client subscribed to channel",
		"client_id", client.ID(),
		"channel", channel,
		"total_subscribers", len(h.channels[channel]))
}

// UnsubscribeClient unsubscribes a client from a channel
func (h *Hub) UnsubscribeClient(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.channels[channel], client)
	delete(client.subscriptions, channel)

	h.logger.Debug("client unsubscribed from channel",
		"client_id", client.ID(),
		"channel", channel,
		"total_subscribers", len(h.channels[channel]))
}

// GetClientCount returns the total number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetUserConnectionCount returns the number of active connections for a specific user
func (h *Hub) GetUserConnectionCount(ajaibID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userConnections[ajaibID]
}

// CanUserConnect checks if a user can establish a new connection based on the limit
func (h *Hub) CanUserConnect(ajaibID string) bool {
	if ajaibID == "" || h.maxConnectionsPerUser <= 0 {
		return true
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userConnections[ajaibID] < h.maxConnectionsPerUser
}

// GetChannelSubscriberCount returns the number of subscribers for a channel
func (h *Hub) GetChannelSubscriberCount(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, exists := h.channels[channel]; exists {
		return len(clients)
	}
	return 0
}

// Broadcast sends a message to all subscribers of a specific channel
func (h *Hub) Broadcast(channel string, data json.RawMessage) {
	h.broadcast <- &ChannelMessage{
		Channel: channel,
		Data:    data,
	}
}
