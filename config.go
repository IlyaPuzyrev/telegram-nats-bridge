package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Subject       string `mapstructure:"subject"`
	TelegramToken string `mapstructure:"telegram_token,omitempty"`
	NATSURL       string `mapstructure:"nats_url,omitempty"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string, logger *slog.Logger) (*Config, error) {
	v := viper.New()

	// Read from environment variables (only for telegram_token and nats_url)
	// Subject is read only from YAML file
	v.BindEnv("telegram_token", "TELEGRAM_BOT_TOKEN")
	v.BindEnv("nats_url", "NATS_URL")

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		logger.Info("loading config file", "path", configPath)

		if err := v.ReadInConfig(); err != nil {
			logger.Error("failed to read config file", "error", err)
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		logger.Info("config file loaded successfully")
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		logger.Error("failed to unmarshal config", "error", err)
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.Subject == "" {
		logger.Error("subject is required")
		return nil, fmt.Errorf("subject is required")
	}

	logger.Info("configuration loaded",
		"subject", cfg.Subject,
		"has_telegram_token", cfg.TelegramToken != "",
		"nats_url", cfg.NATSURL)

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Subject == "" {
		return fmt.Errorf("subject is required (must be defined in YAML config)")
	}

	if c.TelegramToken == "" {
		return fmt.Errorf("telegram token is required (set TELEGRAM_BOT_TOKEN env or telegram_token in config)")
	}

	if c.NATSURL == "" {
		return fmt.Errorf("nats url is required (set NATS_URL env or nats_url in config)")
	}

	return nil
}

// ValidateConfigPath validates that the config file exists
func ValidateConfigPath(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path is required")
	}

	info, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	if info.IsDir() {
		return fmt.Errorf("config path is a directory: %s", configPath)
	}

	// Check file extension
	ext := filepath.Ext(configPath)
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("config file must be a YAML file (.yaml or .yml), got: %s", ext)
	}

	return nil
}
