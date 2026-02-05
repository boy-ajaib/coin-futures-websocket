package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"coin-futures-websocket/internal/util/auth"

	"github.com/centrifugal/centrifuge-go"
)

// CFXClient manages the WebSocket connection to CFX
type CFXClient struct {
	client        *centrifuge.Client
	authenticator *auth.BrokerAuthenticator
	logger        *slog.Logger
	config        *ClientConfig

	// State management
	mu            sync.RWMutex
	connected     bool
	authenticated bool
	privateID     string
	authDone      chan struct{} // Closed when authentication attempt completes
	authError     error

	// Context management for authentication
	cancelAuth context.CancelFunc
}

// ClientConfig holds configuration for the CFX client
type ClientConfig struct {
	Host               string
	MaxServerPingDelay time.Duration
	MaxReconnectDelay  time.Duration
	MinReconnectDelay  time.Duration
	Timeout            time.Duration
}

// AuthResponse represents the response from broker/auth RPC
type AuthResponse struct {
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
	Method    string `json:"method"`
	Channel   string `json:"channel"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      struct {
		PrivateID string `json:"private_id"`
	} `json:"data"`
}

// getPort extracts the port from a URL or returns the default for the scheme
func getPort(u *url.URL) string {
	if u.Port() != "" {
		return u.Port()
	}
	switch u.Scheme {
	case "ws":
		return "80"
	case "wss":
		return "443"
	default:
		return "unknown"
	}
}

// NewCFXClient creates a new CFX WebSocket client
func NewCFXClient(config *ClientConfig, authenticator *auth.BrokerAuthenticator, logger *slog.Logger) (*CFXClient, error) {
	client := &CFXClient{
		authenticator: authenticator,
		logger:        logger,
		config:        config,
		authDone:      nil,
	}

	wsURL := config.Host + "/connection/websocket"

	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("parse WebSocket URL: %w", err)
	}

	logger.Info("CFX client configuration",
		"url", wsURL,
		"scheme", parsedURL.Scheme,
		"host", parsedURL.Host,
		"path", parsedURL.Path,
		"port", getPort(parsedURL),
		"key_id", authenticator.GetKeyID())

	centrifugeConfig := centrifuge.Config{
		Name:               "coin-cfx-streamer-go-client",
		Version:            "1.0.0",
		HandshakeTimeout:   config.Timeout,
		MaxServerPingDelay: config.MaxServerPingDelay,
		MaxReconnectDelay:  config.MaxReconnectDelay,
		MinReconnectDelay:  config.MinReconnectDelay,
		Header: http.Header{
			"User-Agent": []string{"Coin-CFX-Streamer-Go-Client/1.0.0"},
		},
	}

	c := centrifuge.NewJsonClient(wsURL, centrifugeConfig)

	client.client = c
	client.setupEventHandlers()

	return client, nil
}

// setupEventHandlers configures all event handlers for the Centrifuge client
func (c *CFXClient) setupEventHandlers() {
	c.client.OnConnecting(func(e centrifuge.ConnectingEvent) {
		c.logger.Info("connecting to CFX", "code", e.Code, "reason", e.Reason)
	})

	c.client.OnConnected(func(e centrifuge.ConnectedEvent) {
		c.mu.Lock()

		if c.cancelAuth != nil {
			c.cancelAuth()
		}

		c.connected = true
		c.authenticated = false
		c.authError = nil

		c.authDone = make(chan struct{})
		authDone := c.authDone

		var authCtx context.Context
		authCtx, c.cancelAuth = context.WithTimeout(context.Background(), c.config.Timeout)
		c.mu.Unlock()

		c.logger.Info("connected to CFX", "client_id", e.ClientID)

		go c.authenticateWithContext(authCtx, authDone)
	})

	c.client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		c.mu.Lock()
		c.connected = false
		c.authenticated = false
		c.mu.Unlock()

		c.logger.Warn("disconnected from CFX", "code", e.Code, "reason", e.Reason)
	})

	c.client.OnError(func(e centrifuge.ErrorEvent) {
		c.logger.Error("CFX client error", "error", e.Error.Error())
	})

	c.client.OnMessage(func(e centrifuge.MessageEvent) {
		c.logger.Debug("received message", "data", string(e.Data))
	})
}

// Connect establishes connection to CFX WebSocket
func (c *CFXClient) Connect() error {
	return c.ConnectWithContext(context.Background())
}

// ConnectWithContext establishes connection with timeout
func (c *CFXClient) ConnectWithContext(ctx context.Context) error {
	c.logger.Info("initiating connection to CFX", "host", c.config.Host)

	errChan := make(chan error, 1)

	go func() {
		errChan <- c.client.Connect()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("connection timeout: %w", ctx.Err())
	}
}

// authenticateWithContext performs broker authentication via RPC with context cancellation support
func (c *CFXClient) authenticateWithContext(ctx context.Context, authDone chan struct{}) {
	c.logger.Info("starting broker authentication")

	defer close(authDone)

	select {
	case <-ctx.Done():
		c.logger.Debug("authentication cancelled before start")
		c.mu.Lock()
		c.authError = ctx.Err()
		c.mu.Unlock()
		return
	default:
	}

	authRequest, err := c.authenticator.CreateAuthRequest()
	if err != nil {
		c.logger.Error("failed to create auth request", "error", err)
		c.handleAuthFailure(err)
		c.mu.Lock()
		c.authError = err
		c.mu.Unlock()
		return
	}

	result, err := c.client.RPC(ctx, "broker/auth", authRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			c.logger.Debug("authentication RPC cancelled", "error", err)
			c.mu.Lock()
			c.authError = err
			c.mu.Unlock()
			return
		}

		c.logger.Error("broker/auth RPC failed", "error", err)
		c.handleAuthFailure(err)
		c.mu.Lock()
		c.authError = err
		c.mu.Unlock()
		return
	}

	var authResp AuthResponse
	if err := json.Unmarshal(result.Data, &authResp); err != nil {
		c.logger.Error("failed to parse auth response", "error", err, "data", string(result.Data))
		c.handleAuthFailure(err)
		c.mu.Lock()
		c.authError = err
		c.mu.Unlock()
		return
	}

	if authResp.Data.PrivateID == "" {
		c.logger.Error("received empty private_id from auth response")
		c.handleAuthFailure(errors.New("empty private_id"))
		c.mu.Lock()
		c.authError = errors.New("empty private_id")
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.authenticated = true
	c.privateID = authResp.Data.PrivateID
	c.authError = nil
	c.mu.Unlock()

	c.logger.Info("broker authentication successful", "private_id", authResp.Data.PrivateID)
}

// handleAuthFailure handles authentication failures
func (c *CFXClient) handleAuthFailure(err error) {
	c.logger.Error("authentication failed", "error", err)
	c.mu.Lock()
	c.authenticated = false
	c.mu.Unlock()
}

// CentrifugeClient returns the underlying Centrifuge client
func (c *CFXClient) CentrifugeClient() *centrifuge.Client {
	return c.client
}

// IsConnected returns whether the client is connected
func (c *CFXClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// IsAuthenticated returns whether the client is authenticated
func (c *CFXClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// GetPrivateID returns the private ID received from authentication
func (c *CFXClient) GetPrivateID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.privateID
}

// Close gracefully closes the connection
func (c *CFXClient) Close() error {
	c.logger.Info("closing CFX client")
	c.client.Close()
	return nil
}

// CloseWithContext gracefully closes the connection with timeout
func (c *CFXClient) CloseWithContext(ctx context.Context) error {
	c.logger.Info("closing CFX client with context")

	c.mu.Lock()
	if c.cancelAuth != nil {
		c.cancelAuth()
	}
	c.mu.Unlock()

	c.client.Close()
	return nil
}

// WaitForAuthentication waits for authentication to complete with a timeout
func (c *CFXClient) WaitForAuthentication(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var authDone chan struct{}
	for {
		c.mu.RLock()
		authDone = c.authDone
		c.mu.RUnlock()

		if authDone != nil {
			break
		}

		select {
		case <-ticker.C:
			// Continue polling
		case <-timer.C:
			return fmt.Errorf("authentication timeout waiting for connection: %v", timeout)
		}
	}

	// Now wait for authentication to complete
	select {
	case <-authDone:
		c.mu.RLock()
		defer c.mu.RUnlock()
		if c.authenticated {
			return nil
		}
		if c.authError != nil {
			return c.authError
		}
		return errors.New("authentication failed")
	case <-timer.C:
		return fmt.Errorf("authentication timeout after %v", timeout)
	}
}
