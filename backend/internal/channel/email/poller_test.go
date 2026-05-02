package email_test

import (
	"context"
	"testing"
	"time"

	emailch "backend/internal/channel/email"
	"backend/internal/model"
)

func TestNewEmailPoller_DefaultInterval(t *testing.T) {
	ch := model.ChannelEmail{ID: 1, AccountID: 1}
	deps := emailch.PollDeps{}
	poller := emailch.NewEmailPoller(ch, deps, 0)
	if poller == nil {
		t.Fatal("expected non-nil poller")
	}
}

func TestPoller_CancelsOnContextDone(t *testing.T) {
	ch := model.ChannelEmail{ID: 2, AccountID: 1}

	poller := emailch.NewEmailPoller(ch, emailch.PollDeps{}, 10*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		poller.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("poller did not stop after context cancellation")
	}
}
