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

	// Clear environment variables that might interfere with token and URL
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("NATS_URL", "")

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
mode: first
routes:
  - condition: "update.message != nil"
    subject:
      type: string
      value: telegram.messages
telegram_token: test-token
nats_url: nats://test:4222
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := LoadConfig(configPath, logger)
	require.NoError(t, err)
	assert.Equal(t, "first", cfg.Mode)
	assert.Equal(t, 1, len(cfg.Routes))
	assert.Equal(t, "test-token", cfg.TelegramToken)
	assert.Equal(t, "nats://test:4222", cfg.NATSURL)
}

func TestLoadConfig_FromEnvOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create temporary config file with routes only
	// telegram_token and nats_url will come from env
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
mode: all
routes:
  - condition: "update.message != nil"
    subject:
      type: string
      value: telegram.messages
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Set environment variables for token and URL
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	t.Setenv("NATS_URL", "nats://env:4222")

	cfg, err := LoadConfig(configPath, logger)
	require.NoError(t, err)
	assert.Equal(t, "all", cfg.Mode)
	assert.Equal(t, 1, len(cfg.Routes))
	assert.Equal(t, "env-token", cfg.TelegramToken)
	assert.Equal(t, "nats://env:4222", cfg.NATSURL)
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
			name: "valid config",
			config: Config{
				Mode: "first",
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken: "test-token",
				NATSURL:       "nats://localhost:4222",
			},
			wantErr: false,
		},
		{
			name: "empty routes is valid",
			config: Config{
				Mode:          "first",
				Routes:        []Route{},
				TelegramToken: "test-token",
				NATSURL:       "nats://localhost:4222",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: Config{
				Mode: "invalid",
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken: "test-token",
				NATSURL:       "nats://localhost:4222",
			},
			wantErr: true,
			errMsg:  "mode must be 'first' or 'all'",
		},
		{
			name: "missing telegram token",
			config: Config{
				Mode: "first",
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				NATSURL: "nats://localhost:4222",
			},
			wantErr: true,
			errMsg:  "telegram token is required",
		},
		{
			name: "missing nats url",
			config: Config{
				Mode: "first",
				Routes: []Route{
					{
						Condition: "update.message != nil",
						Subject: RouteSubject{
							Type:  SubjectTypeString,
							Value: "telegram.messages",
						},
					},
				},
				TelegramToken: "test-token",
			},
			wantErr: true,
			errMsg:  "nats url is required",
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
