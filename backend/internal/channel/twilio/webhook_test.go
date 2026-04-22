package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"testing"

	"backend/internal/channel"
)

func computeSignature(authToken, fullURL string, params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(fullURL)
	for _, k := range keys {
		for _, v := range params[k] {
			sb.WriteString(k)
			sb.WriteString(v)
		}
	}
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	authToken := "super-secret"
	fullURL := "https://elodesk.test/webhooks/twilio/abc"
	params := url.Values{
		"MessageSid": {"SM1"},
		"From":       {"whatsapp:+5511988887777"},
		"Body":       {"olá"},
	}
	sig := computeSignature(authToken, fullURL, params)
	if !VerifySignature(authToken, fullURL, params, sig) {
		t.Fatalf("expected signature to verify")
	}
}

func TestVerifySignature_Tampered(t *testing.T) {
	authToken := "super-secret"
	fullURL := "https://elodesk.test/webhooks/twilio/abc"
	params := url.Values{
		"MessageSid": {"SM1"},
		"Body":       {"hi"},
	}
	sig := computeSignature(authToken, fullURL, params)
	// flip one form value after signing
	params.Set("Body", "bye")
	if VerifySignature(authToken, fullURL, params, sig) {
		t.Fatalf("tampered params should fail signature check")
	}
}

func TestVerifySignature_MissingInput(t *testing.T) {
	params := url.Values{"A": {"1"}}
	if VerifySignature("", "https://x", params, "sig") {
		t.Fatalf("empty token should reject")
	}
	if VerifySignature("tok", "https://x", params, "") {
		t.Fatalf("empty signature should reject")
	}
}

func TestDetectMedium(t *testing.T) {
	if DetectMedium("whatsapp:+12345") != "whatsapp" {
		t.Fatalf("whatsapp: prefix should map to whatsapp medium")
	}
	if DetectMedium("+12345") != "sms" {
		t.Fatalf("plain E.164 should map to sms medium")
	}
}

func TestParseInbound_WithMedia(t *testing.T) {
	form := url.Values{
		"MessageSid":        {"SM42"},
		"From":              {"whatsapp:+5511999998888"},
		"To":                {"whatsapp:+14155238886"},
		"Body":              {"see attached"},
		"NumMedia":          {"2"},
		"MediaUrl0":         {"https://media.test/a.jpg"},
		"MediaContentType0": {"image/jpeg"},
		"MediaUrl1":         {"https://media.test/b.pdf"},
		"MediaContentType1": {"application/pdf"},
	}
	p := ParseInbound(form)
	if p.MessageSid != "SM42" || p.Body != "see attached" {
		t.Fatalf("basic fields parsed wrong: %+v", p)
	}
	if len(p.MediaURLs) != 2 || p.MediaURLs[0] != "https://media.test/a.jpg" {
		t.Fatalf("media urls parsed wrong: %+v", p.MediaURLs)
	}
	if len(p.MediaTypes) != 2 || p.MediaTypes[1] != "application/pdf" {
		t.Fatalf("media types parsed wrong: %+v", p.MediaTypes)
	}
}

// guard: Channel must satisfy the channel.Channel interface.
var _ channel.Channel = (*Channel)(nil)
