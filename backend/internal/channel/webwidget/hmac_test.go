package webwidget

import (
	"testing"
)

func TestComputeAndVerifyIdentifierHash(t *testing.T) {
	hmacToken := "test-hmac-secret-key-32bytes!!!"
	identifier := "user@acme.com"

	hash := ComputeIdentifierHash(hmacToken, identifier)
	if hash == "" {
		t.Fatal("hash is empty")
	}

	if !VerifyIdentifierHash(hmacToken, identifier, hash) {
		t.Error("valid hash should verify")
	}
}

func TestInvalidIdentifierHash(t *testing.T) {
	hmacToken := "test-hmac-secret-key-32bytes!!!"
	identifier := "user@acme.com"

	if VerifyIdentifierHash(hmacToken, identifier, "invalidhash") {
		t.Error("invalid hash should not verify")
	}
}

func TestDifferentIdentifierProducesDifferentHash(t *testing.T) {
	hmacToken := "test-hmac-secret-key-32bytes!!!"
	hash1 := ComputeIdentifierHash(hmacToken, "user1@acme.com")
	hash2 := ComputeIdentifierHash(hmacToken, "user2@acme.com")

	if hash1 == hash2 {
		t.Error("different identifiers should produce different hashes")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	hmacToken := "test-hmac-secret-key-32bytes!!!"
	identifier := "user@acme.com"
	_ = ComputeIdentifierHash(hmacToken, identifier)

	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	if VerifyIdentifierHash(hmacToken, identifier, wrongHash) {
		t.Error("wrong hash should not verify")
	}
}
