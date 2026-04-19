package telegram

import (
	"testing"
)

func TestParseContentAttrs_ReplyTo(t *testing.T) {
	raw := `{"in_reply_to_source_id":"123"}`
	replyTo, buttons := parseContentAttrs(raw)
	if replyTo != 123 {
		t.Errorf("expected replyTo=123, got %d", replyTo)
	}
	if buttons != nil {
		t.Errorf("expected no buttons, got %+v", buttons)
	}
}

func TestParseContentAttrs_Buttons(t *testing.T) {
	raw := `{"buttons":[[{"text":"Sim","callbackData":"yes"},{"text":"Não","callbackData":"no"}]]}`
	replyTo, buttons := parseContentAttrs(raw)
	if replyTo != 0 {
		t.Errorf("expected replyTo=0, got %d", replyTo)
	}
	if buttons == nil {
		t.Fatal("expected buttons")
	}
	if len(buttons.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row, got %d", len(buttons.InlineKeyboard))
	}
	if len(buttons.InlineKeyboard[0]) != 2 {
		t.Fatalf("expected 2 buttons in row, got %d", len(buttons.InlineKeyboard[0]))
	}
	if buttons.InlineKeyboard[0][0].Text != "Sim" {
		t.Errorf("expected first button text 'Sim', got %q", buttons.InlineKeyboard[0][0].Text)
	}
	if buttons.InlineKeyboard[0][0].CallbackData != "yes" {
		t.Errorf("expected first button callback_data 'yes', got %q", buttons.InlineKeyboard[0][0].CallbackData)
	}
}

func TestParseContentAttrs_Both(t *testing.T) {
	raw := `{"in_reply_to_source_id":"456","buttons":[[{"text":"OK","callbackData":"ok"}]]}`
	replyTo, buttons := parseContentAttrs(raw)
	if replyTo != 456 {
		t.Errorf("expected replyTo=456, got %d", replyTo)
	}
	if buttons == nil {
		t.Fatal("expected buttons")
	}
	if buttons.InlineKeyboard[0][0].CallbackData != "ok" {
		t.Errorf("expected callback_data 'ok', got %q", buttons.InlineKeyboard[0][0].CallbackData)
	}
}

func TestParseContentAttrs_Empty(t *testing.T) {
	replyTo, buttons := parseContentAttrs("")
	if replyTo != 0 {
		t.Errorf("expected replyTo=0, got %d", replyTo)
	}
	if buttons != nil {
		t.Errorf("expected nil buttons, got %+v", buttons)
	}
}

func TestParseContentAttrs_InvalidJSON(t *testing.T) {
	replyTo, buttons := parseContentAttrs("not json")
	if replyTo != 0 {
		t.Errorf("expected replyTo=0, got %d", replyTo)
	}
	if buttons != nil {
		t.Errorf("expected nil buttons, got %+v", buttons)
	}
}

func TestParseContentAttrs_URLButton(t *testing.T) {
	raw := `{"buttons":[[{"text":"Visit","url":"https://example.com"}]]}`
	_, buttons := parseContentAttrs(raw)
	if buttons == nil {
		t.Fatal("expected buttons")
	}
	if buttons.InlineKeyboard[0][0].URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", buttons.InlineKeyboard[0][0].URL)
	}
}
