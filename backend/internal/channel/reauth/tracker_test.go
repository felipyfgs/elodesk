package reauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTracker_RecordError_UnderThreshold(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	prompt1, _ := tr.RecordError(ctx, "ch:1")
	if prompt1 {
		t.Fatal("should not prompt after 1 error")
	}

	prompt2, _ := tr.RecordError(ctx, "ch:1")
	if prompt2 {
		t.Fatal("should not prompt after 2 errors")
	}
}

func TestTracker_RecordError_AtThreshold(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	_, _ = tr.RecordError(ctx, "ch:2")
	_, _ = tr.RecordError(ctx, "ch:2")
	prompt3, _ := tr.RecordError(ctx, "ch:2")
	if !prompt3 {
		t.Fatal("should prompt after 3 errors (threshold)")
	}
}

func TestTracker_Reset(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	_, _ = tr.RecordError(ctx, "ch:3")
	_, _ = tr.RecordError(ctx, "ch:3")

	if err := tr.Reset(ctx, "ch:3"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	prompt, _ := tr.RecordError(ctx, "ch:3")
	if prompt {
		t.Fatal("should not prompt after reset + 1 error")
	}
}

func TestTracker_ShouldPrompt(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	prompt, _ := tr.ShouldPrompt(ctx, "ch:4")
	if prompt {
		t.Fatal("should not prompt with no errors")
	}

	_, _ = tr.RecordError(ctx, "ch:4")
	_, _ = tr.RecordError(ctx, "ch:4")
	_, _ = tr.RecordError(ctx, "ch:4")

	prompt, _ = tr.ShouldPrompt(ctx, "ch:4")
	if !prompt {
		t.Fatal("should prompt after 3 errors")
	}
}

func TestTracker_Expired(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := &Tracker{
		client:    client,
		threshold: 3,
		ttl:       1 * time.Second,
		keyPrefix: "elodesk:reauth:",
	}
	ctx := context.Background()

	_, _ = tr.RecordError(ctx, "ch:5")
	_, _ = tr.RecordError(ctx, "ch:5")

	mr.FastForward(2 * time.Second)

	prompt, _ := tr.RecordError(ctx, "ch:5")
	if prompt {
		t.Fatal("should not prompt after TTL expiry (counter reset)")
	}
}
