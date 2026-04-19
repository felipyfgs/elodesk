package webwidget

import (
	"context"
	"testing"
)

func TestSSE_Keepalive(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestSSE_PublishAndReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestSSE_ClientDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

func TestSSE_WrongPubsubToken(t *testing.T) {
	if testing.Short() {
		t.Skip("requires redis")
	}

	t.Skip("integration test: requires redis")
}

var _ = context.Background()
