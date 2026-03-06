package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coin-futures-websocket/config"
	"coin-futures-websocket/internal/kafka"
	"coin-futures-websocket/internal/service"
	"coin-futures-websocket/internal/websocket/server"

	"github.com/centrifugal/centrifuge"
)

func main() {
	cfg := config.Get()

	logger := initLogger(cfg)
	logger.Info("starting WebSocket service",
		"env", cfg.App.Env,
		"ws_server_enabled", cfg.WebSocketServer.Enabled)

	transformer, currencyService := initTransformer(cfg, logger)
	wsServer := initCentrifugeServer(cfg, logger)

	// Initialize metrics
	metrics := server.NewMetrics(wsServer.Node())
	if err := metrics.Register(); err != nil {
		logger.Warn("failed to register metrics", "error", err)
	} else {
		wsServer.SetMetrics(metrics)
		// Start background metrics collector
		wsServer.StartMetricsCollector(metrics, 10*time.Second)
		logger.Info("metrics endpoint available", "path", "/metrics")
	}

	kafkaConsumer, broadcaster, err := initKafkaConsumer(cfg, transformer, wsServer.Node(), logger)
	if err != nil {
		logger.Error("failed to initialize Kafka consumer", "error", err)
		os.Exit(1)
	}

	// Set the broadcaster on the WebSocket server for subscription tracking
	wsServer.SetBroadcaster(broadcaster)

	// Start Kafka consumer
	go func() {
		if err := kafkaConsumer.Start(context.Background()); err != nil && err != context.Canceled {
			logger.Error("Kafka consumer error", "error", err)
		}
	}()

	// Start Centrifuge WebSocket server
	if err := wsServer.Start(); err != nil {
		logger.Error("failed to start WebSocket server", "error", err)
		os.Exit(1)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","connections":%d}`, wsServer.GetClientCount())
	})
	mux.HandleFunc("/connection", wsServer.ServeHTTP)
	wsServer.SetupMetricsHandler(mux, "/metrics")

	// Create HTTP server (accessible for graceful shutdown)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.WebSocketServer.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in background
	go func() {
		logger.Info("HTTP server listening", "port", cfg.WebSocketServer.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	logger.Info("service running. Press Ctrl+C to exit.")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.WebSocketServer.ShutdownTimeoutMs)*time.Millisecond)
	defer shutdownCancel()

	// Shutdown HTTP server first (stop accepting new connections)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutting down HTTP server", "error", err)
	}

	// Shutdown Centrifuge WebSocket server
	if err := wsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutting down WebSocket server", "error", err)
	}

	// Stop currency service
	currencyService.Stop()

	if kafkaConsumer != nil {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Error("error closing Kafka consumer", "error", err)
		}
	}

	logger.Info("shutdown complete")
}

// initTransformer creates the currency transformer with the coin-data rate provider.
func initTransformer(cfg *config.Configuration, logger *slog.Logger) (service.TransformerInterface, *service.CachedCurrencyService) {
	rateProvider := service.NewHTTPRateProvider(cfg.CoinData.Host, logger)
	currencyService := service.NewCachedCurrencyService(
		rateProvider,
		time.Duration(cfg.CoinData.CacheTTLSeconds)*time.Second,
		logger,
	)
	return service.NewTransformer(currencyService, cfg.CoinData.CfxUsdtAsset, logger), currencyService
}

// initCentrifugeServer creates the Centrifuge WebSocket server.
func initCentrifugeServer(cfg *config.Configuration, logger *slog.Logger) *server.CentrifugeServer {
	wsServer := server.NewCentrifugeServer(&cfg.Centrifuge, logger)
	wsServer.SetMaxConnectionsPerUser(cfg.WebSocketServer.MaxConnectionsPerUser)

	cfxCacheTTL := time.Duration(cfg.CoinCfxAdapter.CacheTTLSeconds) * time.Second
	cfxUserMappingClient := service.NewHTTPCfxUserMappingClient(cfg.CoinCfxAdapter.Host, cfxCacheTTL, logger)
	wsServer.SetCfxUserMapper(cfxUserMappingClient)

	prefCacheTTL := time.Duration(cfg.CoinSetting.CacheTTLSeconds) * time.Second
	userPrefClient := service.NewHTTPUserPreferenceClient(cfg.CoinSetting.Host, prefCacheTTL, logger)
	wsServer.SetUserPreferenceProvider(userPrefClient)

	return wsServer
}

// initKafkaConsumer creates the Broadcaster and Kafka consumer, wiring the broadcaster to the Centrifuge node.
func initKafkaConsumer(cfg *config.Configuration, transformer service.TransformerInterface, node interface{}, logger *slog.Logger) (*kafka.KafkaReaderConsumer, *kafka.Broadcaster, error) {
	// Create the Kafka broadcaster with the Centrifuge node
	broadcaster := kafka.NewBroadcaster(node.(*centrifuge.Node), transformer, logger)

	kafkaConfig := &kafka.ConsumerConfig{
		Brokers:           cfg.Kafka.Brokers,
		GroupID:           cfg.Kafka.ConsumerGroup,
		Topics:            cfg.Kafka.Topics,
		InitialOffset:     cfg.Kafka.InitialOffset,
		SessionTimeout:    time.Duration(cfg.Kafka.SessionTimeout) * time.Millisecond,
		HeartbeatInterval: time.Duration(cfg.Kafka.HeartbeatInterval) * time.Millisecond,
		Handler:           broadcaster.HandleMessage,
		MaxMessageAge:     time.Duration(cfg.Kafka.MaxMessageAgeMs) * time.Millisecond,
	}

	consumer, err := kafka.NewKafkaReaderConsumer(kafkaConfig, logger)
	if err != nil {
		return nil, nil, err
	}

	return consumer, broadcaster, nil
}

// initLogger initializes the structured logger with configuration.
func initLogger(cfg *config.Configuration) *slog.Logger {
	var level slog.Level
	switch cfg.App.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.App.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
