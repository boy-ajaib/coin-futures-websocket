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
		CoinCfxAdapter  CoinCfxAdapterConfiguration  `mapstructure:"coin_cfx_adapter"`
		CoinData        CoinDataConfiguration        `mapstructure:"coin_data"`
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

	CoinCfxAdapterConfiguration struct {
		Host string `mapstructure:"host"`
	}

	CoinDataConfiguration struct {
		Host            string `mapstructure:"host"`
		CacheTTLSeconds int    `mapstructure:"cache_ttl_seconds"`
		CfxUsdtAsset    string `mapstructure:"cfx_usdt_asset"`
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
