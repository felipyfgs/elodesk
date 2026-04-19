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

const dialog360BaseURL = "https://waba.360dialog.io/v1"

type Dialog360Provider struct {
	httpClient *http.Client
	baseURL    string
}

func NewDialog360Provider(httpClient *http.Client) *Dialog360Provider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Dialog360Provider{httpClient: httpClient, baseURL: dialog360BaseURL}
}

func NewDialog360ProviderWithURL(httpClient *http.Client, baseURL string) *Dialog360Provider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Dialog360Provider{httpClient: httpClient, baseURL: baseURL}
}

func (d *Dialog360Provider) VerifyHandshake(_ context.Context, _ map[string]string, _ string) (string, bool) {
	return "", false
}

func (d *Dialog360Provider) VerifySignature(_ context.Context, _ []byte, _ map[string]string, _ string) bool {
	return true
}

func (d *Dialog360Provider) ParsePayload(_ context.Context, body []byte) (*channel.InboundResult, error) {
	var payload Dialog360Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse dialog360 payload: %w", err)
	}
	result := &channel.InboundResult{}
	for _, msg := range payload.Messages {
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
				}
			}
		}
		result.Messages = append(result.Messages, im)
	}
	for _, st := range payload.Statuses {
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
		result.Statuses = append(result.Statuses, su)
	}
	return result, nil
}

func (d *Dialog360Provider) Send(ctx context.Context, apiKey, to, content, mediaURL, mediaType, templateName, templateLang, templateComponents string) (string, error) {
	endpoint := d.baseURL + "/messages"
	body := buildSendBody(to, content, mediaURL, mediaType, templateName, templateLang, templateComponents)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("dialog360: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("D360-API-KEY", apiKey)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("dialog360: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", &ProviderError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var sendResp SendResponse
	if err := json.Unmarshal(respBody, &sendResp); err != nil {
		return "", fmt.Errorf("dialog360: unmarshal send response: %w", err)
	}
	if len(sendResp.Messages) > 0 {
		return sendResp.Messages[0].ID, nil
	}
	return "", fmt.Errorf("dialog360: no message id in response")
}

func (d *Dialog360Provider) SyncTemplates(ctx context.Context, apiKey, _, _ string) ([]channel.Template, error) {
	endpoint := d.baseURL + "/configs/templates"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("dialog360: build templates request: %w", err)
	}
	req.Header.Set("D360-API-KEY", apiKey)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dialog360: sync templates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	var templatesResp Dialog360TemplatesResponse
	if err := json.Unmarshal(respBody, &templatesResp); err != nil {
		return nil, fmt.Errorf("dialog360: unmarshal templates: %w", err)
	}

	var templates []channel.Template
	for _, t := range templatesResp.WabaTemplates {
		templates = append(templates, channel.Template{
			Name:     t.Name,
			Language: t.Language,
			Status:   t.Status,
		})
	}
	return templates, nil
}

func (d *Dialog360Provider) HeadersForRequest(apiKey string) map[string]string {
	return map[string]string{
		"D360-API-KEY": apiKey,
		"Content-Type": "application/json",
	}
}
