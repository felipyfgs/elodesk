package email

import (
	"context"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"backend/internal/model"
)

// SendGmail sends an outbound email via the Gmail API.
func SendGmail(ctx context.Context, ch *model.ChannelEmail, msg *OutboundEmail, decryptFn func(string) (string, error)) (sourceID string, err error) {
	if ch.ProviderConfig == nil {
		return "", fmt.Errorf("gmail: no provider_config")
	}

	configPlain, err := decryptFn(*ch.ProviderConfig)
	if err != nil {
		return "", fmt.Errorf("gmail: decrypt config: %w", err)
	}
	tokens, err := UnmarshalTokens(configPlain)
	if err != nil {
		return "", fmt.Errorf("gmail: unmarshal tokens: %w", err)
	}

	if tokens.NeedsRefresh() {
		tokens, err = GoogleRefreshToken(ctx, tokens.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("gmail: refresh token: %w", err)
		}
	}

	oauthCfg := googleOAuthConfig()
	httpClient := oauthCfg.Client(ctx, &oauth2.Token{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.ExpiresOn,
		TokenType:    "Bearer",
	})
	_ = google.Endpoint // ensure import is used

	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return "", fmt.Errorf("gmail: create service: %w", err)
	}

	if msg.MessageID == "" {
		msg.MessageID = generateMessageID(ch.Email)
	}
	raw := buildRawMessage(msg)

	encoded := base64.URLEncoding.EncodeToString([]byte(raw))
	gmailMsg := &gmail.Message{Raw: encoded}
	if msg.InReplyTo != "" {
		gmailMsg.ThreadId = "" // Gmail uses threadId; left empty — Gmail infers from headers
	}

	sent, err := svc.Users.Messages.Send("me", gmailMsg).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("gmail: send: %w", err)
	}
	return sent.Id, nil
}
