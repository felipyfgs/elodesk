package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultDedupTTL = 24 * time.Hour

type DedupLock struct {
	client redis.Cmdable
	ttl    time.Duration
}

func NewDedupLock(client redis.Cmdable) *DedupLock {
	return &DedupLock{
		client: client,
		ttl:    defaultDedupTTL,
	}
}

func (d *DedupLock) Acquire(ctx context.Context, key string) (bool, error) {
	ok, err := d.client.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("dedup setnx: %w", err)
	}
	return ok, nil
}
