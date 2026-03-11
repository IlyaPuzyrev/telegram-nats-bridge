package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_WithFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("KAFKA_BROKERS", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
broker: nats
mode: first
nats:
  url: nats://test:4222
  engine: core
routes:
  - condition: "update.message != nil"
    subject:
      type: string
      value: telegram.messages
telegram_token: test-token
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := LoadConfig(configPath, logger)
	require.NoError(t, err)
	assert.Equal(t, "first", cfg.Mode)
	assert.Equal(t, BrokerNATS, cfg.Broker)
	assert.Equal(t, 1, len(cfg.Routes))
	assert.Equal(t, "test-token", cfg.TelegramToken)
	assert.Equal(t, "nats://test:4222", cfg.NATS.URL)
}

func TestLoadConfig_FromEnvOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	t.Setenv("NATS_URL", "nats://env:4222")
	t.Setenv("KAFKA_BROKERS", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
broker: nats
mode: all
nats:
  url: nats://env:4222
  engine: core
routes:
  - condition: "update.message != nil"
    subject:
      type: string
      value: telegram.messages
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token")

	cfg, err := LoadConfig(configPath, logger)
	require.NoError(t, err)
	assert.Equal(t, "all", cfg.Mode)
	assert.Equal(t, 1, len(cfg.Routes))
	assert.Equal(t, "env-token", cfg.TelegramToken)
	assert.Equal(t, "nats://env:4222", cfg.NATS.URL)
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	_, err := LoadConfig("/nonexistent/config.yaml", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid nats config",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: &RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: false,
		},
		{
			name: "valid kafka config",
			config: Config{
				Mode:   "first",
				Broker: BrokerKafka,
				Kafka: &KafkaConfig{
					Brokers: []string{"localhost:9092"},
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Topic: &RouteTopic{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: false,
		},
		{
			name: "empty routes is valid",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes:                 []Route{},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: Config{
				Mode:   "invalid",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: &RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "mode must be 'first' or 'all'",
		},
		{
			name: "valid config with broker nats",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: &RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: false,
		},
		{
			name: "missing telegram token",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: &RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "telegram token is required",
		},
		{
			name: "missing nats config when broker is nats",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: &RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "nats configuration is required",
		},
		{
			name: "missing kafka config when broker is kafka",
			config: Config{
				Mode:   "first",
				Broker: BrokerKafka,
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Topic: &RouteTopic{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "kafka configuration is required",
		},
		{
			name: "invalid route_workers",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes:                 []Route{},
				TelegramToken:          "test-token",
				RouteWorkers:           0,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "route_workers must be > 0",
		},
		{
			name: "invalid publish_workers",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes:                 []Route{},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         -1,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "publish_workers must be > 0",
		},
		{
			name: "invalid publish_shutdown_timeout",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes:                 []Route{},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 0,
			},
			wantErr: true,
			errMsg:  "publish_shutdown_timeout must be > 0",
		},
		{
			name: "nats route missing subject",
			config: Config{
				Mode:   "first",
				Broker: BrokerNATS,
				NATS: &NATSConfig{
					URL:    "nats://localhost:4222",
					Engine: EngineCore,
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "subject is required when broker is 'nats'",
		},
		{
			name: "kafka route missing topic",
			config: Config{
				Mode:   "first",
				Broker: BrokerKafka,
				Kafka: &KafkaConfig{
					Brokers: []string{"localhost:9092"},
				},
				Routes: []Route{
					{
						Condition: "update.message != nil",
					},
				},
				TelegramToken:          "test-token",
				RouteWorkers:           5,
				PublishWorkers:         5,
				PublishShutdownTimeout: 10,
			},
			wantErr: true,
			errMsg:  "topic is required when broker is 'kafka'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "config path is required",
		},
		{
			name:    "non-existent file",
			path:    "/nonexistent/config.yaml",
			wantErr: true,
			errMsg:  "does not exist",
		},
		{
			name:    "directory instead of file",
			path:    t.TempDir(),
			wantErr: true,
			errMsg:  "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigPath_InvalidExtension(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))

	err := ValidateConfigPath(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a YAML file")
}

func TestValidateConfigPath_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: first\nroutes: []"), 0644))

	err := ValidateConfigPath(configPath)
	assert.NoError(t, err)
}
