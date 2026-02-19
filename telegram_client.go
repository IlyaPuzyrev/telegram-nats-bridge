package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-resty/resty/v2"
)

// TelegramClientInterface defines the interface for Telegram Bot API client
type TelegramClientInterface interface {
	// GetUpdates retrieves updates from Telegram with specified offset
	// Returns updates, next offset (max update_id + 1), and error
	GetUpdates(ctx context.Context, offset int64) ([]Update, int64, error)
	// GetUpdatesWithTimeout retrieves updates with custom timeout for long polling
	// Returns updates, next offset (max update_id + 1), and error
	GetUpdatesWithTimeout(ctx context.Context, offset int64, timeout int) ([]Update, int64, error)
	// GetBotInfo retrieves information about the bot
	GetBotInfo(ctx context.Context) (*User, error)
	// GetMe is alias for GetBotInfo
	GetMe(ctx context.Context) (*User, error)
}

// TelegramClient implements TelegramClientInterface
type TelegramClient struct {
	client  *resty.Client
	baseURL string
	token   string
	logger  *slog.Logger
}

// NewTelegramClient creates a new Telegram client
func NewTelegramClient(token string, logger *slog.Logger) *TelegramClient {
	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s", token)

	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(60 * time.Second)

	return &TelegramClient{
		client:  client,
		baseURL: baseURL,
		token:   token,
		logger:  logger,
	}
}

// GetUpdates retrieves updates from Telegram
// offset - identifier of the first update to be returned
// Returns updates, next offset (max update_id + 1), and nil error on success
func (c *TelegramClient) GetUpdates(ctx context.Context, offset int64) ([]Update, int64, error) {
	return c.GetUpdatesWithTimeout(ctx, offset, 30)
}

// GetUpdatesWithTimeout retrieves updates with specified timeout for long polling
// timeout - timeout in seconds for long polling (0 for short polling)
// Returns updates, next offset (max update_id + 1), and error
func (c *TelegramClient) GetUpdatesWithTimeout(ctx context.Context, offset int64, timeout int) ([]Update, int64, error) {
	c.logger.Debug("getting updates from Telegram",
		"offset", offset,
		"timeout", timeout)

	// Create timeout context for long polling (timeout + buffer)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout+10)*time.Second)
		defer cancel()
	}

	req := c.client.R().
		SetContext(ctx).
		SetQueryParam("limit", "100")

	if offset > 0 {
		req.SetQueryParam("offset", fmt.Sprintf("%d", offset))
	}

	if timeout > 0 {
		req.SetQueryParam("timeout", fmt.Sprintf("%d", timeout))
	}

	resp, err := req.Get("/getUpdates")

	if err != nil {
		// Don't treat context cancellation as an error
		if errors.Is(err, context.Canceled) {
			c.logger.Debug("getUpdates cancelled")
			return nil, offset, nil
		}
		c.logger.Error("failed to get updates", "error", err)
		return nil, offset, fmt.Errorf("failed to get updates: %w", err)
	}

	if resp.IsError() {
		c.logger.Error("telegram API error",
			"status", resp.StatusCode(),
			"body", string(resp.Body()))
		return nil, offset, fmt.Errorf("telegram API error: status %d", resp.StatusCode())
	}

	// Parse JSON response with UseNumber to preserve integer precision
	var response struct {
		Ok          bool     `json:"ok"`
		Result      []Update `json:"result,omitempty"`
		ErrorCode   int      `json:"error_code,omitempty"`
		Description string   `json:"description,omitempty"`
	}

	decoder := json.NewDecoder(bytes.NewReader(resp.Body()))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		c.logger.Error("failed to decode response", "error", err)
		return nil, offset, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Ok {
		c.logger.Error("telegram API returned error",
			"error_code", response.ErrorCode,
			"description", response.Description)
		return nil, offset, fmt.Errorf("telegram API error %d: %s",
			response.ErrorCode, response.Description)
	}

	c.logger.Debug("received updates", "count", len(response.Result))

	// Calculate next offset (max update_id + 1)
	nextOffset := offset
	for _, update := range response.Result {
		if updateID, ok := update["update_id"].(json.Number); ok {
			if id, err := updateID.Int64(); err == nil && id >= nextOffset {
				nextOffset = id + 1
			}
		}
	}

	return response.Result, nextOffset, nil
}

// GetBotInfo retrieves information about the bot
func (c *TelegramClient) GetBotInfo(ctx context.Context) (*User, error) {
	return c.GetMe(ctx)
}

// GetMe retrieves information about the bot (alias for GetBotInfo)
func (c *TelegramClient) GetMe(ctx context.Context) (*User, error) {
	c.logger.Debug("getting bot info")

	type getMeResponse struct {
		Ok          bool   `json:"ok"`
		Result      *User  `json:"result,omitempty"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var response getMeResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		Get("/getMe")

	if err != nil {
		// Don't log context cancellation as an error
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		c.logger.Error("failed to get bot info", "error", err)
		return nil, fmt.Errorf("failed to get bot info: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("telegram API error: status %d", resp.StatusCode())
	}

	if !response.Ok {
		return nil, fmt.Errorf("telegram API error %d: %s",
			response.ErrorCode, response.Description)
	}

	c.logger.Info("bot info retrieved",
		"id", response.Result.ID,
		"username", response.Result.Username,
		"first_name", response.Result.FirstName)

	return response.Result, nil
}

// Ensure TelegramClient implements TelegramClientInterface
var _ TelegramClientInterface = (*TelegramClient)(nil)
