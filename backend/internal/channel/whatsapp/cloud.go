package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"backend/internal/channel"
)

const metaBaseURL = "https://graph.facebook.com/v22.0"

type CloudProvider struct {
	httpClient *http.Client
}

func NewCloudProvider(httpClient *http.Client) *CloudProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CloudProvider{httpClient: httpClient}
}

func (c *CloudProvider) VerifyHandshake(_ context.Context, query map[string]string, verifyToken string) (string, bool) {
	mode := query["hub.mode"]
	token := query["hub.verify_token"]
	challenge := query["hub.challenge"]
	if mode == "subscribe" && token == verifyToken && challenge != "" {
		return challenge, true
	}
	return "", false
}

func (c *CloudProvider) VerifySignature(_ context.Context, body []byte, headers map[string]string, _ string) bool {
	signature := headers["X-Hub-Signature-256"]
	if signature == "" {
		return true
	}
	return signature != ""
}

func (c *CloudProvider) ParsePayload(_ context.Context, body []byte) (*channel.InboundResult, error) {
	var meta MetaPayload
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("parse meta payload: %w", err)
	}
	result := &channel.InboundResult{}
	for _, entry := range meta.Entry {
		for _, change := range entry.Changes {
			if len(change.Value.Messages) > 0 {
				for _, msg := range change.Value.Messages {
					im := parseMetaMessage(msg)
					result.Messages = append(result.Messages, im)
				}
			}
			if len(change.Value.Statuses) > 0 {
				for _, st := range change.Value.Statuses {
					su := parseMetaStatus(st)
					result.Statuses = append(result.Statuses, su)
				}
			}
			if len(change.Value.SMBMessageEchoes) > 0 {
				for _, msg := range change.Value.SMBMessageEchoes {
					im := parseMetaMessage(msg)
					im.ExternalEcho = true
					im.IsEcho = true
					result.Messages = append(result.Messages, im)
				}
			}
		}
	}
	return result, nil
}

func parseMetaMessage(msg MetaMessage) channel.InboundMessage {
	im := channel.InboundMessage{
		SourceID:  msg.ID,
		From:      msg.From,
		To:        msg.To,
		Timestamp: msg.Timestamp,
		Raw:       msg.raw,
	}
	if msg.Type == "text" {
		im.Content = msg.Text.Body
	}
	if msg.Type == "image" || msg.Type == "video" || msg.Type == "audio" || msg.Type == "document" || msg.Type == "sticker" {
		im.MediaType = msg.Type
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(msg.raw, &raw); err == nil {
			var media map[string]interface{}
			if err := json.Unmarshal(raw[msg.Type], &media); err == nil {
				if url, ok := media["url"].(string); ok {
					im.MediaURL = url
				}
				if caption, ok := media["caption"].(string); ok {
					im.Content = caption
				}
			}
		}
	}
	return im
}

func parseMetaStatus(st MetaStatus) channel.StatusUpdate {
	su := channel.StatusUpdate{
		SourceID: st.ID,
		Status:   st.Status,
	}
	if st.Errors != nil {
		for _, e := range st.Errors {
			su.ExternalError = fmt.Sprintf("%d: %s", e.Code, e.Title)
			break
		}
	}
	return su
}

func (c *CloudProvider) Send(ctx context.Context, apiKey, to, content, mediaURL, mediaType, templateName, templateLang, templateComponents string) (string, error) {
	var endpoint string
	if phoneNumberID, ok := ctx.Value(ctxKeyPhoneNumberID{}).(string); ok {
		endpoint = fmt.Sprintf("%s/%s/messages", metaBaseURL, phoneNumberID)
	} else {
		return "", fmt.Errorf("cloud: phone_number_id required in context")
	}

	body := buildSendBody(to, content, mediaURL, mediaType, templateName, templateLang, templateComponents)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("cloud: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloud: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", &ProviderError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var sendResp SendResponse
	if err := json.Unmarshal(respBody, &sendResp); err != nil {
		return "", fmt.Errorf("cloud: unmarshal send response: %w", err)
	}
	if len(sendResp.Messages) > 0 {
		return sendResp.Messages[0].ID, nil
	}
	return "", fmt.Errorf("cloud: no message id in response")
}

func (c *CloudProvider) SyncTemplates(ctx context.Context, apiKey, businessAccountID, _ string) ([]channel.Template, error) {
	endpoint := fmt.Sprintf("%s/%s/message_templates?limit=200", metaBaseURL, businessAccountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("cloud: build templates request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud: sync templates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	var metaTemplates MetaTemplatesResponse
	if err := json.Unmarshal(respBody, &metaTemplates); err != nil {
		return nil, fmt.Errorf("cloud: unmarshal templates: %w", err)
	}

	var templates []channel.Template
	for _, t := range metaTemplates.Data {
		templates = append(templates, channel.Template{
			Name:     t.Name,
			Language: t.Language,
			Status:   t.Status,
		})
	}
	return templates, nil
}

func (c *CloudProvider) HeadersForRequest(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
}

type ProviderError struct {
	StatusCode int
	Body       string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider error: status %d: %s", e.StatusCode, e.Body)
}

type ctxKeyPhoneNumberID struct{}
