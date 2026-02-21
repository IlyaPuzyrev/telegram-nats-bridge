package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type RouteSubjectType string

const (
	SubjectTypeString RouteSubjectType = "string"
	SubjectTypeExpr   RouteSubjectType = "expr"
)

type RouteSubject struct {
	Type  RouteSubjectType `mapstructure:"type"`
	Value string           `mapstructure:"value"`
}

type Route struct {
	Condition string       `mapstructure:"condition"`
	Subject   RouteSubject `mapstructure:"subject"`
}

// Config holds the application configuration
type Config struct {
	Mode                   string  `mapstructure:"mode"`
	Routes                 []Route `mapstructure:"routes"`
	TelegramToken          string  `mapstructure:"telegram_token,omitempty"`
	NATSURL                string  `mapstructure:"nats_url,omitempty"`
	RouteWorkers           int     `mapstructure:"route_workers"`
	PublishWorkers         int     `mapstructure:"publish_workers"`
	PublishShutdownTimeout int     `mapstructure:"publish_shutdown_timeout"`
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

	if cfg.Mode == "" {
		cfg.Mode = "first"
	}

	if cfg.RouteWorkers == 0 {
		cfg.RouteWorkers = 5
	}

	if cfg.PublishWorkers == 0 {
		cfg.PublishWorkers = 5
	}

	if cfg.PublishShutdownTimeout == 0 {
		cfg.PublishShutdownTimeout = 10
	}

	logger.Info("configuration loaded",
		"mode", cfg.Mode,
		"routes_count", len(cfg.Routes),
		"has_telegram_token", cfg.TelegramToken != "",
		"nats_url", cfg.NATSURL,
		"route_workers", cfg.RouteWorkers,
		"publish_workers", cfg.PublishWorkers,
		"publish_shutdown_timeout", cfg.PublishShutdownTimeout)

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Mode != "first" && c.Mode != "all" {
		return fmt.Errorf("mode must be 'first' or 'all'")
	}

	if c.RouteWorkers <= 0 {
		return fmt.Errorf("route_workers must be > 0")
	}

	if c.PublishWorkers <= 0 {
		return fmt.Errorf("publish_workers must be > 0")
	}

	if c.PublishShutdownTimeout <= 0 {
		return fmt.Errorf("publish_shutdown_timeout must be > 0")
	}

	for i, route := range c.Routes {
		if route.Condition == "" {
			return fmt.Errorf("routes[%d].condition is required", i)
		}
		if route.Subject.Type == "" {
			return fmt.Errorf("routes[%d].subject.type is required", i)
		}
		if route.Subject.Value == "" {
			return fmt.Errorf("routes[%d].subject.value is required", i)
		}
		if route.Subject.Type != SubjectTypeString && route.Subject.Type != SubjectTypeExpr {
			return fmt.Errorf("routes[%d].subject.type must be 'string' or 'expr'", i)
		}
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
