package telegram

import (
	"encoding/json"
	"testing"
)

func TestProcessWebhook_TextMessage(t *testing.T) {
	update := Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 100,
			From: &User{
				ID:        42,
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			Chat: Chat{ID: 42, Type: "private"},
			Date: 1700000000,
			Text: strPtr("hello world"),
		},
	}

	body, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Update
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Message == nil {
		t.Fatal("expected message to be parsed")
	}
	if parsed.Message.Text == nil || *parsed.Message.Text != "hello world" {
		t.Errorf("expected text 'hello world', got %v", parsed.Message.Text)
	}
}

func TestProcessWebhook_PhotoMessage(t *testing.T) {
	update := Update{
		UpdateID: 2,
		Message: &Message{
			MessageID: 101,
			From: &User{
				ID:        42,
				FirstName: "John",
			},
			Chat: Chat{ID: 42, Type: "private"},
			Date: 1700000000,
			Photo: []Photo{
				{FileID: "small", FileUID: "u1", Width: 100, Height: 100},
				{FileID: "large", FileUID: "u2", Width: 800, Height: 600, FileSize: 50000},
			},
		},
	}

	body, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Update
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}

	if len(parsed.Message.Photo) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(parsed.Message.Photo))
	}
}

func TestProcessWebhook_GroupIgnored(t *testing.T) {
	update := Update{
		UpdateID: 3,
		Message: &Message{
			MessageID: 102,
			From: &User{
				ID:        42,
				FirstName: "John",
			},
			Chat: Chat{ID: 100, Type: "group", Title: "Test Group"},
			Date: 1700000000,
			Text: strPtr("hello group"),
		},
	}

	body, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Update
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Message.Chat.Type != "group" {
		t.Errorf("expected chat type 'group', got %s", parsed.Message.Chat.Type)
	}
}

func TestProcessWebhook_EditedMessage(t *testing.T) {
	update := Update{
		UpdateID: 4,
		EditedMessage: &Message{
			MessageID: 103,
			From: &User{
				ID:        42,
				FirstName: "John",
			},
			Chat: Chat{ID: 42, Type: "private"},
			Date: 1700000000,
			Text: strPtr("edited text"),
		},
	}

	body, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Update
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.EditedMessage == nil {
		t.Fatal("expected edited_message to be parsed")
	}
	if parsed.Message != nil {
		t.Error("expected message to be nil for edited_message")
	}
}

func TestProcessWebhook_CallbackQuery(t *testing.T) {
	update := Update{
		UpdateID: 5,
		CallbackQuery: &CallbackQuery{
			ID:   "cb_123",
			From: User{ID: 42, FirstName: "John"},
			Data: "button_click",
			Message: &Message{
				MessageID: 104,
				Chat:      Chat{ID: 42, Type: "private"},
			},
		},
	}

	body, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Update
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.CallbackQuery == nil {
		t.Fatal("expected callback_query to be parsed")
	}
	if parsed.CallbackQuery.Data != "button_click" {
		t.Errorf("expected data 'button_click', got %s", parsed.CallbackQuery.Data)
	}
}

func TestExtractContent_Text(t *testing.T) {
	msg := &Message{Text: strPtr("hello")}
	content, ct, _ := extractContent(msg)
	if content != "hello" {
		t.Errorf("expected 'hello', got %q", content)
	}
	if ct != 0 {
		t.Errorf("expected ContentTypeText, got %d", ct)
	}
}

func TestExtractContent_Photo(t *testing.T) {
	msg := &Message{
		Photo: []Photo{
			{FileID: "sm", Width: 100, Height: 100},
			{FileID: "lg", Width: 800, Height: 600, FileSize: 50000},
		},
	}
	content, ct, attrs := extractContent(msg)
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
	if ct != 9 {
		t.Errorf("expected ContentTypeImage, got %d", ct)
	}
	if attrs == nil {
		t.Fatal("expected content attributes")
	}
}

func TestExtractContent_Unsupported(t *testing.T) {
	msg := &Message{}
	content, ct, _ := extractContent(msg)
	if content != "[unsupported]" {
		t.Errorf("expected '[unsupported]', got %q", content)
	}
	if ct != 0 {
		t.Errorf("expected ContentTypeText, got %d", ct)
	}
}

func strPtr(s string) *string { return &s }
