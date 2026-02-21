package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSClientInterface defines the interface for NATS client
type NATSClientInterface interface {
	// Connect establishes connection to NATS server
	Connect(ctx context.Context) error
	// Publish sends a message to the specified subject
	Publish(ctx context.Context, subject string, data interface{}) error
	// Close closes the NATS connection
	Close() error
}

// NATSClient implements NATSClientInterface
type NATSClient struct {
	url     string
	conn    *nats.Conn
	logger  *slog.Logger
	encoder *json.Encoder
}

// NewNATSClient creates a new NATS client
func NewNATSClient(url string, logger *slog.Logger) *NATSClient {
	return &NATSClient{
		url:    url,
		logger: logger,
	}
}

// Connect establishes connection to NATS server
func (c *NATSClient) Connect(ctx context.Context) error {
	c.logger.Info("connecting to NATS", "url", c.url)

	// Set connection timeout based on context or default
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 0
		}
	}

	opts := []nats.Option{
		nats.Name("telegram-nats-bridge"),
		nats.MaxReconnects(5),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			c.logger.Warn("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
	}

	conn, err := nats.Connect(c.url, opts...)
	if err != nil {
		c.logger.Error("failed to connect to NATS", "error", err)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	c.conn = conn
	c.logger.Info("connected to NATS", "server", conn.ConnectedUrl())
	return nil
}

// Publish sends a message to the specified subject
func (c *NATSClient) Publish(ctx context.Context, subject string, data interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("NATS connection is not established")
	}

	if c.conn.IsClosed() {
		return fmt.Errorf("NATS connection is closed")
	}

	// Marshal data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("failed to marshal data", "error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Check context cancellation before publishing
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := c.conn.Publish(subject, payload); err != nil {
		c.logger.Error("failed to publish message", "subject", subject, "error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Flush to ensure message is sent
	if err := c.conn.Flush(); err != nil {
		c.logger.Error("failed to flush NATS connection", "error", err)
		return fmt.Errorf("failed to flush: %w", err)
	}

	c.logger.Debug("message published", "subject", subject, "size", len(payload))
	return nil
}

// Close closes the NATS connection
func (c *NATSClient) Close() error {
	if c.conn == nil {
		return nil
	}

	c.logger.Info("closing NATS connection")
	c.conn.Close()
	c.logger.Info("NATS connection closed")
	return nil
}

// IsConnected returns true if client is connected
func (c *NATSClient) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Ensure NATSClient implements NATSClientInterface
var _ NATSClientInterface = (*NATSClient)(nil)

type publishTask struct {
	subject string
	data    interface{}
}

type Publisher struct {
	workers    int
	timeoutSec int
	tasks      chan publishTask
	natsClient NATSClientInterface
	logger     *slog.Logger
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewPublisher(workers, timeoutSec int, natsClient NATSClientInterface, logger *slog.Logger) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Publisher{
		workers:    workers,
		timeoutSec: timeoutSec,
		tasks:      make(chan publishTask, workers*2),
		natsClient: natsClient,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (p *Publisher) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	p.logger.Info("publisher started", "workers", p.workers)
}

func (p *Publisher) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			p.publishTask(task)
		}
	}
}

func (p *Publisher) publishTask(task publishTask) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	if err := p.natsClient.Publish(ctx, task.subject, task.data); err != nil {
		p.logger.Error("failed to publish message", "subject", task.subject, "error", err)
	}
}

func (p *Publisher) Publish(subject string, data interface{}) {
	select {
	case <-p.ctx.Done():
		return
	case p.tasks <- publishTask{subject: subject, data: data}:
	}
}

func (p *Publisher) Close() {
	p.cancel()
	close(p.tasks)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("publisher closed")
	case <-time.After(time.Duration(p.timeoutSec) * time.Second):
		p.logger.Warn("publisher close timeout", "timeout_sec", p.timeoutSec)
	}
}
