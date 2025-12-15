package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Port            int    `mapstructure:"PORT"`
	ServerCert      string `mapstructure:"SERVER_CERT"`
	ServerKey       string `mapstructure:"SERVER_KEY"`
	CAFile          string `mapstructure:"CA_FILE"`
	RequireClientCert bool `mapstructure:"REQUIRE_CLIENT_CERT"`
	RedisURL string `mapstructure:"REDIS_URL"`
	IdempotencyTTL int `mapstructure:"IDEMPOTENCY_TTL_HOURS"`
}

func Load() (*Config, error) {
	viper.SetDefault("PORT", 8443)
	viper.SetDefault("REQUIRE_CLIENT_CERT", true)

	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// Allow overrides via .env file (useful locally)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig() 
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("IDEMPOTENCY_TTL_HOURS", 24)// ignore error if no file

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Required paths
	required := []string{"SERVER_CERT", "SERVER_KEY", "CA_FILE"}
	for _, key := range required {
		if viper.GetString(key) == "" {
			return nil, fmt.Errorf("missing required config: %s", key)
		}
	}

	return cfg, nil
}