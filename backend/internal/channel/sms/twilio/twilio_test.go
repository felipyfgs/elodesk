package twilio

import (
	"net/http"
	"net/url"
	"testing"
)

func TestComputeSignature(t *testing.T) {
	p := &Provider{}

	vals := url.Values{}
	vals.Set("From", "+14155551234")
	vals.Set("To", "+5511988887777")
	vals.Set("Body", "Hello")

	sig := p.computeSignature("https://example.com/webhook", vals, "secret_token")
	if sig == "" {
		t.Error("expected non-empty signature")
	}
}

func TestParseInbound_Text(t *testing.T) {
	form := url.Values{}
	form.Set("MessageSid", "SM123")
	form.Set("From", "+14155551234")
	form.Set("To", "+5511988887777")
	form.Set("Body", "Hello World")
	form.Set("NumMedia", "0")

	req := &http.Request{Form: form}
	p := &Provider{}

	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if msg.SourceID != "SM123" {
		t.Errorf("SourceID = %q, want %q", msg.SourceID, "SM123")
	}
	if msg.From != "+14155551234" {
		t.Errorf("From = %q, want %q", msg.From, "+14155551234")
	}
	if msg.Content != "Hello World" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello World")
	}
}

func TestParseInbound_MMS(t *testing.T) {
	form := url.Values{}
	form.Set("MessageSid", "SM456")
	form.Set("From", "+14155551234")
	form.Set("To", "+5511988887777")
	form.Set("Body", "Check this out")
	form.Set("NumMedia", "2")
	form.Set("MediaUrl0", "https://api.twilio.com/media1")
	form.Set("MediaContentType0", "image/jpeg")
	form.Set("MediaUrl1", "https://api.twilio.com/media2")
	form.Set("MediaContentType1", "video/mp4")

	req := &http.Request{Form: form}
	p := &Provider{}

	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if len(msg.MediaURLs) != 2 {
		t.Fatalf("MediaURLs len = %d, want 2", len(msg.MediaURLs))
	}
	if msg.MediaURLs[0] != "https://api.twilio.com/media1" {
		t.Errorf("MediaURLs[0] = %q, want %q", msg.MediaURLs[0], "https://api.twilio.com/media1")
	}
	if msg.MediaTypes[0] != "image/jpeg" {
		t.Errorf("MediaTypes[0] = %q, want %q", msg.MediaTypes[0], "image/jpeg")
	}
}

func TestParseDeliveryStatus(t *testing.T) {
	tests := []struct {
		status     string
		errorCode  string
		wantStatus string
	}{
		{"delivered", "", "delivered"},
		{"failed", "30003", "failed"},
		{"sent", "", "sent"},
		{"unknown", "", "sent"},
	}

	for _, tt := range tests {
		form := url.Values{}
		form.Set("MessageSid", "SM789")
		form.Set("MessageStatus", tt.status)
		if tt.errorCode != "" {
			form.Set("ErrorCode", tt.errorCode)
		}

		req := &http.Request{Form: form}
		p := &Provider{}

		cb, err := p.ParseDeliveryStatus(req)
		if err != nil {
			t.Fatalf("ParseDeliveryStatus error: %v", err)
		}
		if cb.Status != tt.wantStatus {
			t.Errorf("Status = %q, want %q", cb.Status, tt.wantStatus)
		}
		if cb.SourceID != "SM789" {
			t.Errorf("SourceID = %q, want SM789", cb.SourceID)
		}
	}
}

func TestName(t *testing.T) {
	p := &Provider{}
	if p.Name() != "twilio" {
		t.Errorf("Name() = %q, want %q", p.Name(), "twilio")
	}
}
