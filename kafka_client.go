package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	brokers []string
	logger  *slog.Logger
	writer  *kafka.Writer
	wg      sync.WaitGroup
	closed  bool
	mu      sync.Mutex
}

type KafkaClientConfig struct {
	Brokers     []string
	Async       bool
	AckRequired int
	BatchSize   int
	BatchBytes  int64
}

func NewKafkaClient(cfg KafkaClientConfig, logger *slog.Logger) *KafkaClient {
	var requiredAcks kafka.RequiredAcks
	switch cfg.AckRequired {
	case 0:
		requiredAcks = kafka.RequireNone
	case 1:
		requiredAcks = kafka.RequireOne
	default:
		requiredAcks = kafka.RequireAll
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Async:        cfg.Async,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchBytes:   cfg.BatchBytes,
		RequiredAcks: requiredAcks,
	}

	return &KafkaClient{
		brokers: cfg.Brokers,
		logger:  logger,
		writer:  writer,
	}
}

func (c *KafkaClient) Connect(ctx context.Context) error {
	c.logger.Info("connecting to Kafka", "brokers", c.brokers)

	dialer := &kafka.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.brokers[0])
	if err != nil {
		c.logger.Error("failed to connect to Kafka", "broker", c.brokers[0], "error", err)
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	_, err = conn.Brokers()
	if err != nil {
		c.logger.Error("failed to get brokers", "error", err)
		return fmt.Errorf("failed to get brokers: %w", err)
	}

	c.logger.Info("connected to Kafka", "brokers", c.brokers)
	return nil
}

func (c *KafkaClient) Publish(ctx context.Context, dest Destination, data interface{}) error {
	if dest.Topic == "" {
		return fmt.Errorf("Kafka topic is required")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("failed to marshal data", "error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	var key []byte
	if dest.Key != "" {
		key = []byte(dest.Key)
	}

	msg := kafka.Message{
		Topic: dest.Topic,
		Key:   key,
		Value: payload,
	}

	err = c.writer.WriteMessages(ctx, msg)
	if err != nil {
		c.logger.Error("failed to publish message", "topic", dest.Topic, "key", dest.Key, "error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Debug("message published", "topic", dest.Topic, "key", dest.Key, "size", len(payload))
	return nil
}

func (c *KafkaClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	c.logger.Info("closing Kafka writer")
	c.writer.Close()
	c.logger.Info("Kafka writer closed")
	return nil
}

var _ BrokerInterface = (*KafkaClient)(nil)
