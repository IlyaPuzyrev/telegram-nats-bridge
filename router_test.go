package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Run("valid routes", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)
		assert.NotNil(t, router)
	})

	t.Run("invalid condition expr", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message.!!!",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		_, err := NewRouter(routes, "first", 5, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compile condition")
	})

	t.Run("invalid subject expr", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeExpr,
					Value: "sprintf(!!!)",
				},
			},
		}
		_, err := NewRouter(routes, "first", 5, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compile subject expression")
	})

	t.Run("empty routes", func(t *testing.T) {
		router, err := NewRouter([]Route{}, "first", 5, logger)
		require.NoError(t, err)
		assert.NotNil(t, router)
		assert.Empty(t, router.routes)
	})
}

func TestRouter_Route(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Run("mode first - first match", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.CallbackQuery != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.callbacks",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			Message: &gotgbot.Message{
				Text: "hello",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.messages"}}, dests)
	})

	t.Run("mode first - second match", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.CallbackQuery != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.callbacks",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			CallbackQuery: &gotgbot.CallbackQuery{
				Id:   "123",
				Data: "test",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.callbacks"}}, dests)
	})

	t.Run("mode all - multiple matches", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.Message.Text != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.texts",
				},
			},
		}
		router, err := NewRouter(routes, "all", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			Message: &gotgbot.Message{
				Text: "hello",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.messages"}, {Subject: "telegram.texts"}}, dests)
	})

	t.Run("no match - empty result", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			CallbackQuery: &gotgbot.CallbackQuery{
				Id:   "123",
				Data: "test",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Empty(t, dests)
	})

	t.Run("dynamic subject with expr", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeExpr,
					Value: "sprintf(\"telegram.%d.messages\", update.Message.From.Id)",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			Message: &gotgbot.Message{
				Text: "hello",
				From: &gotgbot.User{
					Id: 12345,
				},
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.12345.messages"}}, dests)
	})

	t.Run("empty update", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Empty(t, dests)
	})

	t.Run("kafka routes with topic and key", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.Message != nil",
				Topic: &RouteTopic{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
				Key: &RouteKey{
					Type:  SubjectTypeExpr,
					Value: "sprintf(\"%v\", update.Message.From.Id)",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := gotgbot.Update{
			UpdateId: 1,
			Message: &gotgbot.Message{
				Text: "hello",
				From: &gotgbot.User{
					Id: 12345,
				},
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Topic: "telegram.messages", Key: "12345"}}, dests)
	})
}
