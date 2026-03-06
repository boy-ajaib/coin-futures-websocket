package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"coin-futures-websocket/internal/auth"
	"coin-futures-websocket/internal/websocket/channel"

	"github.com/centrifugal/centrifuge"
)

// SetupHandlers configures all Centrifuge event handlers
func (s *CentrifugeServer) SetupHandlers() {
	// Connecting handler - called when client tries to connect (before connection is established)
	s.node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return s.handleConnect(ctx, e)
	})

	// Connect handler - called when client connects and is ready to communicate
	// We set up per-client handlers here.
	s.node.OnConnect(func(client *centrifuge.Client) {
		// Track successful connection in metrics
		if s.metrics != nil {
			s.metrics.RecordConnection(s.config.NodeName)
		}
		s.setupClientHandlers(client)
	})

	s.logger.Info("centrifuge handlers configured")
}

// handleConnect handles client connection requests with JWT authentication
func (s *CentrifugeServer) handleConnect(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
	reply := centrifuge.ConnectReply{}

	// Extract JWT from the token field in ConnectEvent
	// In Centrifuge, clients typically send a connection token in the Connect command
	token := e.Token

	// If no token in ConnectEvent, try to extract from HTTP headers
	// This handles backward compatibility with existing clients that use
	// X-Socket-Authorization header
	if token == "" {
		var err error
		token, err = s.extractTokenFromContext(ctx, e)
		if err != nil {
			s.logger.Warn("unauthorized, failed to extract JWT",
				"client_id", e.ClientID,
				"error", err)
			return reply, NewError(CodeUnauthorized, DisconnectReasons.Unauthorized())
		}
	}

	ajaibID, err := s.parseAjaibIDFromToken(token)
	if err != nil {
		s.logger.Warn("unauthorized, failed to parse ajaib_id from token",
			"client_id", e.ClientID,
			"error", err)
		return reply, NewError(CodeUnauthorized, DisconnectReasons.Unauthorized())
	}

	// Resolve CFX user ID
	cfxUserID, err := s.resolveCfxUserID(ctx, ajaibID)
	if err != nil {
		s.logger.Error("failed to resolve cfx user id",
			"client_id", e.ClientID,
			"ajaib_id", ajaibID,
			"error", err)
		return reply, NewError(CodeCfxUserResolution, DisconnectReasons.CfxUserResolutionError())
	}

	// Fetch user quote preference
	quotePreference, err := s.resolveQuotePreference(ctx, ajaibID)
	if err != nil {
		s.logger.Error("failed to fetch user quote preference",
			"client_id", e.ClientID,
			"ajaib_id", ajaibID,
			"error", err)
		return reply, NewError(CodeUserPreference, DisconnectReasons.UserPreferenceError())
	}

	// Create connection info with user data
	connInfo := ClientInfo{
		AjaibID:         ajaibID,
		CfxUserID:       cfxUserID,
		QuotePreference: quotePreference,
		ConnectedAt:     time.Now().UnixMilli(),
	}
	infoData, _ := json.Marshal(connInfo)

	// Create connection credentials
	reply.Credentials = &centrifuge.Credentials{
		UserID: ajaibID,
		Info:   infoData,
	}

	s.logger.Info("client connected via centrifuge",
		"client_id", e.ClientID,
		"ajaib_id", ajaibID,
		"cfx_user_id", cfxUserID,
		"quote_preference", quotePreference)

	return reply, nil
}

// setupClientHandlers configures handlers for an individual client connection
func (s *CentrifugeServer) setupClientHandlers(client *centrifuge.Client) {
	// Refresh handler - for token expiration
	client.OnRefresh(func(e centrifuge.RefreshEvent, callback centrifuge.RefreshCallback) {
		s.handleRefresh(e, callback)
	})

	// Subscribe handler - for channel subscription validation
	client.OnSubscribe(func(e centrifuge.SubscribeEvent, callback centrifuge.SubscribeCallback) {
		s.handleSubscribe(client, e, callback)
	})

	// Publish handler - for client publish validation
	client.OnPublish(func(e centrifuge.PublishEvent, callback centrifuge.PublishCallback) {
		s.handlePublish(e, callback)
	})

	// RPC handler - for future extensibility
	client.OnRPC(func(e centrifuge.RPCEvent, callback centrifuge.RPCCallback) {
		s.handleRPC(e, callback)
	})

	// Disconnect handler - for cleanup
	client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
		s.handleDisconnect(client, e)
	})
}

// handleRefresh handles client token refresh requests
func (s *CentrifugeServer) handleRefresh(e centrifuge.RefreshEvent, callback centrifuge.RefreshCallback) {
	// For now, we don't have token expiration, so just allow refresh without changes
	reply := centrifuge.RefreshReply{
		ExpireAt: 0, // No expiration
	}
	callback(reply, nil)
}

// handleSubscribe handles channel subscription requests
func (s *CentrifugeServer) handleSubscribe(client *centrifuge.Client, e centrifuge.SubscribeEvent, callback centrifuge.SubscribeCallback) {
	reply := centrifuge.SubscribeReply{}

	// Parse and validate channel format
	channelInfo, err := channel.ParseChannel(e.Channel)
	if err != nil {
		s.logger.Warn("subscription validation failed",
			"client_id", client.ID(),
			"channel", e.Channel,
			"error", err)
		callback(reply, NewError(CodeChannelNotFound, err.Error()))
		return
	}

	// Get user info from client credentials to validate channel ownership
	clientInfo := s.getClientInfo(client)
	if clientInfo != nil && clientInfo.AjaibID != "" {
		// Verify user can only subscribe to their own channels
		if clientInfo.AjaibID != channelInfo.AjaibID {
			s.logger.Warn("subscription ajaib_id mismatch",
				"client_id", client.ID(),
				"client_ajaib_id", clientInfo.AjaibID,
				"channel_ajaib_id", channelInfo.AjaibID,
				"channel", e.Channel)
			callback(reply, NewError(CodeChannelNotFound, DisconnectReasons.ChannelNotFound()))
			return
		}
	}

	s.logger.Info("client subscribed to channel",
		"client_id", client.ID(),
		"channel", e.Channel,
		"ajaib_id", channelInfo.AjaibID)

	// Track subscription in metrics
	if s.metrics != nil {
		s.metrics.RecordSubscription(s.config.NodeName, e.Channel)
	}

	// Register subscription with Kafka broadcaster
	if s.broadcaster != nil && clientInfo != nil && clientInfo.CfxUserID != "" {
		s.broadcaster.RegisterSubscription(clientInfo.CfxUserID, clientInfo.AjaibID, clientInfo.QuotePreference)
	}

	callback(reply, nil)
}

// handlePublish handles client publish requests
func (s *CentrifugeServer) handlePublish(e centrifuge.PublishEvent, callback centrifuge.PublishCallback) {
	reply := centrifuge.PublishReply{}

	// For now, clients are not allowed to publish to channels
	// All publications come from the Kafka broadcaster
	callback(reply, NewError(CodeBadRequest, "client publishing not allowed"))
}

// handleRPC handles client RPC requests
func (s *CentrifugeServer) handleRPC(e centrifuge.RPCEvent, callback centrifuge.RPCCallback) {
	reply := centrifuge.RPCReply{}

	// For now, RPC is not implemented
	// This can be used for future extensibility (e.g., querying state)
	callback(reply, NewError(CodeBadRequest, "RPC not implemented"))
}

// handleDisconnect handles client disconnection
func (s *CentrifugeServer) handleDisconnect(client *centrifuge.Client, e centrifuge.DisconnectEvent) {
	// Track disconnection in metrics
	if s.metrics != nil {
		s.metrics.RecordDisconnection(s.config.NodeName)
	}

	clientInfo := s.getClientInfo(client)
	if clientInfo != nil {
		s.logger.Info("client disconnected",
			"client_id", client.ID(),
			"ajaib_id", clientInfo.AjaibID,
			"user_id", client.UserID(),
			"disconnect_code", e.Code,
			"disconnect_reason", e.Reason)

		// Unregister subscription with Kafka broadcaster
		if s.broadcaster != nil && clientInfo.CfxUserID != "" {
			s.broadcaster.UnregisterSubscription(clientInfo.CfxUserID)
		}
	} else {
		s.logger.Info("client disconnected",
			"client_id", client.ID(),
			"user_id", client.UserID(),
			"disconnect_code", e.Code,
			"disconnect_reason", e.Reason)
	}
}

// getClientInfo extracts connection info from client
func (s *CentrifugeServer) getClientInfo(client *centrifuge.Client) *ClientInfo {
	info := client.Info()
	if len(info) == 0 {
		return nil
	}

	var clientInfo ClientInfo
	if err := json.Unmarshal(info, &clientInfo); err != nil {
		return nil
	}
	return &clientInfo
}

// parseAjaibIDFromToken extracts ajaib_id from a JWT token using the auth package
func (s *CentrifugeServer) parseAjaibIDFromToken(token string) (string, error) {
	parser := auth.NewParser()
	return parser.ParseSubject(token)
}

// extractTokenFromContext extracts JWT token from context or HTTP headers
func (s *CentrifugeServer) extractTokenFromContext(ctx context.Context, e centrifuge.ConnectEvent) (string, error) {
	// First try to get from context (set by middleware)
	if token, ok := auth.TokenFrom(ctx); ok {
		return token, nil
	}

	// Fallback: try to extract from HTTP headers in the ConnectEvent
	// This is a workaround since Centrifuge doesn't expose the raw HTTP request
	// in ConnectEvent. In production, clients should use the Centrifuge token flow.
	return "", fmt.Errorf("no token provided and HTTP headers not accessible in ConnectEvent")
}

// resolveCfxUserID maps an Ajaib ID string to a CFX user ID via the configured mapper
func (s *CentrifugeServer) resolveCfxUserID(ctx context.Context, ajaibID string) (string, error) {
	if ajaibID == "" || s.cfxUserMapper == nil {
		return "", fmt.Errorf("ajaib_id is empty or cfx user mapper is not configured")
	}

	id, err := strconv.ParseInt(ajaibID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid ajaib_id format: %w", err)
	}

	cfxUserID, err := s.cfxUserMapper.GetCfxUserID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ajaib_id to cfx_user_id: %w", err)
	}

	return cfxUserID, nil
}

// resolveQuotePreference fetches the user's futures quote preference via the configured provider
func (s *CentrifugeServer) resolveQuotePreference(ctx context.Context, ajaibID string) (string, error) {
	if ajaibID == "" || s.userPrefProvider == nil {
		return "", nil
	}

	pref, err := s.userPrefProvider.GetQuotePreference(ctx, ajaibID)
	if err != nil {
		return "", err
	}

	return pref, nil
}

// ClientInfo holds user connection metadata
// This data is stored in the connection info and accessible from all client handlers
type ClientInfo struct {
	AjaibID         string `json:"ajaib_id"`
	CfxUserID       string `json:"cfx_user_id,omitempty"`
	QuotePreference string `json:"quote_preference"`
	ConnectedAt     int64  `json:"connected_at"`
}

// GetAjaibID returns the Ajaib user ID
func (ci *ClientInfo) GetAjaibID() string {
	return ci.AjaibID
}

// GetCfxUserID returns the CFX user ID
func (ci *ClientInfo) GetCfxUserID() string {
	return ci.CfxUserID
}

// GetQuotePreference returns the user's quote preference
func (ci *ClientInfo) GetQuotePreference() string {
	return ci.QuotePreference
}

// GetConnectedAt returns the connection timestamp in milliseconds
func (ci *ClientInfo) GetConnectedAt() int64 {
	return ci.ConnectedAt
}
