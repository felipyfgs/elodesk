package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"backend/internal/model"
)

const graphSendMailURL = "https://graph.microsoft.com/v1.0/me/sendMail"

// SendGraph sends an outbound email via the Microsoft Graph API.
func SendGraph(ctx context.Context, ch *model.ChannelEmail, msg *OutboundEmail, decryptFn func(string) (string, error)) (sourceID string, err error) {
	if ch.ProviderConfig == nil {
		return "", fmt.Errorf("microsoft: no provider_config")
	}

	configPlain, err := decryptFn(*ch.ProviderConfig)
	if err != nil {
		return "", fmt.Errorf("microsoft: decrypt config: %w", err)
	}
	tokens, err := UnmarshalTokens(configPlain)
	if err != nil {
		return "", fmt.Errorf("microsoft: unmarshal tokens: %w", err)
	}

	if tokens.NeedsRefresh() {
		tokens, err = MicrosoftRefreshToken(ctx, tokens.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("microsoft: refresh token: %w", err)
		}
	}

	if msg.MessageID == "" {
		msg.MessageID = generateMessageID(ch.Email)
	}

	body := graphBody(msg)
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("microsoft: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphSendMailURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("microsoft: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("microsoft: send: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("microsoft: send returned HTTP %d", resp.StatusCode)
	}
	return msg.MessageID, nil
}

type graphEmailAddress struct {
	Address string `json:"address"`
}
type graphRecipient struct {
	EmailAddress graphEmailAddress `json:"emailAddress"`
}
type graphBodyContent struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}
type graphMessage struct {
	Subject                string           `json:"subject"`
	Body                   graphBodyContent `json:"body"`
	ToRecipients           []graphRecipient `json:"toRecipients"`
	InternetMessageID      string           `json:"internetMessageId,omitempty"`
	InternetMessageHeaders []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"internetMessageHeaders,omitempty"`
}
type graphSendMailBody struct {
	Message graphMessage `json:"message"`
}

func graphBody(msg *OutboundEmail) graphSendMailBody {
	recipients := make([]graphRecipient, len(msg.To))
	for i, addr := range msg.To {
		recipients[i] = graphRecipient{EmailAddress: graphEmailAddress{Address: addr}}
	}
	contentType := "Text"
	content := msg.TextBody
	if msg.HTMLBody != "" {
		contentType = "HTML"
		content = msg.HTMLBody
	}
	var headers []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if msg.InReplyTo != "" {
		headers = append(headers, struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: "In-Reply-To", Value: msg.InReplyTo})
	}
	if msg.References != "" {
		headers = append(headers, struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: "References", Value: msg.References})
	}
	return graphSendMailBody{
		Message: graphMessage{
			Subject:                msg.Subject,
			Body:                   graphBodyContent{ContentType: contentType, Content: content},
			ToRecipients:           recipients,
			InternetMessageID:      msg.MessageID,
			InternetMessageHeaders: headers,
		},
	}
}
