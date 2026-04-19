package webwidget

import (
	"testing"
	"time"
)

func TestIssueAndParseVisitorJWT(t *testing.T) {
	svc := NewVisitorJWTService("test-secret-at-least-32-chars-long!", 30)

	token, err := svc.Issue(1, "user@test.com", "token123", 42)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if claims.ContactID != 1 {
		t.Errorf("contact_id = %d, want 1", claims.ContactID)
	}
	if claims.Sub != "user@test.com" {
		t.Errorf("sub = %q, want %q", claims.Sub, "user@test.com")
	}
	if claims.WebsiteToken != "token123" {
		t.Errorf("website_token = %q, want %q", claims.WebsiteToken, "token123")
	}
	if claims.ConversationID != 42 {
		t.Errorf("conversation_id = %d, want 42", claims.ConversationID)
	}
}

func TestExpiredJWT(t *testing.T) {
	svc := NewVisitorJWTService("test-secret-at-least-32-chars-long!", 0)
	svc.ttl = -time.Hour

	token, err := svc.Issue(1, "user@test.com", "token123", 42)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = svc.Parse(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestWrongSignature(t *testing.T) {
	svc1 := NewVisitorJWTService("secret-one-at-least-32-chars-long!!", 30)
	svc2 := NewVisitorJWTService("secret-two-at-least-32-chars-long!!", 30)

	token, _ := svc1.Issue(1, "user@test.com", "token123", 42)
	_, err := svc2.Parse(token)
	if err == nil {
		t.Fatal("expected error for wrong signature")
	}
}
