package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSClient implements BrokerInterface
type NATSClient struct {
	url    string
	conn   *nats.Conn
	logger *slog.Logger
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
func (c *NATSClient) Publish(ctx context.Context, dest Destination, data interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("NATS connection is not established")
	}

	if c.conn.IsClosed() {
		return fmt.Errorf("NATS connection is closed")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("failed to marshal data", "error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := c.conn.Publish(dest.Subject, payload); err != nil {
		c.logger.Error("failed to publish message", "subject", dest.Subject, "error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	if err := c.conn.Flush(); err != nil {
		c.logger.Error("failed to flush NATS connection", "error", err)
		return fmt.Errorf("failed to flush: %w", err)
	}

	c.logger.Debug("message published", "subject", dest.Subject, "size", len(payload))
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

// Ensure NATSClient implements BrokerInterface
var _ BrokerInterface = (*NATSClient)(nil)

// JetStreamClient implements BrokerInterface for JetStream
type JetStreamClient struct {
	url    string
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *slog.Logger
}

// NewJetStreamClient creates a new JetStream client
func NewJetStreamClient(url string, logger *slog.Logger) *JetStreamClient {
	return &JetStreamClient{
		url:    url,
		logger: logger,
	}
}

// Connect establishes connection to NATS server with JetStream
func (c *JetStreamClient) Connect(ctx context.Context) error {
	c.logger.Info("connecting to NATS with JetStream", "url", c.url)

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

	nc, err := nats.Connect(c.url, opts...)
	if err != nil {
		c.logger.Error("failed to connect to NATS", "error", err)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		c.logger.Error("failed to create JetStream context", "error", err)
		nc.Close()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	c.nc = nc
	c.js = js
	c.logger.Info("connected to NATS with JetStream", "server", nc.ConnectedUrl())
	return nil
}

// Publish sends a message to the specified subject via JetStream
func (c *JetStreamClient) Publish(ctx context.Context, dest Destination, data interface{}) error {
	if c.nc == nil {
		return fmt.Errorf("NATS connection is not established")
	}

	if c.nc.IsClosed() {
		return fmt.Errorf("NATS connection is closed")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("failed to marshal data", "error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, err = c.js.Publish(ctx, dest.Subject, payload)
	if err != nil {
		c.logger.Error("failed to publish message", "subject", dest.Subject, "error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Debug("message published via JetStream", "subject", dest.Subject, "size", len(payload))
	return nil
}

// Close closes the NATS connection
func (c *JetStreamClient) Close() error {
	if c.nc == nil {
		return nil
	}

	c.logger.Info("closing NATS connection")
	c.nc.Close()
	c.logger.Info("NATS connection closed")
	return nil
}

// Ensure JetStreamClient implements BrokerInterface
var _ BrokerInterface = (*JetStreamClient)(nil)

// EnsureStream creates or updates a JetStream stream based on config file
func (c *JetStreamClient) EnsureStream(ctx context.Context, configPath string) error {
	if c.js == nil {
		return fmt.Errorf("JetStream is not connected")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		c.logger.Error("failed to read stream config file", "path", configPath, "error", err)
		return fmt.Errorf("failed to read stream config file: %w", err)
	}

	var streamCfg jetstream.StreamConfig
	if err := json.Unmarshal(configData, &streamCfg); err != nil {
		c.logger.Error("failed to parse stream config", "path", configPath, "error", err)
		return fmt.Errorf("failed to parse stream config: %w", err)
	}

	_, err = c.js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		c.logger.Error("failed to create/update stream", "name", streamCfg.Name, "error", err)
		return fmt.Errorf("failed to create/update stream: %w", err)
	}

	c.logger.Info("stream created/updated", "name", streamCfg.Name)
	return nil
}

type publishTask struct {
	dest Destination
	data interface{}
}

type Publisher struct {
	workers      int
	timeoutSec   int
	tasks        chan publishTask
	brokerClient BrokerInterface
	logger       *slog.Logger
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewPublisher(workers, timeoutSec int, brokerClient BrokerInterface, logger *slog.Logger) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Publisher{
		workers:      workers,
		timeoutSec:   timeoutSec,
		tasks:        make(chan publishTask, workers*2),
		brokerClient: brokerClient,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
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

	if err := p.brokerClient.Publish(ctx, task.dest, task.data); err != nil {
		p.logger.Error("failed to publish message", "destination", task.dest, "error", err)
	}
}

func (p *Publisher) Publish(dest Destination, data interface{}) {
	select {
	case <-p.ctx.Done():
		return
	case p.tasks <- publishTask{dest: dest, data: data}:
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
