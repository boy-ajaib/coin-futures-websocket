package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type (
	Configuration struct {
		IsLoaded        bool                         `mapstructure:"is_loaded"`
		App             AppConfiguration             `mapstructure:"app"`
		Kafka           KafkaConfiguration           `mapstructure:"kafka"`
		WebSocketServer WebSocketServerConfiguration `mapstructure:"websocket_server"`
		Centrifuge      CentrifugeConfiguration      `mapstructure:"centrifuge"`
		CoinCfxAdapter  CoinCfxAdapterConfiguration  `mapstructure:"coin_cfx_adapter"`
		CoinData        CoinDataConfiguration        `mapstructure:"coin_data"`
		CoinSetting     CoinSettingConfiguration     `mapstructure:"coin_setting"`
	}

	AppConfiguration struct {
		Env      string `mapstructure:"env"`
		LogLevel string `mapstructure:"log_level"`
	}

	KafkaConfiguration struct {
		Brokers           []string `mapstructure:"brokers"`
		Topics            []string `mapstructure:"topics"`
		ConsumerGroup     string   `mapstructure:"consumer_group"`
		InitialOffset     string   `mapstructure:"initial_offset"`
		SessionTimeout    int      `mapstructure:"session_timeout"`
		HeartbeatInterval int      `mapstructure:"heartbeat_interval"`
		MaxMessageAgeMs   int      `mapstructure:"max_message_age_ms"`
	}

	WebSocketServerConfiguration struct {
		Enabled               bool   `mapstructure:"enabled"`
		Port                  int    `mapstructure:"port"`
		TLSCertPath           string `mapstructure:"tls_cert_path"`
		TLSKeyPath            string `mapstructure:"tls_key_path"`
		PingIntervalMs        int    `mapstructure:"ping_interval_ms"`
		PingTimeoutMs         int    `mapstructure:"ping_timeout_ms"`
		MaxConnectionsPerUser int    `mapstructure:"max_connections_per_user"`
		ReadBufferSize        int    `mapstructure:"read_buffer_size"`
		WriteBufferSize       int    `mapstructure:"write_buffer_size"`
		ShutdownTimeoutMs     int    `mapstructure:"shutdown_timeout_ms"`
	}

	RedisBrokerConfiguration struct {
		Enabled        bool   `mapstructure:"enabled"`
		Address        string `mapstructure:"address"`
		Password       string `mapstructure:"password"`
		DB             int    `mapstructure:"db"`
		Prefix         string `mapstructure:"prefix"`
		ConnectTimeout int    `mapstructure:"connect_timeout_ms"`
		IOTimeout      int    `mapstructure:"io_timeout_ms"`
	}

	CentrifugeConfiguration struct {
		// NodeName is the unique identifier for this Centrifuge node
		NodeName string `mapstructure:"node_name"`

		// Namespace is the application namespace for channels
		Namespace string `mapstructure:"namespace"`

		// LogLevel is the Centrifuge log level (debug, info, warn, error)
		LogLevel string `mapstructure:"log_level"`

		// ClientInsecure disables client-side token verification (for development only)
		ClientInsecure bool `mapstructure:"client_insecure"`

		// ClientAnnounce sends announcement messages to clients
		ClientAnnounce bool `mapstructure:"client_announce"`

		// Presence enables presence tracking for channels
		Presence bool `mapstructure:"presence"`

		// JoinLeave enables join/leave messages for channels
		JoinLeave bool `mapstructure:"join_leave"`

		// HistorySize is the number of messages to keep in channel history
		HistorySize int `mapstructure:"history_size"`

		// HistoryTTL is the time-to-live for channel history messages in seconds
		HistoryTTL int `mapstructure:"history_ttl_seconds"`

		// ForceRecovery enables position recovery for clients
		ForceRecovery bool `mapstructure:"force_recovery"`

		// RedisBroker configures Redis-based broker for cross-pod message delivery
		RedisBroker RedisBrokerConfiguration `mapstructure:"redis_broker"`
	}

	CoinCfxAdapterConfiguration struct {
		Host            string `mapstructure:"host"`
		CacheTTLSeconds int    `mapstructure:"cache_ttl_seconds"`
	}

	CoinDataConfiguration struct {
		Host            string `mapstructure:"host"`
		CacheTTLSeconds int    `mapstructure:"cache_ttl_seconds"`
		CfxUsdtAsset    string `mapstructure:"cfx_usdt_asset"`
	}

	CoinSettingConfiguration struct {
		Host            string `mapstructure:"host"`
		CacheTTLSeconds int    `mapstructure:"cache_ttl_seconds"`
	}
)

var configuration Configuration

// Get returns the configuration instance
func Get() *Configuration {
	if configuration.IsLoaded {
		return &configuration
	}

	configPath := "config/config.yml"
	env := os.Getenv("ENV")
	if env == "development" {
		configPath = "config/development.yml"
	}

	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file. %s", err)
	}

	if err := viper.Unmarshal(&configuration); err != nil {
		log.Fatalf("Unable to decode into struct. %v", err)
	}

	configuration.IsLoaded = true
	return &configuration
}
