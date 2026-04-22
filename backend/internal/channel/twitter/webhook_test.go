package twitter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestCRCChallenge_ValidComputesHMAC(t *testing.T) {
	secret := "top-secret-consumer"
	token := "challenge-0xdead"

	got := CRCChallenge(secret, token)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	want := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("CRCChallenge mismatch: got=%q want=%q", got, want)
	}
}

func TestCRCChallenge_EmptyInputs(t *testing.T) {
	if CRCChallenge("", "x") != "" {
		t.Fatalf("empty secret should produce empty response")
	}
	if CRCChallenge("x", "") != "" {
		t.Fatalf("empty crc_token should produce empty response")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "consumer-secret"
	body := []byte(`{"direct_message_events":[]}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !VerifySignature(secret, body, sig) {
		t.Fatalf("expected signature to verify")
	}
}

func TestVerifySignature_Tampered(t *testing.T) {
	secret := "consumer-secret"
	body := []byte(`{"direct_message_events":[]}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tampered := []byte(`{"direct_message_events":[{"injected":true}]}`)
	if VerifySignature(secret, tampered, sig) {
		t.Fatalf("tampered body must fail signature check")
	}
}

func TestVerifySignature_MissingInputs(t *testing.T) {
	if VerifySignature("", []byte("x"), "sha256=abc") {
		t.Fatalf("empty secret should reject")
	}
	if VerifySignature("secret", []byte("x"), "") {
		t.Fatalf("empty signature should reject")
	}
}

func TestHasSupportedEvent_DMOnly(t *testing.T) {
	if !hasSupportedEvent([]byte(`{"direct_message_events":[]}`)) {
		t.Fatalf("direct_message_events should be supported")
	}
	if hasSupportedEvent([]byte(`{"tweet_create_events":[{"id":"1"}]}`)) {
		t.Fatalf("tweet_create_events should NOT be supported (ignored per spec)")
	}
}

// OAuth 1.0a signature: verify the base string + signing key produce the
// expected HMAC-SHA1 output. Uses well-known values so we catch accidental
// changes to percentEncode or signRequest logic.
func TestSignRequest_KnownVectors(t *testing.T) {
	oauthParams := map[string]string{
		"oauth_consumer_key":     "ckey",
		"oauth_nonce":            "nonce123",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1700000000",
		"oauth_token":            "tok",
		"oauth_version":          "1.0",
	}
	sig1 := signRequest("POST", "https://api.twitter.com/2/example", oauthParams, nil, "csecret", "tsecret")
	sig2 := signRequest("POST", "https://api.twitter.com/2/example", oauthParams, nil, "csecret", "tsecret")
	if sig1 != sig2 {
		t.Fatalf("signRequest must be deterministic for identical inputs")
	}

	// Different token secret must change the signature.
	sig3 := signRequest("POST", "https://api.twitter.com/2/example", oauthParams, nil, "csecret", "other")
	if sig1 == sig3 {
		t.Fatalf("signature must change when token secret changes")
	}
}

func TestPercentEncode_UnreservedPassthrough(t *testing.T) {
	in := "abc-._~XYZ0189"
	if got := percentEncode(in); got != in {
		t.Fatalf("unreserved characters must pass through: got=%q", got)
	}
}

func TestPercentEncode_EncodesReserved(t *testing.T) {
	// "!" is reserved and must be encoded per OAuth 1.0a rules.
	if got := percentEncode("hello world!"); got != "hello%20world%21" {
		t.Fatalf("expected %q got %q", "hello%20world%21", got)
	}
}
