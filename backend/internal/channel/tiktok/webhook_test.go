package tiktok

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"backend/internal/channel"
)

func signedHeader(secret, body string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + body))
	return "t=" + strconv.FormatInt(ts, 10) + ",s=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	secret := "app-secret"
	body := `{"event":"im_receive_msg"}`
	now := time.Now()
	header := signedHeader(secret, body, now.Unix())
	if !VerifySignature(secret, []byte(body), header, now) {
		t.Fatalf("expected valid signature")
	}
}

func TestVerifySignature_SkewTooLarge(t *testing.T) {
	secret := "app-secret"
	body := `{}`
	now := time.Now()
	header := signedHeader(secret, body, now.Add(-10*time.Second).Unix())
	if VerifySignature(secret, []byte(body), header, now) {
		t.Fatalf("expected skew rejection")
	}
}

func TestVerifySignature_InvalidHash(t *testing.T) {
	secret := "app-secret"
	body := `{}`
	now := time.Now()
	header := fmt.Sprintf("t=%d,s=deadbeef", now.Unix())
	if VerifySignature(secret, []byte(body), header, now) {
		t.Fatalf("expected bad signature to reject")
	}
}

func TestVerifySignature_MissingParts(t *testing.T) {
	if VerifySignature("secret", []byte("x"), "", time.Now()) {
		t.Fatalf("empty header should fail")
	}
	if VerifySignature("", []byte("x"), "t=1,s=2", time.Now()) {
		t.Fatalf("empty secret should fail")
	}
	if VerifySignature("secret", []byte("x"), "garbage", time.Now()) {
		t.Fatalf("missing t/s parts should fail")
	}
}

func TestScopesGranted_All(t *testing.T) {
	granted := ""
	for i, s := range RequiredScopes {
		if i > 0 {
			granted += ","
		}
		granted += s
	}
	if !ScopesGranted(granted) {
		t.Fatalf("expected all required scopes to be granted")
	}
}

func TestScopesGranted_Missing(t *testing.T) {
	if ScopesGranted("user.info.basic") {
		t.Fatalf("expected partial scope set to be rejected")
	}
}

func TestExtractContent_Text(t *testing.T) {
	c := &EventContent{Type: MessageTypeText, ConversationID: "c1", MessageID: "m1", Text: &EventTextBody{Body: "hi"}}
	content, _, attrs := extractContent(c, false)
	if content != "hi" {
		t.Fatalf("expected text content, got %q", content)
	}
	if attrs == nil {
		t.Fatalf("expected attrs with conversation_id")
	}
}

func TestExtractContent_Image(t *testing.T) {
	c := &EventContent{Type: MessageTypeImage, ConversationID: "c1", MessageID: "m1", Image: &EventImageBody{MediaID: "media-1"}}
	_, _, attrs := extractContent(c, false)
	if attrs == nil {
		t.Fatalf("expected attrs with media_id")
	}
}

// guard: Channel must satisfy the channel.Channel interface.
var _ channel.Channel = (*Channel)(nil)
