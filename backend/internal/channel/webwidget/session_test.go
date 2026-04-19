package webwidget

import (
	"context"
	"testing"
	"time"
)

func TestCreateOrResumeSession_NewSession(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestCreateOrResumeSession_ResumeViaCookie(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestCreateOrResumeSession_ExpiredJWT(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestCreateOrResumeSession_GeneratesPubsubToken(t *testing.T) {
	token := generatePubsubToken()
	if token == "" {
		t.Fatal("pubsub_token should not be empty")
	}
	if len(token) < 32 {
		t.Fatalf("pubsub_token too short: %d", len(token))
	}

	token2 := generatePubsubToken()
	if token == token2 {
		t.Fatal("pubsub_tokens should be unique")
	}
}

func TestGenerateRandomID(t *testing.T) {
	id := generateRandomID(16)
	if id == "" {
		t.Fatal("random ID should not be empty")
	}

	id2 := generateRandomID(16)
	if id == id2 {
		t.Fatal("random IDs should be unique")
	}
}

var _ = context.Background()
var _ = time.Now
