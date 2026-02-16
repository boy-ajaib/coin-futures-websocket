package server

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coin-futures-websocket/config"
	"coin-futures-websocket/internal/websocket/protocol"

	"github.com/gorilla/websocket"
)

// CfxUserMapper resolves an Ajaib user ID to a CFX user ID
type CfxUserMapper interface {
	GetCfxUserID(ctx context.Context, ajaibID int64) (string, error)
}

// UserPreferenceProvider fetches a user's futures quote preference
type UserPreferenceProvider interface {
	GetQuotePreference(ctx context.Context, ajaibID string) (string, error)
}

// Server represents a WebSocket server
type Server struct {
	hub              *Hub
	httpServer       *http.Server
	upgrader         websocket.Upgrader
	config           *config.WebSocketServerConfiguration
	logger           *slog.Logger
	handler          MessageHandler
	clientConfig     *ClientConfig
	cfxUserMapper    CfxUserMapper
	userPrefProvider UserPreferenceProvider
}

// NewServer creates a new WebSocket server
func NewServer(cfg *config.WebSocketServerConfiguration, logger *slog.Logger) *Server {
	hub := NewHub(cfg.MaxConnectionsPerUser, logger)

	clientConfig := &ClientConfig{
		PingInterval: time.Duration(cfg.PingIntervalMs) * time.Millisecond,
		PingTimeout:  time.Duration(cfg.PingTimeoutMs) * time.Millisecond,
		WriteWait:    10 * time.Second,
		ReadLimit:    512 * 1024, // 512KB
		SendBuffer:   256,
	}

	readBufferSize := cfg.ReadBufferSize
	if readBufferSize == 0 {
		readBufferSize = 1024
	}

	writeBufferSize := cfg.WriteBufferSize
	if writeBufferSize == 0 {
		writeBufferSize = 1024
	}

	s := &Server{
		hub:    hub,
		config: cfg,
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clientConfig: clientConfig,
	}

	return s
}

// SetMessageHandler sets the handler for incoming client messages
func (s *Server) SetMessageHandler(handler MessageHandler) {
	s.handler = handler
}

// SetCfxUserMapper sets the mapper used to resolve Ajaib ID to CFX user ID at connection time
func (s *Server) SetCfxUserMapper(mapper CfxUserMapper) {
	s.cfxUserMapper = mapper
}

// SetUserPreferenceProvider sets the provider used to fetch user quote preference at connection time
func (s *Server) SetUserPreferenceProvider(provider UserPreferenceProvider) {
	s.userPrefProvider = provider
}

// Hub returns the server's hub instance
func (s *Server) Hub() *Hub {
	return s.hub
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ws", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start hub in a goroutine
	go s.hub.Run()

	s.logger.Info("starting WebSocket server",
		"port", s.config.Port,
		"tls", s.config.TLSCertPath != "")

	// TODO: Start with TLS if configured
	if s.config.TLSCertPath != "" && s.config.TLSKeyPath != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		s.httpServer.TLSConfig = tlsConfig
		return s.httpServer.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down WebSocket server")
	return s.httpServer.Shutdown(ctx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","connections":%d}`, s.hub.GetClientCount())
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ajaibID := s.parseAjaibIDFromHeader(r)

	if ajaibID == "" {
		s.logger.Warn("unauthorized, ajaib_id/sub is empty",
			"ajaib_id", ajaibID,
			"remote_addr", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"type":"error","code":%d,"message":"unauthorized"}`, protocol.CodeUnauthorized)
		return
	}
	if !s.hub.CanUserConnect(ajaibID) {
		s.logger.Warn("connection limit reached, rejecting connection",
			"ajaib_id", ajaibID,
			"remote_addr", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, `{"type":"error","code":%d,"message":"connection limit reached"}`, protocol.CodeConnectionLimit)
		return
	}

	cfxUserID, err := s.resolveCfxUserID(ajaibID)
	if err != nil {
		s.logger.Error("failed to resolve cfx user id",
			"ajaib_id", ajaibID,
			"error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"type":"error","code":%d,"message":"failed to resolve cfx user id"}`, protocol.CodeCfxUserResolution)
		return
	}

	quotePreference, err := s.resolveQuotePreference(ajaibID)
	if err != nil {
		s.logger.Error("failed to fetch user quote preference",
			"ajaib_id", ajaibID,
			"error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"type":"error","code":%d,"message":"failed to fetch user preference"}`, protocol.CodeUserPreference)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	client := NewClient(s.hub, conn, s.clientConfig, ajaibID, cfxUserID, quotePreference, s.logger)

	// Register client with hub
	s.hub.register <- client

	// Start WritePump before sending any messages so the consumer is ready
	go client.WritePump()

	if err := client.SendMessage(protocol.NewConnectedMessage(client.ID(), ajaibID)); err != nil {
		s.logger.Warn("failed to send connected message", "client_id", client.ID(), "error", err)
	}

	s.logger.Info("new websocket connection",
		"client_id", client.ID(),
		"ajaib_id", ajaibID,
		"cfx_user_id", cfxUserID,
		"remote_addr", r.RemoteAddr)

	go client.ReadPump(s.handler)
}

// parseAjaibIDFromHeader extracts ajaib_id from the JWT in the X-Socket-Authorization header
func (s *Server) parseAjaibIDFromHeader(r *http.Request) string {
	header := r.Header.Get("X-Socket-Authorization")
	if header == "" {
		// Fallback: browsers cannot send custom headers on WebSocket upgrade
		header = r.URL.Query().Get("token")
	}
	if header == "" {
		return ""
	}

	token := strings.TrimPrefix(header, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		s.logger.Warn("invalid JWT format in X-Socket-Authorization")
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		s.logger.Warn("failed to decode JWT payload", "error", err)
		return ""
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		s.logger.Warn("failed to parse JWT claims", "error", err)
		return ""
	}

	return claims.Sub
}

// resolveCfxUserID maps an Ajaib ID string to a CFX user ID via the configured mapper
func (s *Server) resolveCfxUserID(ajaibID string) (string, error) {
	if ajaibID == "" || s.cfxUserMapper == nil {
		return "", fmt.Errorf("ajaib_id is empty or cfx user mapper is not configured")
	}

	id, err := strconv.ParseInt(ajaibID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid ajaib_id format: %w", err)
	}

	cfxUserID, err := s.cfxUserMapper.GetCfxUserID(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ajaib_id to cfx_user_id: %w", err)
	}

	return cfxUserID, nil
}

// resolveQuotePreference fetches the user's futures quote preference via the configured provider
func (s *Server) resolveQuotePreference(ajaibID string) (string, error) {
	if ajaibID == "" || s.userPrefProvider == nil {
		return "", nil
	}

	return s.userPrefProvider.GetQuotePreference(context.Background(), ajaibID)
}
