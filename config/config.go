package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type (
	Configuration struct {
		IsLoaded bool             `mapstructure:"is_loaded"`
		App      AppConfiguration `mapstructure:"app"`
		Cfx      CfxConfiguration `mapstructure:"cfx"`
	}

	AppConfiguration struct {
		Env      string `mapstructure:"env"`
		LogLevel string `mapstructure:"log_level"`
	}

	CfxConfiguration struct {
		KeyBrokerage CfxKeyBrokerageConfiguration `mapstructure:"key_brokerage"`
		Ws           CfxWsConfiguration           `mapstructure:"ws"`
	}

	CfxKeyBrokerageConfiguration struct {
		KeyId      int    `mapstructure:"key_id"`
		PrivateKey string `mapstructure:"private_key"`
	}

	CfxWsConfiguration struct {
		Host               string `mapstructure:"host"`
		MaxServerPingDelay int    `mapstructure:"maxServerPingDelay"`
		MaxReconnectDelay  int    `mapstructure:"maxReconnectDelay"`
		MinReconnectDelay  int    `mapstructure:"minReconnectDelay"`
		Timeout            int    `mapstructure:"timeout"`
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
