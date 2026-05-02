package instagram

import (
	"testing"
	"time"

	"backend/internal/model"
)

func TestRefreshThreshold(t *testing.T) {
	ch := &model.ChannelInstagram{
		ExpiresAt: time.Now().Add(20 * 24 * time.Hour),
	}
	threshold := time.Now().Add(refreshThresholdDays * 24 * time.Hour)
	if !ch.ExpiresAt.After(threshold) {
		t.Fatal("expected no refresh needed for token expiring in 20 days")
	}
}

func TestRefreshThreshold_ShouldRefresh(t *testing.T) {
	ch := &model.ChannelInstagram{
		ExpiresAt: time.Now().Add(5 * 24 * time.Hour),
	}
	threshold := time.Now().Add(refreshThresholdDays * 24 * time.Hour)
	if ch.ExpiresAt.After(threshold) {
		t.Fatal("expected refresh needed for token expiring in 5 days")
	}
}
