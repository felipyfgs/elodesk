package line

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"backend/internal/channel"
	"backend/internal/model"
)

func TestVerifySignature_Valid(t *testing.T) {
	secret := "line-channel-secret"
	body := []byte(`{"destination":"U123","events":[]}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !VerifySignature(secret, body, sig) {
		t.Fatalf("expected VerifySignature to accept a valid signature")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	cases := []struct {
		name string
		sig  string
	}{
		{"empty", ""},
		{"garbage", "not-a-valid-base64-signature"},
		{"wrong", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if VerifySignature("secret", []byte("body"), tc.sig) {
				t.Fatalf("expected VerifySignature to reject case %q", tc.name)
			}
		})
	}
}

func TestVerifySignature_EmptySecret(t *testing.T) {
	if VerifySignature("", []byte("body"), "sig") {
		t.Fatalf("expected empty secret to reject any signature")
	}
}

func TestExtractContent_Text(t *testing.T) {
	msg := &EventMessage{Type: MessageTypeText, Text: "hello"}
	content, ct, attrs := extractContent(msg)
	if content != "hello" {
		t.Fatalf("expected text content, got %q", content)
	}
	if ct != model.ContentTypeText {
		t.Fatalf("expected text content type, got %v", ct)
	}
	if attrs != nil {
		t.Fatalf("expected nil attrs for plain text, got %v", *attrs)
	}
}

func TestExtractContent_Sticker(t *testing.T) {
	msg := &EventMessage{Type: MessageTypeSticker, StickerID: "s1", PackageID: "p1", ID: "m1"}
	content, ct, attrs := extractContent(msg)
	if content == "" {
		t.Fatalf("expected sticker markdown content")
	}
	if ct != model.ContentTypeSticker {
		t.Fatalf("expected sticker content type, got %v", ct)
	}
	if attrs == nil {
		t.Fatalf("expected sticker attrs to be set")
	}
}

func TestExtractContent_Unsupported(t *testing.T) {
	msg := &EventMessage{Type: "mysterious"}
	content, ct, attrs := extractContent(msg)
	if content != "[unsupported]" {
		t.Fatalf("expected fallback content, got %q", content)
	}
	if ct != model.ContentTypeText {
		t.Fatalf("expected text fallback, got %v", ct)
	}
	if attrs == nil {
		t.Fatalf("expected unsupported attrs to be set")
	}
}

// compile-time guard: Channel must satisfy the channel.Channel interface.
var _ channel.Channel = (*Channel)(nil)
