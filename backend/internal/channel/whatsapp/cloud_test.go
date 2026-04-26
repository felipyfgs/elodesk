package whatsapp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCloudProvider_VerifyHandshake_Valid(t *testing.T) {
	p := NewCloudProvider(nil)
	query := map[string]string{
		"hub.mode":         "subscribe",
		"hub.verify_token": "my_token",
		"hub.challenge":    "challenge_123",
	}
	challenge, ok := p.VerifyHandshake(context.Background(), query, "my_token")
	if !ok {
		t.Fatal("expected handshake to succeed")
	}
	if challenge != "challenge_123" {
		t.Fatalf("expected challenge_123, got %s", challenge)
	}
}

func TestCloudProvider_VerifyHandshake_InvalidToken(t *testing.T) {
	p := NewCloudProvider(nil)
	query := map[string]string{
		"hub.mode":         "subscribe",
		"hub.verify_token": "wrong_token",
		"hub.challenge":    "challenge_123",
	}
	_, ok := p.VerifyHandshake(context.Background(), query, "my_token")
	if ok {
		t.Fatal("expected handshake to fail")
	}
}

func TestCloudProvider_VerifyHandshake_MissingMode(t *testing.T) {
	p := NewCloudProvider(nil)
	query := map[string]string{
		"hub.verify_token": "my_token",
		"hub.challenge":    "challenge_123",
	}
	_, ok := p.VerifyHandshake(context.Background(), query, "my_token")
	if ok {
		t.Fatal("expected handshake to fail without mode")
	}
}

func TestCloudProvider_VerifySignature(t *testing.T) {
	p := NewCloudProvider(nil)
	headers := map[string]string{}
	if !p.VerifySignature(context.TODO(), nil, headers, "") {
		t.Fatal("expected signature to pass when empty")
	}
}

func TestCloudProvider_ParsePayload_TextMessage(t *testing.T) {
	p := NewCloudProvider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"messages": [{
						"from": "5511999999999",
						"id": "wamid.ABC123",
						"to": "5511888888888",
						"type": "text",
						"timestamp": 1700000000,
						"text": {"body": "Hello!"}
					}]
				}
			}]
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].SourceID != "wamid.ABC123" {
		t.Fatalf("expected source_id wamid.ABC123, got %s", result.Messages[0].SourceID)
	}
	if result.Messages[0].Content != "Hello!" {
		t.Fatalf("expected content Hello!, got %s", result.Messages[0].Content)
	}
	if result.Messages[0].From != "5511999999999" {
		t.Fatalf("expected from 5511999999999, got %s", result.Messages[0].From)
	}
}

func TestCloudProvider_ParsePayload_Status(t *testing.T) {
	p := NewCloudProvider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"statuses": [{
						"id": "wamid.X",
						"status": "delivered",
						"recipient_id": "5511999999999"
					}]
				}
			}]
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(result.Statuses))
	}
	if result.Statuses[0].SourceID != "wamid.X" {
		t.Fatalf("expected source_id wamid.X, got %s", result.Statuses[0].SourceID)
	}
	if result.Statuses[0].Status != "delivered" {
		t.Fatalf("expected delivered, got %s", result.Statuses[0].Status)
	}
}

func TestCloudProvider_ParsePayload_StatusFailed(t *testing.T) {
	p := NewCloudProvider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"statuses": [{
						"id": "wamid.X",
						"status": "failed",
						"errors": [{"code": 131026, "title": "Message Undeliverable"}]
					}]
				}
			}]
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(result.Statuses))
	}
	if result.Statuses[0].ExternalError != "131026: Message Undeliverable" {
		t.Fatalf("expected error, got %s", result.Statuses[0].ExternalError)
	}
}

func TestCloudProvider_ParsePayload_SMBEcho(t *testing.T) {
	p := NewCloudProvider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"smb_message_echoes": [{
						"from": "5511888888888",
						"id": "wamid.ECHO1",
						"to": "5511999999999",
						"type": "text",
						"timestamp": 1700000000,
						"text": {"body": "Echo message"}
					}]
				}
			}]
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if !result.Messages[0].ExternalEcho {
		t.Fatal("expected external_echo to be true")
	}
	if !result.Messages[0].IsEcho {
		t.Fatal("expected is_echo to be true")
	}
}

func TestCloudProvider_ParsePayload_ImageMessage(t *testing.T) {
	p := NewCloudProvider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"value": {
					"messages": [{
						"from": "5511999999999",
						"id": "wamid.IMG1",
						"to": "5511888888888",
						"type": "image",
						"timestamp": 1700000000,
						"image": {"url": "https://example.com/image.jpg", "caption": "Photo"}
					}]
				}
			}]
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].MediaType != "image" {
		t.Fatalf("expected mediaType image, got %s", result.Messages[0].MediaType)
	}
	if result.Messages[0].MediaURL != "https://example.com/image.jpg" {
		t.Fatalf("expected media URL, got %s", result.Messages[0].MediaURL)
	}
	if result.Messages[0].Content != "Photo" {
		t.Fatalf("expected caption Photo, got %s", result.Messages[0].Content)
	}
}

func TestBuildSendBody_Text(t *testing.T) {
	body := buildSendBody("5511999999999", "Hello", "", "", "", "", "")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatal(err)
	}
	if m["messaging_product"] != "whatsapp" {
		t.Fatalf("expected whatsapp, got %v", m["messaging_product"])
	}
	if m["to"] != "5511999999999" {
		t.Fatalf("expected 5511999999999, got %v", m["to"])
	}
	if m["type"] != "text" {
		t.Fatalf("expected text, got %v", m["type"])
	}
}

func TestBuildSendBody_Image(t *testing.T) {
	body := buildSendBody("5511999999999", "Caption", "https://example.com/img.jpg", "image", "", "", "")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "image" {
		t.Fatalf("expected image, got %v", m["type"])
	}
}

func TestBuildSendBody_Template(t *testing.T) {
	body := buildSendBody("5511999999999", "", "", "", "welcome", "pt_BR", `[{"type":"body","parameters":[{"type":"text","text":"John"}]}]`)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "template" {
		t.Fatalf("expected template, got %v", m["type"])
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5511999999999", "+5511999999999"},
		{"+5511999999999", "+5511999999999"},
		{" 5511999999999 ", "+5511999999999"},
	}
	for _, tt := range tests {
		got := normalizePhone(tt.input)
		if got != tt.expected {
			t.Errorf("normalizePhone(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeWaSourceID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"559992032709", "559992032709"},
		{"559992032709@s.whatsapp.net", "559992032709"},
		{"559992032709:7@s.whatsapp.net", "559992032709"},
		{"559992032709:5", "559992032709"},
		{"5511999999999@g.us", "5511999999999"},
		{" 559992032709 ", "559992032709"},
		{"not-a-jid", "not-a-jid"},
	}
	for _, tt := range tests {
		got := normalizeWaSourceID(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeWaSourceID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
