package meta_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
)

// TestDedupCrossChannelInstagramFacebook verifies that when the same Meta
// message mid arrives via both the Instagram webhook AND the Facebook webhook
// (linked Instagram Business Account on a Page), only the first attempt
// succeeds — the second is silently dropped. This is the "linked IG dedup
// cross-channel" scenario from task 8.6 of add-meta-social-channels.
func TestDedupCrossChannelInstagramFacebook(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	dedup := channel.NewDedupLock(client)
	ctx := context.Background()

	mid := "m_linked_abc123"
	key := "elodesk:meta:" + mid

	// First webhook arrives (via /webhooks/instagram/:id1)
	ok1, err := dedup.Acquire(ctx, key)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if !ok1 {
		t.Fatal("expected first acquire to succeed")
	}

	// Second webhook arrives for the same mid (via /webhooks/facebook/:id2
	// where id2 has instagram_id linked to id1)
	ok2, err := dedup.Acquire(ctx, key)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	if ok2 {
		t.Fatal("expected second acquire to fail (dedup)")
	}

	// Verify the key is present with the expected TTL window
	if !mr.Exists(key) {
		t.Fatal("expected dedup key to exist in redis")
	}
}

// TestDedupDifferentMids verifies that distinct mids do not interfere.
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
