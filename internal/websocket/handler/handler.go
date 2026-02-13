package handler

import (
	"errors"
	"log/slog"

	"coin-futures-websocket/internal/websocket/channel"
	"coin-futures-websocket/internal/websocket/protocol"
	"coin-futures-websocket/internal/websocket/server"
)

// KafkaBroadcaster is an interface for broadcasting Kafka messages to WebSocket clients
type KafkaBroadcaster interface {
	RegisterSubscription(cfxUserID, ajaibID string)
	UnregisterSubscription(cfxUserID string)
}

// DefaultHandler handles WebSocket protocol messages
type DefaultHandler struct {
	hub              *server.Hub
	kafkaBroadcaster KafkaBroadcaster
	logger           *slog.Logger
}

// NewDefaultHandler creates a new default message handler
func NewDefaultHandler(hub *server.Hub, logger *slog.Logger) *DefaultHandler {
	return &DefaultHandler{
		hub:    hub,
		logger: logger,
	}
}

// SetKafkaBroadcaster sets the Kafka broadcaster for user subscription tracking
func (h *DefaultHandler) SetKafkaBroadcaster(broadcaster KafkaBroadcaster) {
	h.kafkaBroadcaster = broadcaster
}

// HandleMessage processes incoming WebSocket messages
func (h *DefaultHandler) HandleMessage(client *server.Client, message []byte) error {
	msg, err := protocol.ParseMessage(message)
	if err != nil {
		h.logger.Warn("failed to parse message",
			"client_id", client.ID(),
			"error", err)
		return client.SendMessage(protocol.NewErrorMessage("", protocol.CodeBadRequest, "invalid message format"))
	}

	h.logger.Debug("received message",
		"client_id", client.ID(),
		"type", msg.Type,
		"id", msg.ID)

	switch msg.Type {
	case protocol.TypePing:
		return h.handlePing(client, msg)
	case protocol.TypeSubscribe:
		return h.handleSubscribe(client, msg)
	case protocol.TypeUnsubscribe:
		return h.handleUnsubscribe(client, msg)
	default:
		h.logger.Warn("unknown message type",
			"client_id", client.ID(),
			"type", msg.Type)
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeBadRequest, "unknown message type"))
	}
}

// handlePing handles ping messages
func (h *DefaultHandler) handlePing(client *server.Client, msg *protocol.Message) error {
	return client.SendMessage(protocol.NewPongMessage(msg.ID))
}

// handleSubscribe handles channel subscription requests
func (h *DefaultHandler) handleSubscribe(client *server.Client, msg *protocol.Message) error {
	channelName := msg.Channel
	if channelName == "" {
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeBadRequest, "channel required"))
	}

	if h.hub.IsClientSubscribed(client, channelName) {
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeAlreadySubscribed, "already subscribed to channel"))
	}

	channelInfo, err := channel.ParseChannel(channelName)
	if err != nil {
		h.logger.Warn("subscription validation failed",
			"client_id", client.ID(),
			"channel", channelName,
			"error", err)
		code := protocol.CodeBadRequest
		switch {
		case errors.Is(err, channel.ErrInvalidChannelFormat),
			errors.Is(err, channel.ErrInvalidCFXUserID):
			code = protocol.CodeChannelNotFound
		}
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, code, err.Error()))
	}

	if client.AjaibID() != "" && client.AjaibID() != channelInfo.AjaibID {
		h.logger.Warn("subscription ajaib_id mismatch",
			"client_id", client.ID(),
			"client_ajaib_id", client.AjaibID(),
			"channel_ajaib_id", channelInfo.AjaibID,
			"channel", channelName)
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeChannelNotFound, "channel not found"))
	}

	if h.kafkaBroadcaster != nil && client.CfxUserID() != "" {
		h.kafkaBroadcaster.RegisterSubscription(client.CfxUserID(), client.AjaibID())
	}

	h.hub.SubscribeClient(client, channelName)

	h.logger.Info("client subscribed to channel",
		"client_id", client.ID(),
		"channel", channelName)

	return client.SendMessage(protocol.NewSubscribedMessage(msg.ID, channelName))
}

// handleUnsubscribe handles channel unsubscription requests
func (h *DefaultHandler) handleUnsubscribe(client *server.Client, msg *protocol.Message) error {
	channelName := msg.Channel
	if channelName == "" {
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeBadRequest, "channel required"))
	}

	if !h.hub.IsClientSubscribed(client, channelName) {
		return client.SendMessage(protocol.NewErrorMessage(msg.ID, protocol.CodeNotSubscribed, "not subscribed to channel"))
	}

	h.hub.UnsubscribeClient(client, channelName)

	if h.kafkaBroadcaster != nil && client.CfxUserID() != "" {
		h.kafkaBroadcaster.UnregisterSubscription(client.CfxUserID())
	}

	h.logger.Info("client unsubscribed from channel",
		"client_id", client.ID(),
		"channel", channelName)

	return client.SendMessage(protocol.NewUnsubscribedMessage(msg.ID, channelName))
}

// OnClientDisconnect should be called when a client disconnects to cleanup tracking
func (h *DefaultHandler) OnClientDisconnect(clientID, cfxUserID string) {
	if h.kafkaBroadcaster != nil && cfxUserID != "" {
		h.kafkaBroadcaster.UnregisterSubscription(cfxUserID)
	}
}

// Stop stops the handler and cleans up resources
func (h *DefaultHandler) Stop() {
}
