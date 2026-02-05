package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coin-futures-websocket/config"
	"coin-futures-websocket/internal/client"
	"coin-futures-websocket/internal/handler"
	"coin-futures-websocket/internal/subscription"
	"coin-futures-websocket/internal/util/auth"
)

func main() {
	cfg := config.Get()

	logger := initLogger(cfg)
	logger.Info("starting CFX WebSocket client",
		"env", cfg.App.Env,
		"host", cfg.Cfx.Ws.Host)

	authenticator, err := auth.NewBrokerAuthenticator(
		cfg.Cfx.KeyBrokerage.PrivateKey,
		cfg.Cfx.KeyBrokerage.KeyId,
	)
	if err != nil {
		logger.Error("failed to create authenticator", "error", err)
		os.Exit(1)
	}

	clientConfig := &client.ClientConfig{
		Host:               cfg.Cfx.Ws.Host,
		MaxServerPingDelay: time.Duration(cfg.Cfx.Ws.MaxServerPingDelay) * time.Millisecond,
		MaxReconnectDelay:  time.Duration(cfg.Cfx.Ws.MaxReconnectDelay) * time.Millisecond,
		MinReconnectDelay:  time.Duration(cfg.Cfx.Ws.MinReconnectDelay) * time.Millisecond,
		Timeout:            time.Duration(cfg.Cfx.Ws.Timeout) * time.Millisecond,
	}

	cfxClient, err := client.NewCFXClient(clientConfig, authenticator, logger)
	if err != nil {
		logger.Error("failed to create CFX client", "error", err)
		os.Exit(1)
	}

	subManager := subscription.NewManager(cfxClient.CentrifugeClient(), logger)

	// Register handlers
	heartbeatHandler := handler.NewHeartbeatHandler(logger)
	subManager.RegisterHandler(heartbeatHandler)

	if err := cfxClient.Connect(); err != nil {
		logger.Error("failed to connect", "error", err)
		os.Exit(1)
	}

	// Wait for authentication to complete before subscribing
	if err := cfxClient.WaitForAuthentication(10 * time.Second); err != nil {
		logger.Error("authentication failed", "error", err)
		cfxClient.Close()
		os.Exit(1)
	}

	// Subscribe to channels
	if err := subManager.Subscribe("heartbeat"); err != nil {
		logger.Error("failed to subscribe to heartbeat", "error", err)
		cfxClient.Close()
		os.Exit(1)
	}

	logger.Info("CFX client running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	subManager.UnsubscribeAll()

	if err := cfxClient.CloseWithContext(shutdownCtx); err != nil {
		logger.Error("error closing client", "error", err)
	}

	logger.Info("shutdown complete")
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
