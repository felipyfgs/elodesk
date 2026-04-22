package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"backend/internal/model"
)

// OutboundInput is the channel-agnostic input to Send; it flattens whatever
// came in via channel.OutboundMessage plus the persisted channel record.
type OutboundInput struct {
	Channel          *model.ChannelTwilio
	AuthToken        string
	To               string
	Body             string
	MediaURLs        []string
	ContentSID       string
	ContentVariables map[string]string
	StatusCallback   string
}

// Send dispatches an outbound Twilio message, automatically choosing sender
// (phone_number vs messaging_service_sid) and prefixing whatsapp: when needed.
func Send(ctx context.Context, client *Client, in OutboundInput) (string, error) {
	if client == nil {
		return "", fmt.Errorf("twilio send: nil client")
	}
	if in.Channel == nil {
		return "", fmt.Errorf("twilio send: nil channel")
	}
	if in.To == "" {
		return "", fmt.Errorf("twilio send: missing recipient")
	}

	to := in.To
	from := ""
	if in.Channel.PhoneNumber != nil {
		from = *in.Channel.PhoneNumber
	}
	mss := ""
	if in.Channel.MessagingServiceSID != nil {
		mss = *in.Channel.MessagingServiceSID
	}
	if in.Channel.Medium == model.TwilioMediumWhatsApp {
		if !strings.HasPrefix(to, WhatsappPrefix) {
			to = WhatsappPrefix + to
		}
		if from != "" && !strings.HasPrefix(from, WhatsappPrefix) {
			from = WhatsappPrefix + from
		}
	}

	apiKey := ""
	if in.Channel.APIKeySID != nil {
		apiKey = *in.Channel.APIKeySID
	}

	opts := SendOptions{
		AccountSID:          in.Channel.AccountSID,
		APIKeySID:           apiKey,
		AuthToken:           in.AuthToken,
		From:                from,
		MessagingServiceSID: mss,
		To:                  to,
		Body:                in.Body,
		MediaURLs:           in.MediaURLs,
		ContentSID:          in.ContentSID,
		StatusCallback:      in.StatusCallback,
	}
	if len(in.ContentVariables) > 0 {
		if b, err := json.Marshal(in.ContentVariables); err == nil {
			opts.ContentVariables = string(b)
		}
	}

	resp, err := client.SendMessage(ctx, opts)
	if err != nil {
		return "", err
	}
	return resp.SID, nil
}
