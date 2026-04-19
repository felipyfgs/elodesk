package bandwidth

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestParseInbound(t *testing.T) {
	payload := `[{"type":"message-received","time":"2024-01-01T00:00:00Z","message":{"id":"msg123","from":"+14155551234","to":["5511988887777"],"text":"Hello from Bandwidth","media":[]}}]`
	req := &http.Request{Body: http.NoBody}
	req.Body = http.NoBody

	p := &Provider{}

	var buf bytes.Buffer
	buf.WriteString(payload)
	req.Body = io.NopCloser(&buf)

	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if msg.SourceID != "msg123" {
		t.Errorf("SourceID = %q, want msg123", msg.SourceID)
	}
	if msg.From != "+14155551234" {
		t.Errorf("From = %q, want +14155551234", msg.From)
	}
	if msg.Content != "Hello from Bandwidth" {
		t.Errorf("Content = %q, want Hello from Bandwidth", msg.Content)
	}
}

func TestParseInbound_WithMedia(t *testing.T) {
	payload := `[{"type":"message-received","time":"2024-01-01T00:00:00Z","message":{"id":"msg456","from":"+14155551234","to":["5511988887777"],"text":"","media":["https://example.com/img.jpg"]}}]`

	p := &Provider{}

	var buf bytes.Buffer
	buf.WriteString(payload)
	req := &http.Request{Body: io.NopCloser(&buf)}

	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if len(msg.MediaURLs) != 1 {
		t.Fatalf("MediaURLs len = %d, want 1", len(msg.MediaURLs))
	}
	if msg.MediaURLs[0] != "https://example.com/img.jpg" {
		t.Errorf("MediaURLs[0] = %q, want %q", msg.MediaURLs[0], "https://example.com/img.jpg")
	}
}

func TestParseDeliveryStatus(t *testing.T) {
	tests := []struct {
		payload    string
		wantStatus string
	}{
		{`[{"type":"message-delivered","message":{"id":"msg789"}}]`, "delivered"},
		{`[{"type":"message-failed","message":{"id":"msg789"}}]`, "failed"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		buf.WriteString(tt.payload)
		req := &http.Request{Body: io.NopCloser(&buf)}

		p := &Provider{}
		cb, err := p.ParseDeliveryStatus(req)
		if err != nil {
			t.Fatalf("ParseDeliveryStatus error: %v", err)
		}
		if cb.Status != tt.wantStatus {
			t.Errorf("Status = %q, want %q", cb.Status, tt.wantStatus)
		}
		if cb.SourceID != "msg789" {
			t.Errorf("SourceID = %q, want msg789", cb.SourceID)
		}
	}
}

func TestName(t *testing.T) {
	p := &Provider{}
	if p.Name() != "bandwidth" {
		t.Errorf("Name() = %q, want bandwidth", p.Name())
	}
}
