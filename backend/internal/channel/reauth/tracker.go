package reauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultThreshold = 3
	defaultTTL       = 1 * time.Hour
)

type Tracker struct {
	client    redis.Cmdable
	threshold int
	ttl       time.Duration
	keyPrefix string
}

func NewTracker(client redis.Cmdable) *Tracker {
	return &Tracker{
		client:    client,
		threshold: defaultThreshold,
		ttl:       defaultTTL,
		keyPrefix: "elodesk:reauth:",
	}
}

func (t *Tracker) RecordError(ctx context.Context, key string) (promptReauth bool, err error) {
	k := t.keyPrefix + key
	count, err := t.client.Incr(ctx, k).Result()
	if err != nil {
		return false, fmt.Errorf("reauth incr: %w", err)
	}
	if count == 1 {
		_ = t.client.Expire(ctx, k, t.ttl)
	}
	return count >= int64(t.threshold), nil
}

func (t *Tracker) Reset(ctx context.Context, key string) error {
	k := t.keyPrefix + key
	_, err := t.client.Del(ctx, k).Result()
	if err != nil {
		return fmt.Errorf("reauth reset: %w", err)
	}
	return nil
}

func (t *Tracker) ShouldPrompt(ctx context.Context, key string) (bool, error) {
	k := t.keyPrefix + key
	count, err := t.client.Get(ctx, k).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reauth get: %w", err)
	}
	return count >= t.threshold, nil
}
