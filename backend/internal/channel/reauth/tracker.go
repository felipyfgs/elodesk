package reauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
)

const (
	defaultThreshold = 3
	defaultTTL       = 1 * time.Hour
)

// kindThresholds overrides the default threshold for channels that are known
// to be sensitive to auth failures (OAuth flows where a single 401 usually
// means the token is gone) or tolerant (email IMAP that transient-fails a lot).
var kindThresholds = map[channel.Kind]int{
	channel.KindInstagram: 1,
	channel.KindTiktok:    1,
	channel.KindTwitter:   1,
}

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

// thresholdFor returns the error-count threshold at which the given kind is
// considered in need of reauth. Kinds without an override use the global
// default.
func (t *Tracker) thresholdFor(kind channel.Kind) int {
	if v, ok := kindThresholds[kind]; ok {
		return v
	}
	return t.threshold
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

// RecordErrorForKind is the kind-aware variant of RecordError. Callers that
// know the channel kind should prefer this so per-kind thresholds apply.
func (t *Tracker) RecordErrorForKind(ctx context.Context, kind channel.Kind, key string) (promptReauth bool, err error) {
	k := t.keyPrefix + key
	count, err := t.client.Incr(ctx, k).Result()
	if err != nil {
		return false, fmt.Errorf("reauth incr: %w", err)
	}
	if count == 1 {
		_ = t.client.Expire(ctx, k, t.ttl)
	}
	return count >= int64(t.thresholdFor(kind)), nil
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

// ShouldPromptForKind is the kind-aware variant of ShouldPrompt.
func (t *Tracker) ShouldPromptForKind(ctx context.Context, kind channel.Kind, key string) (bool, error) {
	k := t.keyPrefix + key
	count, err := t.client.Get(ctx, k).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reauth get: %w", err)
	}
	return count >= t.thresholdFor(kind), nil
}
