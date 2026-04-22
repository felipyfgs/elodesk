package tiktok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// SendText pushes a text message over a TikTok conversation.
func SendText(ctx context.Context, api *APIClient, accessToken, businessID, conversationID, text string, referencedMessageID string) (string, error) {
	if accessToken == "" {
		return "", fmt.Errorf("tiktok send: empty access token")
	}
	if conversationID == "" {
		return "", fmt.Errorf("tiktok send: empty conversation id")
	}
	body := SendMessageRequest{
		BusinessID:    businessID,
		RecipientType: "CONVERSATION",
		Recipient:     conversationID,
		MessageType:   "TEXT",
		Text:          &TextBody{Body: text},
	}
	_ = referencedMessageID // placeholder until referenced_message_info is threaded through
	info, err := api.SendMessage(ctx, accessToken, body)
	if err != nil {
		return "", err
	}
	return info.MessageID, nil
}

// ReauthErrorFor wraps a SendMessage error and bubbles up ErrReauthRequired when
// the API signals that the token is no longer valid.
func ReauthErrorFor(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrReauthRequired) {
		return err
	}
	return nil
}

// ConversationIDFromAttrs returns the conversation_id stored on a message's
// content_attributes JSON blob, falling back to the empty string when absent.
func ConversationIDFromAttrs(raw string) string {
	if raw == "" {
		return ""
	}
	var attrs map[string]any
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return ""
	}
	if v, ok := attrs["tiktok_conversation_id"].(string); ok {
		return v
	}
	return ""
}
