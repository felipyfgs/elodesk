package middleware

import (
	"testing"
	"time"
)

func TestWidgetRateLimiter_ByIP(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestWidgetRateLimiter_ByToken(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestWidgetRateLimiter_ResetAfterWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestWidgetRateLimiter_IndependentBuckets(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

var _ = time.Second
