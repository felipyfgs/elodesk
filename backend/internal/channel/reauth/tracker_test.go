package reauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
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

func TestTracker_RecordErrorForKind_Instagram(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	// Instagram threshold is 1 — single failure is enough to prompt reauth.
	prompt, err := tr.RecordErrorForKind(ctx, channel.KindInstagram, "ig:42")
	if err != nil {
		t.Fatalf("record error: %v", err)
	}
	if !prompt {
		t.Fatal("instagram should prompt after 1 error (threshold=1)")
	}
}

func TestTracker_ShouldPromptForKind_TiktokThreshold(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	// No errors yet: should not prompt.
	prompt, _ := tr.ShouldPromptForKind(ctx, channel.KindTiktok, "tt:1")
	if prompt {
		t.Fatal("tiktok: no errors yet, should not prompt")
	}

	// One error: tiktok threshold is 1, should prompt.
	_, _ = tr.RecordErrorForKind(ctx, channel.KindTiktok, "tt:1")
	prompt, _ = tr.ShouldPromptForKind(ctx, channel.KindTiktok, "tt:1")
	if !prompt {
		t.Fatal("tiktok: after 1 error should prompt (threshold=1)")
	}
}

func TestTracker_RecordErrorForKind_DefaultKind(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	tr := NewTracker(client)
	ctx := context.Background()

	// Twilio isn't in kindThresholds → uses default threshold (3). First 2
	// errors must not prompt.
	for i := 0; i < 2; i++ {
		prompt, _ := tr.RecordErrorForKind(ctx, channel.KindTwilio, "tw:1")
		if prompt {
			t.Fatalf("twilio: should not prompt after %d errors (default threshold=3)", i+1)
		}
	}

	// Third error hits default threshold.
	prompt, _ := tr.RecordErrorForKind(ctx, channel.KindTwilio, "tw:1")
	if !prompt {
		t.Fatal("twilio: should prompt after 3 errors (default threshold)")
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
