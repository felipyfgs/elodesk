package facebook

import (
	"context"
	"fmt"

	"backend/internal/channel/meta"
	"backend/internal/model"
)

const facebookGraphBase = "https://graph.facebook.com"

type sendResponse struct {
	MessageID string `json:"message_id"`
	Messages  []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// SendRequest holds the parameters for sending a Facebook Messenger message.
type SendRequest struct {
	To           string
	Content      string
	MediaURL     string
	MediaType    string
	QuickReplies []QuickReply
	MessagingTag string
}

type QuickReply struct {
	ContentType string `json:"content_type"`
	Title       string `json:"title"`
	Payload     string `json:"payload"`
}

// Send posts a message to the Facebook Messenger Send API.
func Send(ctx context.Context, ch *model.ChannelFacebookPage, accessToken, appSecret string, req SendRequest) (string, error) {
	client := meta.NewClient(facebookGraphBase)

	body := buildSendBody(req)

	path := fmt.Sprintf("/%s/messages?appsecret_proof=%s",
		ch.PageID,
		meta.AppSecretProof(accessToken, appSecret),
	)

	var resp sendResponse
	if err := client.Post(ctx, path, accessToken, body, &resp); err != nil {
		return "", fmt.Errorf("facebook send: %w", err)
	}

	if resp.MessageID != "" {
		return resp.MessageID, nil
	}
	if len(resp.Messages) > 0 {
		return resp.Messages[0].ID, nil
	}
	return "", fmt.Errorf("facebook send: no message_id in response")
}

func buildSendBody(req SendRequest) map[string]any {
	recipient := map[string]any{"id": req.To}

	var message map[string]any
	if req.MediaURL != "" {
		fileType := req.MediaType
		if fileType == "" {
			fileType = "file"
		}
		message = map[string]any{
			"attachment": map[string]any{
				"type":    fileType,
				"payload": map[string]any{"url": req.MediaURL, "is_reusable": false},
			},
		}
	} else {
		message = map[string]any{"text": req.Content}
	}

	if len(req.QuickReplies) > 0 {
		message["quick_replies"] = req.QuickReplies
	}

	body := map[string]any{
		"recipient": recipient,
		"message":   message,
	}
	if req.MessagingTag != "" {
		body["messaging_type"] = "MESSAGE_TAG"
		body["tag"] = req.MessagingTag
	}
	return body
}
