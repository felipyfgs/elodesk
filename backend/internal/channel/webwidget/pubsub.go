package webwidget

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type PubsubService struct {
	redisClient *redis.Client
}

func NewPubsubService(redisClient *redis.Client) *PubsubService {
	return &PubsubService{redisClient: redisClient}
}

func (s *PubsubService) Publish(ctx context.Context, pubsubToken string, eventType string, data any) error {
	channel := widgetPubsubPrefix + pubsubToken

	payload, err := json.Marshal(map[string]any{
		"type": eventType,
		"data": data,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal pubsub payload: %w", err)
	}

	if err := s.redisClient.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish to redis: %w", err)
	}

	return nil
}

func (s *PubsubService) PublishMessageCreated(ctx context.Context, pubsubToken string, message any) error {
	return s.Publish(ctx, pubsubToken, "message.created", message)
}
