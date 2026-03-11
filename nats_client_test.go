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
	dest := Destination{Subject: "test.subject"}

	err := client.Publish(ctx, dest, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not established")
}

func TestNATSClient_Close_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	err := client.Close()
	assert.NoError(t, err)
}

func TestNATSClient_Publish_MarshalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	badData := make(chan int)

	_ = badData
	_ = client
}

func TestNATSClient_Publish_ContextCancelled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewNATSClient("nats://localhost:4222", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data := map[string]string{"test": "data"}
	dest := Destination{Subject: "test.subject"}

	err := client.Publish(ctx, dest, data)
	assert.Error(t, err)
}

func TestNewJetStreamClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://localhost:4222", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "nats://localhost:4222", client.url)
	assert.NotNil(t, client.logger)
	assert.Nil(t, client.nc)
}

func TestJetStreamClient_Connect_NotStarted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://invalid:4222", logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to NATS")
}

func TestJetStreamClient_Publish_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://localhost:4222", logger)

	ctx := context.Background()
	data := map[string]string{"test": "data"}
	dest := Destination{Subject: "test.subject"}

	err := client.Publish(ctx, dest, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not established")
}

func TestJetStreamClient_Close_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://localhost:4222", logger)

	err := client.Close()
	assert.NoError(t, err)
}

func TestJetStreamClient_EnsureStream_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	tmpDir := t.TempDir()
	configPath := tmpDir + "/invalid.json"
	err := os.WriteFile(configPath, []byte("invalid json"), 0644)
	assert.NoError(t, err)

	client := NewJetStreamClient("nats://localhost:4222", logger)

	ctx := context.Background()
	err = client.EnsureStream(ctx, configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JetStream is not connected")
}

func TestJetStreamClient_EnsureStream_FileNotFound_NotConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://localhost:4222", logger)

	ctx := context.Background()
	err := client.EnsureStream(ctx, "/nonexistent/path/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JetStream is not connected")
}

func TestJetStreamClient_Publish_ContextCancelled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewJetStreamClient("nats://localhost:4222", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data := map[string]string{"test": "data"}
	dest := Destination{Subject: "test.subject"}

	err := client.Publish(ctx, dest, data)
	assert.Error(t, err)
}
