package main

import (
	"log/slog"
	"os"
	"testing"

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
				Condition: "update.message != nil",
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
				Condition: "update.message.!!!",
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
				Condition: "update.message != nil",
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
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.callback_query != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.callbacks",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"message": map[string]any{
				"text": "hello",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.messages"}}, dests)
	})

	t.Run("mode first - second match", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.callback_query != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.callbacks",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"callback_query": map[string]any{
				"data": "test",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.callbacks"}}, dests)
	})

	t.Run("mode all - multiple matches", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
			{
				Condition: "update.message.text != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.texts",
				},
			},
		}
		router, err := NewRouter(routes, "all", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"message": map[string]any{
				"text": "hello",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Subject: "telegram.messages"}, {Subject: "telegram.texts"}}, dests)
	})

	t.Run("no match - empty result", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"callback_query": map[string]any{
				"data": "test",
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Empty(t, dests)
	})

	t.Run("dynamic subject with expr", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeExpr,
					Value: "sprintf(\"telegram.%d.messages\", update.message.from.id)",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"message": map[string]any{
				"text": "hello",
				"from": map[string]any{
					"id": 12345,
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
				Condition: "update.message != nil",
				Subject: &RouteSubject{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Empty(t, dests)
	})

	t.Run("kafka routes with topic and key", func(t *testing.T) {
		routes := []Route{
			{
				Condition: "update.message != nil",
				Topic: &RouteTopic{
					Type:  SubjectTypeString,
					Value: "telegram.messages",
				},
				Key: &RouteKey{
					Type:  SubjectTypeExpr,
					Value: "sprintf(\"%v\", update.message.from.id)",
				},
			},
		}
		router, err := NewRouter(routes, "first", 5, logger)
		require.NoError(t, err)

		update := Update{
			"update_id": 1,
			"message": map[string]any{
				"text": "hello",
				"from": map[string]any{
					"id": 12345,
				},
			},
		}

		dests, err := router.Route(update)
		require.NoError(t, err)
		assert.Equal(t, []Destination{{Topic: "telegram.messages", Key: "12345"}}, dests)
	})
}
