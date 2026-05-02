package tiktok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

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
	// TODO: Implement referenced_message_id threading.
	// TikTok does not currently expose referenced_message_info in its messaging API.
	// This requires model changes and API support before it can be wired.
	_ = referencedMessageID
	info, err := api.SendMessage(ctx, accessToken, body)
	if err != nil {
		return "", err
	}
	return info.MessageID, nil
}

func ReauthErrorFor(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrReauthRequired) {
		return err
	}
	return nil
}

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
