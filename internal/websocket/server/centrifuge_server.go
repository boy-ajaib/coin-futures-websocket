package server

import (
	"context"
	"fmt"
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

	// Setup Redis broker for cross-pod message delivery if enabled
	if cfg.RedisBroker.Enabled {
		shard, err := centrifuge.NewRedisShard(node, centrifuge.RedisShardConfig{
			Address:        cfg.RedisBroker.Address,
			Password:       cfg.RedisBroker.Password,
			DB:             cfg.RedisBroker.DB,
			ConnectTimeout: time.Duration(cfg.RedisBroker.ConnectTimeout) * time.Millisecond,
			IOTimeout:      time.Duration(cfg.RedisBroker.IOTimeout) * time.Millisecond,
		})
		if err != nil {
			logger.Error("failed to create redis shard", "error", err)
			panic(err)
		}

		broker, err := centrifuge.NewRedisBroker(node, centrifuge.RedisBrokerConfig{
			Prefix: cfg.RedisBroker.Prefix,
			Shards: []*centrifuge.RedisShard{shard},
		})
		if err != nil {
			logger.Error("failed to create redis broker", "error", err)
			panic(err)
		}

		node.SetBroker(broker)
		logger.Info("centrifuge redis broker enabled", "address", cfg.RedisBroker.Address, "prefix", cfg.RedisBroker.Prefix)
	} else {
		logger.Info("centrifuge using in-memory broker (redis broker disabled)")
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

// Start starts the Centrifuge server.
func (s *CentrifugeServer) Start() error {
	s.logger.Info("starting centrifuge server",
		"node_name", s.config.NodeName,
		"namespace", s.config.Namespace)

	// Setup handlers before running the node
	s.SetupHandlers()

	// Run node synchronously to ensure broker is fully initialized before returning
	if err := s.node.Run(); err != nil {
		return fmt.Errorf("failed to start centrifuge node: %w", err)
	}

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
