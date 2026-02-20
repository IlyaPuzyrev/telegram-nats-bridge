package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// getLogLevel returns slog.Level from LOG_LEVEL env variable, defaults to WARN
func getLogLevel() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	switch levelStr {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "telegram-nats-bridge",
		Short: "Bridge between Telegram Bot API and NATS",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the bridge",
		Run:   runBridge,
	}
	runCmd.Flags().String("config", "", "Path to configuration file (required)")

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check utilities",
	}

	checkBotCmd := &cobra.Command{
		Use:   "bot",
		Short: "Check bot connection and print updates as JSON",
		RunE:  checkBot,
	}
	checkBotCmd.Flags().String("config", "", "Path to configuration file (required)")

	checkCmd.AddCommand(checkBotCmd)
	rootCmd.AddCommand(runCmd, checkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBridge(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))

	// Get config path from flag
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		logger.Error("failed to get config flag", "error", err)
		os.Exit(1)
	}

	if configPath == "" {
		logger.Error("--config flag is required")
		os.Exit(1)
	}

	// Validate config path
	if err := ValidateConfigPath(configPath); err != nil {
		logger.Error("invalid config path", "error", err)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := LoadConfig(configPath, logger)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Get Telegram token from config (loaded from env or YAML)
	token := cfg.TelegramToken

	// Create Telegram client
	tgClient := NewTelegramClient(token, logger)

	// Test: Get bot info
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	botInfo, err := tgClient.GetMe(ctx)
	if err != nil {
		logger.Error("failed to get bot info", "error", err)
		os.Exit(1)
	}

	logger.Info("bot connected",
		"id", botInfo.ID,
		"username", botInfo.Username,
		"name", botInfo.FirstName)

	// Create and connect NATS client
	natsClient := NewNATSClient(cfg.NATSURL, logger)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := natsClient.Connect(ctx); err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	logger.Info("NATS connected", "url", cfg.NATSURL)

	// Create router
	router, err := NewRouter(cfg.Routes, cfg.Mode, logger)
	if err != nil {
		logger.Error("failed to create router", "error", err)
		os.Exit(1)
	}

	// Start polling for updates
	logger.Info("starting to poll for updates...")

	// Setup graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down...")
		cancel()
	}()

	// Poll for updates and publish to NATS
	var offset int64 = 0
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown complete")
			return
		default:
		}

		updates, nextOffset, err := tgClient.GetUpdates(ctx, offset)
		if err != nil {
			// Check if this is a graceful shutdown
			select {
			case <-ctx.Done():
				logger.Info("shutdown complete")
				return
			default:
			}
			logger.Error("failed to get updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			var updateID int64
			if idNum, ok := update["update_id"].(json.Number); ok {
				updateID, _ = idNum.Int64()
			}
			_, hasMessage := update["message"]
			logger.Info("received update",
				"update_id", updateID,
				"has_message", hasMessage)

			// Route update to NATS subjects
			subjects, err := router.Route(update)
			if err != nil {
				logger.Error("failed to route update", "error", err, "update_id", updateID)
				continue
			}

			for subject := range subjects {
				if err := natsClient.Publish(ctx, subject, update); err != nil {
					logger.Error("failed to publish update to NATS", "error", err, "update_id", updateID, "subject", subject)
				}
			}
		}

		// Update offset for next poll
		offset = nextOffset

		if len(updates) == 0 {
			// No updates, short sleep before next poll
			time.Sleep(1 * time.Second)
		}
	}
}

func checkBot(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))

	// Get config path from flag
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		logger.Error("failed to get config flag", "error", err)
		return fmt.Errorf("failed to get config flag: %w", err)
	}

	if configPath == "" {
		logger.Error("--config flag is required")
		return fmt.Errorf("--config flag is required")
	}

	// Validate config path
	if err := ValidateConfigPath(configPath); err != nil {
		logger.Error("invalid config path", "error", err)
		return fmt.Errorf("invalid config path: %w", err)
	}

	// Load configuration
	cfg, err := LoadConfig(configPath, logger)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Telegram client
	client := NewTelegramClient(cfg.TelegramToken, logger)

	// Test: Get bot info
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	botInfo, err := client.GetMe(ctx)
	if err != nil {
		logger.Error("failed to get bot info", "error", err)
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	logger.Info("bot connected", "username", botInfo.Username, "id", botInfo.ID)
	logger.Info("send a message to the bot to see JSON output, press Ctrl+C to exit")

	// Setup graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down...")
		cancel()
	}()

	// Poll for updates and output as JSON
	var offset int64 = 0
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		updates, nextOffset, err := client.GetUpdates(ctx, offset)
		if err != nil {
			// Check if this is a graceful shutdown
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			logger.Error("failed to get updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			// Output update as JSON
			if err := encoder.Encode(update); err != nil {
				logger.Error("failed to encode update", "error", err)
			}
			fmt.Println() // Empty line between updates
		}

		// Update offset for next poll
		offset = nextOffset

		if len(updates) == 0 {
			// No updates, short sleep before next poll
			time.Sleep(1 * time.Second)
		}
	}
}
