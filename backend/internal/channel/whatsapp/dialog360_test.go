package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDialog360Provider_HeadersForRequest(t *testing.T) {
	p := NewDialog360Provider(nil)
	headers := p.HeadersForRequest("test_key")
	if headers["D360-API-KEY"] != "test_key" {
		t.Fatalf("expected D360-API-KEY header, got %v", headers)
	}
}

func TestDialog360Provider_VerifyHandshake_AlwaysFalse(t *testing.T) {
	p := NewDialog360Provider(nil)
	_, ok := p.VerifyHandshake(context.Background(), nil, "")
	if ok {
		t.Fatal("dialog360 does not support handshake")
	}
}

func TestDialog360Provider_ParsePayload_Text(t *testing.T) {
	p := NewDialog360Provider(nil)
	body := `{
		"object": "whatsapp_business_account",
		"messages": [{
			"from": "5511999999999",
			"id": "wamid.D360",
			"to": "5511888888888",
			"type": "text",
			"timestamp": 1700000000,
			"text": {"body": "Hi from 360"}
		}]
	}`
	result, err := p.ParsePayload(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	if result.Messages[0].SourceID != "wamid.D360" {
		t.Fatalf("expected wamid.D360, got %s", result.Messages[0].SourceID)
	}
	if result.Messages[0].Content != "Hi from 360" {
		t.Fatalf("expected 'Hi from 360', got %s", result.Messages[0].Content)
	}
}

func TestDialog360Provider_Send(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("D360-API-KEY") != "test_key" {
			t.Errorf("expected D360-API-KEY header")
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected /messages path, got %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["to"] != "5511999999999" {
			t.Errorf("expected to 5511999999999, got %v", body["to"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"id": "wamid.RESP1"}},
		})
	}))
	defer srv.Close()

	p := NewDialog360ProviderWithURL(srv.Client(), srv.URL)
	sourceID, err := p.Send(context.Background(), "test_key", "5511999999999", "Hello", "", "", "", "", "")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if sourceID != "wamid.RESP1" {
		t.Fatalf("expected wamid.RESP1, got %s", sourceID)
	}
}

func TestDialog360Provider_SyncTemplates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("D360-API-KEY") != "test_key" {
			t.Errorf("expected D360-API-KEY header")
		}
		if r.URL.Path != "/configs/templates" {
			t.Errorf("expected /configs/templates path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"waba_templates": []map[string]string{
				{"name": "welcome", "language": "pt_BR", "status": "APPROVED"},
			},
		})
	}))
	defer srv.Close()

	p := NewDialog360ProviderWithURL(srv.Client(), srv.URL)
	templates, err := p.SyncTemplates(context.Background(), "test_key", "", "")
	if err != nil {
		t.Fatalf("sync templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Name != "welcome" {
		t.Fatalf("expected welcome, got %s", templates[0].Name)
	}
}
