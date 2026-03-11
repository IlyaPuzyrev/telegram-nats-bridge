package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func splitBrokers(s string) []string {
	if s == "" {
		return nil
	}
	brokers := strings.Split(s, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}
	return brokers
}

type RouteSubjectType string

const (
	SubjectTypeString RouteSubjectType = "string"
	SubjectTypeExpr   RouteSubjectType = "expr"
)

type BrokerType string

const (
	BrokerNATS  BrokerType = "nats"
	BrokerKafka BrokerType = "kafka"
)

type EngineType string

const (
	EngineCore      EngineType = "core"
	EngineJetStream EngineType = "jetstream"
)

type NATSConfig struct {
	URL       string           `mapstructure:"url"`
	Engine    EngineType       `mapstructure:"engine"`
	JetStream *JetStreamConfig `mapstructure:"jetstream"`
}

type KafkaConfig struct {
	Brokers           []string `mapstructure:"brokers"`
	Async             bool     `mapstructure:"async"`
	AckRequired       int      `mapstructure:"ack_required"`
	BatchSize         int      `mapstructure:"batch_size"`
	BatchBytes        int64    `mapstructure:"batch_bytes"`
	ReadTimeout       int      `mapstructure:"read_timeout"`
	WriteTimeout      int      `mapstructure:"write_timeout"`
	HeartbeatInterval int      `mapstructure:"heartbeat_interval"`
	CommitInterval    int      `mapstructure:"commit_interval"`
}

type JetStreamConfig struct {
	StreamConfig string `mapstructure:"stream_config"`
}

type RouteSubject struct {
	Type  RouteSubjectType `mapstructure:"type"`
	Value string           `mapstructure:"value"`
}

type RouteTopic struct {
	Type  RouteSubjectType `mapstructure:"type"`
	Value string           `mapstructure:"value"`
}

type RouteKey struct {
	Type  RouteSubjectType `mapstructure:"type"`
	Value string           `mapstructure:"value"`
}

type Route struct {
	Condition string        `mapstructure:"condition"`
	Subject   *RouteSubject `mapstructure:"subject,omitempty"`
	Topic     *RouteTopic   `mapstructure:"topic,omitempty"`
	Key       *RouteKey     `mapstructure:"key,omitempty"`
}

// Config holds the application configuration
type Config struct {
	Mode                   string       `mapstructure:"mode"`
	Routes                 []Route      `mapstructure:"routes"`
	Broker                 BrokerType   `mapstructure:"broker"`
	NATS                   *NATSConfig  `mapstructure:"nats,omitempty"`
	Kafka                  *KafkaConfig `mapstructure:"kafka,omitempty"`
	TelegramToken          string       `mapstructure:"telegram_token,omitempty"`
	RouteWorkers           int          `mapstructure:"route_workers"`
	PublishWorkers         int          `mapstructure:"publish_workers"`
	PublishShutdownTimeout int          `mapstructure:"publish_shutdown_timeout"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string, logger *slog.Logger) (*Config, error) {
	v := viper.New()

	// Read from environment variables (only for telegram_token and nats_url)
	// Subject is read only from YAML file
	v.BindEnv("telegram_token", "TELEGRAM_BOT_TOKEN")
	v.BindEnv("nats.url", "NATS_URL")

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

	// Handle KAFKA_BROKERS env variable manually (comma-separated string to slice)
	if brokersEnv := os.Getenv("KAFKA_BROKERS"); brokersEnv != "" {
		if cfg.Kafka == nil {
			cfg.Kafka = &KafkaConfig{}
		}
		cfg.Kafka.Brokers = splitBrokers(brokersEnv)
	}

	if cfg.Mode == "" {
		cfg.Mode = "first"
	}

	if cfg.Broker == "" {
		cfg.Broker = BrokerNATS
	}

	if cfg.Broker == BrokerNATS {
		if cfg.NATS == nil {
			cfg.NATS = &NATSConfig{}
		}
		if cfg.NATS.Engine == "" {
			cfg.NATS.Engine = EngineCore
		}
	}

	if cfg.Broker == BrokerKafka {
		if cfg.Kafka == nil {
			cfg.Kafka = &KafkaConfig{}
		}
		if cfg.Kafka.AckRequired == 0 {
			cfg.Kafka.AckRequired = -1
		}
		if cfg.Kafka.BatchBytes == 0 {
			cfg.Kafka.BatchBytes = 1048576
		}
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
		"broker", cfg.Broker,
		"routes_count", len(cfg.Routes),
		"has_telegram_token", cfg.TelegramToken != "",
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

	if c.Broker != BrokerNATS && c.Broker != BrokerKafka {
		return fmt.Errorf("broker must be 'nats' or 'kafka'")
	}

	if c.Broker == BrokerNATS {
		if c.NATS == nil {
			return fmt.Errorf("nats configuration is required when broker is 'nats'")
		}
		if c.NATS.URL == "" {
			return fmt.Errorf("nats.url is required (set NATS_URL env or nats.url in config)")
		}
		if c.NATS.Engine != EngineCore && c.NATS.Engine != EngineJetStream {
			return fmt.Errorf("nats.engine must be 'core' or 'jetstream'")
		}
		if c.NATS.Engine == EngineJetStream {
			if c.NATS.JetStream == nil {
				return fmt.Errorf("nats.jetstream configuration is required when engine is 'jetstream'")
			}
			if c.NATS.JetStream.StreamConfig == "" {
				return fmt.Errorf("nats.jetstream.stream_config is required when engine is 'jetstream'")
			}
			if _, err := os.Stat(c.NATS.JetStream.StreamConfig); os.IsNotExist(err) {
				return fmt.Errorf("nats.jetstream.stream_config file does not exist: %s", c.NATS.JetStream.StreamConfig)
			}
		}
	}

	if c.Broker == BrokerKafka {
		if c.Kafka == nil {
			return fmt.Errorf("kafka configuration is required when broker is 'kafka'")
		}
		if len(c.Kafka.Brokers) == 0 {
			return fmt.Errorf("kafka.brokers is required when broker is 'kafka'")
		}
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

		if c.Broker == BrokerNATS {
			if route.Subject == nil {
				return fmt.Errorf("routes[%d].subject is required when broker is 'nats'", i)
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

		if c.Broker == BrokerKafka {
			if route.Topic == nil {
				return fmt.Errorf("routes[%d].topic is required when broker is 'kafka'", i)
			}
			if route.Topic.Type == "" {
				return fmt.Errorf("routes[%d].topic.type is required", i)
			}
			if route.Topic.Value == "" {
				return fmt.Errorf("routes[%d].topic.value is required", i)
			}
			if route.Topic.Type != SubjectTypeString && route.Topic.Type != SubjectTypeExpr {
				return fmt.Errorf("routes[%d].topic.type must be 'string' or 'expr'", i)
			}
			if route.Key != nil {
				if route.Key.Type != SubjectTypeString && route.Key.Type != SubjectTypeExpr {
					return fmt.Errorf("routes[%d].key.type must be 'string' or 'expr'", i)
				}
			}
		}
	}

	if c.TelegramToken == "" {
		return fmt.Errorf("telegram token is required (set TELEGRAM_BOT_TOKEN env or telegram_token in config)")
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
