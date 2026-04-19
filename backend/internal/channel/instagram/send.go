package instagram

import (
	"context"
	"fmt"

	"backend/internal/channel/meta"
	"backend/internal/model"
)

const instagramGraphBase = "https://graph.instagram.com"

type sendResponse struct {
	MessageID string `json:"message_id"`
	Messages  []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// Send posts a message to the Instagram Messaging API and returns the
// provider's message ID as sourceID.
func Send(ctx context.Context, ch *model.ChannelInstagram, accessToken, appSecret, to, content, mediaURL, mediaType string) (string, error) {
	client := meta.NewClient(instagramGraphBase)

	body := buildSendBody(to, content, mediaURL, mediaType)

	path := fmt.Sprintf("/%s/messages?appsecret_proof=%s",
		ch.InstagramID,
		meta.AppSecretProof(accessToken, appSecret),
	)

	var resp sendResponse
	if err := client.Post(ctx, path, accessToken, body, &resp); err != nil {
		return "", fmt.Errorf("instagram send: %w", err)
	}

	if resp.MessageID != "" {
		return resp.MessageID, nil
	}
	if len(resp.Messages) > 0 {
		return resp.Messages[0].ID, nil
	}
	return "", fmt.Errorf("instagram send: no message_id in response")
}

func buildSendBody(to, content, mediaURL, mediaType string) map[string]any {
	recipient := map[string]any{"id": to}

	if mediaURL != "" {
		fileType := mediaType
		if fileType == "" {
			fileType = "file"
		}
		return map[string]any{
			"recipient": recipient,
			"message": map[string]any{
				"attachment": map[string]any{
					"type":    fileType,
					"payload": map[string]any{"url": mediaURL, "is_reusable": false},
				},
			},
		}
	}

	return map[string]any{
		"recipient": recipient,
		"message":   map[string]any{"text": content},
	}
}
