package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte(`{"object":"instagram","entry":[]}`)
	secret := "test-app-secret"
	header := signBody(body, secret)

	if !VerifySignature(body, header, secret) {
		t.Fatal("expected true for valid signature")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte(`{"object":"instagram","entry":[]}`)
	if VerifySignature(body, "sha256=deadbeef", "test-secret") {
		t.Fatal("expected false for invalid signature")
	}
}

func TestVerifySignature_MissingHeader(t *testing.T) {
	if VerifySignature([]byte("body"), "", "secret") {
		t.Fatal("expected false for empty header")
	}
}

func TestVerifySignature_MissingPrefix(t *testing.T) {
	if VerifySignature([]byte("body"), "abc123", "secret") {
		t.Fatal("expected false for header without sha256= prefix")
	}
}

func TestVerifySignature_AllowUnsigned(t *testing.T) {
	t.Setenv("META_ALLOW_UNSIGNED", "true")
	// Reload GraphVersion (won't affect VerifySignature, but re-init env)
	_ = os.Setenv("META_ALLOW_UNSIGNED", "true")
	defer func() { _ = os.Unsetenv("META_ALLOW_UNSIGNED") }()

	if !VerifySignature([]byte("anything"), "", "secret") {
		t.Fatal("expected true when META_ALLOW_UNSIGNED=true")
	}
}
