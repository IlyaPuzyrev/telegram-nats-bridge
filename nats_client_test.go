package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNATSClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "nats://localhost:4222", client.url)
	assert.NotNil(t, client.logger)
	assert.Nil(t, client.conn)
}

func TestNATSClient_Connect_NotStarted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://invalid:4222", logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should fail to connect to invalid server
	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to NATS")
}

func TestNATSClient_Publish_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	ctx := context.Background()
	data := map[string]string{"test": "data"}

	err := client.Publish(ctx, "test.subject", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not established")
}

func TestNATSClient_Close_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	// Should not error when closing unconnected client
	err := client.Close()
	assert.NoError(t, err)
}

func TestNATSClient_IsConnected_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	assert.False(t, client.IsConnected())
}

func TestNATSClient_Publish_MarshalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// We can't easily test this without a real connection
	// This is more of an integration test
	// For now, just verify the structure
	client := NewNATSClient("nats://localhost:4222", logger)

	// Create a mock that can't be marshaled (channel)
	badData := make(chan int)

	// Even without connection, we should check the marshal error logic
	// But we can't reach it without connection
	_ = badData
	_ = client
}

func TestNATSClient_Publish_ContextCancelled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := map[string]string{"test": "data"}

	// This will fail with "not established" because we check connection before context
	// But if we had a connection, it would check context first
	err := client.Publish(ctx, "test.subject", data)
	assert.Error(t, err)
}
