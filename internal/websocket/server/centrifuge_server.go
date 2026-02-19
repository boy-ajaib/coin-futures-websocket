package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"coin-futures-websocket/config"

	"github.com/centrifugal/centrifuge"
)

// CfxUserMapper resolves an Ajaib user ID to a CFX user ID
type CfxUserMapper interface {
	GetCfxUserID(ctx context.Context, ajaibID int64) (string, error)
}

// UserPreferenceProvider fetches a user's futures quote preference
type UserPreferenceProvider interface {
	GetQuotePreference(ctx context.Context, ajaibID string) (string, error)
}

// KafkaBroadcaster is the interface for the Kafka broadcaster (used to avoid circular dependency)
type KafkaBroadcaster interface {
	RegisterSubscription(cfxUserID, ajaibID, quotePreference string)
	UnregisterSubscription(cfxUserID string)
}

// CentrifugeServer wraps the Centrifuge library server
type CentrifugeServer struct {
	node      *centrifuge.Node
	wsHandler *centrifuge.WebsocketHandler
	config    *config.CentrifugeConfiguration
	logger    *slog.Logger
	metrics   *Metrics

	// Dependencies for handlers
	cfxUserMapper    CfxUserMapper
	userPrefProvider UserPreferenceProvider
	broadcaster      KafkaBroadcaster
}

// NewCentrifugeServer creates a new Centrifuge server instance
func NewCentrifugeServer(cfg *config.CentrifugeConfiguration, logger *slog.Logger) *CentrifugeServer {
	// Create structured log handler for Centrifuge
	var logHandler centrifuge.LogHandler
	if cfg.LogLevel == "debug" || cfg.LogLevel == "info" {
		// Use text handler for development
		logHandler = NewDevelopmentLogHandler(cfg.LogLevel)
	} else {
		// Use JSON handler for production
		logHandler = NewProductionLogHandler(cfg.LogLevel)
	}

	// Create Centrifuge node configuration
	centrifugeCfg := centrifuge.Config{
		Name:               cfg.NodeName,
		LogHandler:         logHandler,
		LogLevel:           centrifuge.LogLevelInfo,
		ChannelMaxLength:   255,
		ClientQueueMaxSize: 1048576, // 1MB default
	}

	// Set log level based on config
	switch cfg.LogLevel {
	case "debug":
		centrifugeCfg.LogLevel = centrifuge.LogLevelDebug
	case "info":
		centrifugeCfg.LogLevel = centrifuge.LogLevelInfo
	case "warn":
		centrifugeCfg.LogLevel = centrifuge.LogLevelWarn
	case "error":
		centrifugeCfg.LogLevel = centrifuge.LogLevelError
	}

	node, err := centrifuge.New(centrifugeCfg)
	if err != nil {
		logger.Error("failed to create centrifuge node", "error", err)
		panic(err)
	}

	// Create WebSocket handler
	wsCfg := centrifuge.WebsocketConfig{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
		PingPongConfig: centrifuge.PingPongConfig{
			PingInterval: 2 * time.Second,
		},
	}
	wsHandler := centrifuge.NewWebsocketHandler(node, wsCfg)

	return &CentrifugeServer{
		node:      node,
		wsHandler: wsHandler,
		config:    cfg,
		logger:    logger,
	}
}

// SetCfxUserMapper sets the mapper used to resolve Ajaib ID to CFX user ID
func (s *CentrifugeServer) SetCfxUserMapper(mapper CfxUserMapper) {
	s.cfxUserMapper = mapper
}

// SetUserPreferenceProvider sets the provider used to fetch user quote preference
func (s *CentrifugeServer) SetUserPreferenceProvider(provider UserPreferenceProvider) {
	s.userPrefProvider = provider
}

// SetBroadcaster sets the Kafka broadcaster for subscription tracking
func (s *CentrifugeServer) SetBroadcaster(broadcaster KafkaBroadcaster) {
	s.broadcaster = broadcaster
}

// SetMetrics sets the metrics collector for the server
func (s *CentrifugeServer) SetMetrics(metrics *Metrics) {
	s.metrics = metrics
}

// Node returns the underlying Centrifuge node
func (s *CentrifugeServer) Node() *centrifuge.Node {
	return s.node
}

// Start starts the Centrifuge server
func (s *CentrifugeServer) Start() error {
	s.logger.Info("starting centrifuge server",
		"node_name", s.config.NodeName,
		"namespace", s.config.Namespace)

	// Setup handlers before running the node
	s.SetupHandlers()

	// Run node in background.
	go func() {
		if err := s.node.Run(); err != nil {
			s.logger.Error("centrifuge node run error", "error", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the server
func (s *CentrifugeServer) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down centrifuge server")
	return s.node.Shutdown(ctx)
}

// ServeHTTP serves WebSocket connections via HTTP handler
func (s *CentrifugeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.wsHandler.ServeHTTP(w, r)
}

// GetClientCount returns the total number of connected clients
func (s *CentrifugeServer) GetClientCount() int {
	return s.node.Hub().NumClients()
}
