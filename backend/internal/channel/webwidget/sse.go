package webwidget

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"backend/internal/logger"
	"backend/internal/repo"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

const (
	sseKeepaliveInterval = 30 * time.Second
	widgetPubsubPrefix   = "widget:pubsub:"
)

type SSEHandler struct {
	redisClient      *redis.Client
	conversationRepo *repo.ConversationRepo
	widgetRepo       *repo.ChannelWebWidgetRepo
}

func NewSSEHandler(
	redisClient *redis.Client,
	conversationRepo *repo.ConversationRepo,
	widgetRepo *repo.ChannelWebWidgetRepo,
) *SSEHandler {
	return &SSEHandler{
		redisClient:      redisClient,
		conversationRepo: conversationRepo,
		widgetRepo:       widgetRepo,
	}
}

func (h *SSEHandler) HandleSSE(c *fiber.Ctx) error {
	pubsubToken := c.Query("pubsubToken")
	if pubsubToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "pubsub_token is required"})
	}

	websiteToken := c.Params("websiteToken")
	if websiteToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "website_token is required"})
	}

	_, err := h.widgetRepo.FindByWebsiteToken(c.Context(), websiteToken)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "widget not found"})
	}

	conversation, err := h.conversationRepo.FindByPubsubToken(c.Context(), pubsubToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid pubsub_token"})
	}

	if conversation.PubsubToken == nil || *conversation.PubsubToken != pubsubToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "pubsub_token mismatch"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ctx := context.Background()
		sub := h.redisClient.Subscribe(ctx, widgetPubsubPrefix+pubsubToken)
		defer func() { _ = sub.Close() }()

		ch := sub.Channel()
		keepalive := time.NewTicker(sseKeepaliveInterval)
		defer keepalive.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg.Payload); err != nil {
					logger.Info().Str("component", "webwidget.sse").Err(err).Msg("failed to write SSE event")
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-keepalive.C:
				if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	return nil
}
