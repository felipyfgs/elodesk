package line

import (
	"context"
	"encoding/json"
	"fmt"
)

type SendOptions struct {
	To           string
	ReplyToken   string
	Content      string
	MediaURL     string
	MediaType    string
	ContentAttrs string
}

func Send(ctx context.Context, api *APIClient, channelToken string, opts SendOptions) (string, error) {
	if channelToken == "" {
		return "", fmt.Errorf("line send: empty channel token")
	}

	messages := buildOutboundMessages(opts)
	if len(messages) == 0 {
		return "", fmt.Errorf("line send: no content to deliver")
	}

	replyToken := replyTokenFromAttrs(opts.ContentAttrs)
	if opts.ReplyToken != "" {
		replyToken = opts.ReplyToken
	}

	if replyToken != "" {
		if err := api.Reply(ctx, channelToken, ReplyRequest{ReplyToken: replyToken, Messages: messages}); err != nil {
			return "", err
		}
		return "", nil
	}

	if opts.To == "" {
		return "", fmt.Errorf("line send: missing push recipient")
	}
	if err := api.Push(ctx, channelToken, PushRequest{To: opts.To, Messages: messages}); err != nil {
		return "", err
	}
	return "", nil
}

func buildOutboundMessages(opts SendOptions) []Message {
	msgs := make([]Message, 0, 2)
	if opts.Content != "" {
		msgs = append(msgs, Message{Type: MessageTypeText, Text: opts.Content})
	}
	if opts.MediaURL != "" {
		switch opts.MediaType {
		case MessageTypeImage:
			msgs = append(msgs, Message{
				Type:               MessageTypeImage,
				OriginalContentURL: opts.MediaURL,
				PreviewImageURL:    opts.MediaURL,
			})
		case MessageTypeVideo:
			msgs = append(msgs, Message{
				Type:               MessageTypeVideo,
				OriginalContentURL: opts.MediaURL,
				PreviewImageURL:    opts.MediaURL,
			})
		}
	}
	return msgs
}

func replyTokenFromAttrs(raw string) string {
	if raw == "" {
		return ""
	}
	var attrs map[string]any
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return ""
	}
	if v, ok := attrs[replyTokenAttrsKey].(string); ok {
		return v
	}
	return ""
}
