package zenvia

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestParseInbound_Text(t *testing.T) {
	payload := `{"id":"msg123","from":"5511988887777","to":"5511999999999","contents":[{"type":"text","text":"Hello from Zenvia"}]}`

	var buf bytes.Buffer
	buf.WriteString(payload)
	req := &http.Request{Body: io.NopCloser(&buf)}

	p := &Provider{}
	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if msg.SourceID != "msg123" {
		t.Errorf("SourceID = %q, want msg123", msg.SourceID)
	}
	if msg.From != "5511988887777" {
		t.Errorf("From = %q, want 5511988887777", msg.From)
	}
	if msg.Content != "Hello from Zenvia" {
		t.Errorf("Content = %q, want Hello from Zenvia", msg.Content)
	}
}

func TestParseInbound_Media(t *testing.T) {
	payload := `{"id":"msg456","from":"5511988887777","to":"5511999999999","contents":[{"type":"text","text":"Check this"},{"type":"media","payload":{"mediaUrl":"https://example.com/img.jpg","mediaType":"image/jpeg"}}]}`

	var buf bytes.Buffer
	buf.WriteString(payload)
	req := &http.Request{Body: io.NopCloser(&buf)}

	p := &Provider{}
	msg, err := p.ParseInbound(req)
	if err != nil {
		t.Fatalf("ParseInbound error: %v", err)
	}
	if len(msg.MediaURLs) != 1 {
		t.Fatalf("MediaURLs len = %d, want 1", len(msg.MediaURLs))
	}
	if msg.MediaURLs[0] != "https://example.com/img.jpg" {
		t.Errorf("MediaURLs[0] = %q, want https://example.com/img.jpg", msg.MediaURLs[0])
	}
}

func TestParseDeliveryStatus(t *testing.T) {
	tests := []struct {
		payload    string
		wantStatus string
	}{
		{`{"messageId":"msg789","messageStatus":{"code":"DELIVERED"}}`, "delivered"},
		{`{"messageId":"msg789","messageStatus":{"code":"SENT"}}`, "sent"},
		{`{"messageId":"msg789","messageStatus":{"code":"NOT_DELIVERED"}}`, "failed"},
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
	}
}

func TestName(t *testing.T) {
	p := &Provider{}
	if p.Name() != "zenvia" {
		t.Errorf("Name() = %q, want zenvia", p.Name())
	}
}
