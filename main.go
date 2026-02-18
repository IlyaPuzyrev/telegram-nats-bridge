package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

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

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBridge(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Get Telegram bot token from environment
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		logger.Error("TELEGRAM_BOT_TOKEN environment variable is required")
		os.Exit(1)
	}

	// Create Telegram client
	client := NewTelegramClient(token, logger)

	// Test: Get bot info
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	botInfo, err := client.GetMe(ctx)
	if err != nil {
		logger.Error("failed to get bot info", "error", err)
		os.Exit(1)
	}

	logger.Info("bot connected",
		"id", botInfo.ID,
		"username", botInfo.Username,
		"name", botInfo.FirstName)

	// Test: Get updates
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

	// Poll for updates
	var offset int64 = 0
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown complete")
			return
		default:
		}

		updates, err := client.GetUpdates(ctx, offset)
		if err != nil {
			logger.Error("failed to get updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			logger.Info("received update",
				"update_id", update.UpdateID,
				"has_message", update.Message != nil)

			if update.Message != nil && update.Message.Text != "" {
				logger.Info("message received",
					"chat_id", update.Message.Chat.ID,
					"from", update.Message.From.Username,
					"text", update.Message.Text)
			}

			// Update offset to get next batch
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
		}

		if len(updates) == 0 {
			// No updates, short sleep before next poll
			time.Sleep(1 * time.Second)
		}
	}
}
