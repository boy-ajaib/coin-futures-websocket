package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coin-futures-websocket/config"
	"coin-futures-websocket/internal/kafka"
	"coin-futures-websocket/internal/service"
	wshandler "coin-futures-websocket/internal/websocket/handler"
	"coin-futures-websocket/internal/websocket/server"
)

func main() {
	cfg := config.Get()

	logger := initLogger(cfg)
	logger.Info("starting WebSocket service",
		"env", cfg.App.Env,
		"ws_server_enabled", cfg.WebSocketServer.Enabled)

	transformer := initTransformer(cfg, logger)

	wsServer, messageHandler, err := initWebSocketServer(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize WebSocket server", "error", err)
		os.Exit(1)
	}

	kafkaConsumer, err := initKafkaConsumer(cfg, transformer, wsServer.Hub(), messageHandler, logger)
	if err != nil {
		logger.Error("failed to initialize Kafka consumer", "error", err)
		os.Exit(1)
	}

	// Start Kafka consumer
	go func() {
		if err := kafkaConsumer.Start(context.Background()); err != nil && err != context.Canceled {
			logger.Error("Kafka consumer error", "error", err)
		}
	}()

	// Start WebSocket server
	go func() {
		if err := wsServer.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("WebSocket server error", "error", err)
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

	if err := wsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutting down WebSocket server", "error", err)
	}

	messageHandler.Stop()

	if kafkaConsumer != nil {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Error("error closing Kafka consumer", "error", err)
		}
	}

	logger.Info("shutdown complete")
}

// initTransformer creates the currency transformer with the coin-data rate provider.
func initTransformer(cfg *config.Configuration, logger *slog.Logger) service.TransformerInterface {
	rateProvider := service.NewHTTPRateProvider(cfg.CoinData.Host, logger)
	currencyService := service.NewCachedCurrencyService(
		rateProvider,
		time.Duration(cfg.CoinData.CacheTTLSeconds)*time.Second,
		logger,
	)
	return service.NewTransformer(currencyService, cfg.CoinData.CfxUsdtAsset, logger)
}

// initWebSocketServer creates the WebSocket server, channel manager, and message handler.
func initWebSocketServer(cfg *config.Configuration, logger *slog.Logger) (*server.Server, *wshandler.DefaultHandler, error) {
	wsServer := server.NewServer(&cfg.WebSocketServer, logger)

	cfxUserMappingClient := service.NewHTTPCfxUserMappingClient(cfg.CoinCfxAdapter.Host, logger)
	wsServer.SetCfxUserMapper(cfxUserMappingClient)

	messageHandler := wshandler.NewDefaultHandler(wsServer.Hub(), logger)

	wsServer.SetMessageHandler(messageHandler)
	return wsServer, messageHandler, nil
}

// initKafkaConsumer creates the Broadcaster and Kafka consumer, wiring the broadcaster to the message handler.
func initKafkaConsumer(cfg *config.Configuration, transformer service.TransformerInterface, hub *server.Hub, messageHandler *wshandler.DefaultHandler, logger *slog.Logger) (*kafka.KafkaReaderConsumer, error) {
	broadcaster := kafka.NewBroadcaster(hub, transformer, logger)
	messageHandler.SetKafkaBroadcaster(broadcaster)

	kafkaConfig := &kafka.ConsumerConfig{
		Brokers:           cfg.Kafka.Brokers,
		GroupID:           cfg.Kafka.ConsumerGroup,
		Topics:            cfg.Kafka.Topics,
		InitialOffset:     cfg.Kafka.InitialOffset,
		SessionTimeout:    time.Duration(cfg.Kafka.SessionTimeout) * time.Millisecond,
		HeartbeatInterval: time.Duration(cfg.Kafka.HeartbeatInterval) * time.Millisecond,
		Handler:           broadcaster.HandleMessage,
	}

	return kafka.NewKafkaReaderConsumer(kafkaConfig, logger)
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
