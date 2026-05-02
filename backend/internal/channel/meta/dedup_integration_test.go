package meta_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
)

func TestDedupCrossChannelInstagramFacebook(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	dedup := channel.NewDedupLock(client)
	ctx := context.Background()

	mid := "m_linked_abc123"
	key := "elodesk:meta:" + mid

	ok1, err := dedup.Acquire(ctx, key)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if !ok1 {
		t.Fatal("expected first acquire to succeed")
	}

	ok2, err := dedup.Acquire(ctx, key)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	if ok2 {
		t.Fatal("expected second acquire to fail (dedup)")
	}

	if !mr.Exists(key) {
		t.Fatal("expected dedup key to exist in redis")
	}
}

func TestDedupDifferentMids(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	dedup := channel.NewDedupLock(client)
	ctx := context.Background()

	ok1, _ := dedup.Acquire(ctx, "elodesk:meta:mid_one")
	ok2, _ := dedup.Acquire(ctx, "elodesk:meta:mid_two")

	if !ok1 || !ok2 {
		t.Fatal("expected both distinct mids to acquire successfully")
	}
}
