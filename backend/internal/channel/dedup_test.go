package channel

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, client
}

func TestDedupLock_Acquire_First(t *testing.T) {
	mr, client := newTestRedis(t)
	dl := NewDedupLock(client)
	ctx := context.Background()

	acquired, err := dl.Acquire(ctx, "test:dedup:first")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected first acquire to succeed")
	}

	_, err = mr.Get("test:dedup:first")
	if err != nil {
		t.Fatal("expected key to exist in redis")
	}
}

func TestDedupLock_Acquire_Duplicate(t *testing.T) {
	_, client := newTestRedis(t)
	dl := NewDedupLock(client)
	ctx := context.Background()

	acquired1, _ := dl.Acquire(ctx, "test:dedup:dup")
	if !acquired1 {
		t.Fatal("first acquire should succeed")
	}

	acquired2, _ := dl.Acquire(ctx, "test:dedup:dup")
	if acquired2 {
		t.Fatal("second acquire should fail (duplicate)")
	}
}

func TestDedupLock_Acquire_Expired(t *testing.T) {
	mr, client := newTestRedis(t)
	dl := &DedupLock{client: client, ttl: 100 * time.Millisecond}
	ctx := context.Background()

	acquired1, _ := dl.Acquire(ctx, "test:dedup:expire")
	if !acquired1 {
		t.Fatal("first acquire should succeed")
	}

	mr.FastForward(200 * time.Millisecond)

	acquired2, _ := dl.Acquire(ctx, "test:dedup:expire")
	if !acquired2 {
		t.Fatal("acquire after TTL should succeed")
	}
}
