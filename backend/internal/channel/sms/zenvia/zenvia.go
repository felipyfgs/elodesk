package zenvia

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"backend/internal/channel/sms"
	"backend/internal/crypto"
	"backend/internal/model"
)

const (
	zenviaAPIBase = "https://api.zenvia.com"
)

type Provider struct {
	httpClient *http.Client
	cipher     *crypto.Cipher
}

func New(httpClient *http.Client, cipher *crypto.Cipher) *Provider {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Provider{httpClient: httpClient, cipher: cipher}
}

func (p *Provider) Name() string {
	return "zenvia"
}

func (p *Provider) VerifyWebhook(r *http.Request, channel *model.ChannelSMS) error {
	signature := r.Header.Get("X-Zenvia-Signature")
	if signature == "" {
		return fmt.Errorf("zenvia: missing X-Zenvia-Signature header")
	}

	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return fmt.Errorf("zenvia: decrypt config: %w", err)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("zenvia: read body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	mac := hmac.New(sha256.New, []byte(config.APIToken))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("zenvia: signature mismatch")
	}

	return nil
}

func (p *Provider) ParseInbound(r *http.Request) (*sms.InboundMessage, error) {
	var payload zenviaInbound
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("zenvia: decode body: %w", err)
	}

	msg := &sms.InboundMessage{
		SourceID: payload.ID,
		From:     payload.From,
		To:       payload.To,
	}

	for _, c := range payload.Contents {
		if c.Type == "text" && msg.Content == "" {
			msg.Content = c.Text
		}
		if c.Type == "media" && c.Payload != nil {
			if c.Payload.MediaURL != "" {
				msg.MediaURLs = append(msg.MediaURLs, c.Payload.MediaURL)
				msg.MediaTypes = append(msg.MediaTypes, c.Payload.MediaType)
			}
		}
	}

	return msg, nil
}

func (p *Provider) Send(ctx context.Context, channel *model.ChannelSMS, out *sms.OutboundMessage, statusCallbackURL string) (string, error) {
	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return "", fmt.Errorf("zenvia: decrypt config: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/channels/sms/messages", zenviaAPIBase)

	contents := []zenviaContent{
		{Type: "text", Text: out.Content},
	}

	for _, m := range out.MediaURL {
		contents = append(contents, zenviaContent{
			Type: "media",
			Payload: &zenviaPayload{
				MediaURL: m,
			},
		})
	}

	payload := zenviaSendRequest{
		From:     channel.PhoneNumber,
		To:       out.To,
		Contents: contents,
	}

	if statusCallbackURL != "" {
		payload.NotificationURL = statusCallbackURL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("zenvia: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("zenvia: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-TOKEN", config.APIToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("zenvia: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return "", &sms.ProviderError{StatusCode: 429, Message: "rate limited"}
	}
	if resp.StatusCode >= 400 {
		return "", &sms.ProviderError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	var result zenviaSendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("zenvia: parse response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("zenvia: empty id in response")
	}

	return result.ID, nil
}

func (p *Provider) ParseDeliveryStatus(r *http.Request) (*sms.StatusCallback, error) {
	var payload zenviaStatus
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("zenvia: decode status body: %w", err)
	}

	cb := &sms.StatusCallback{
		SourceID: payload.MessageID,
	}

	switch payload.MessageStatus.Code {
	case "SENT":
		cb.Status = "sent"
	case "DELIVERED":
		cb.Status = "delivered"
	case "NOT_DELIVERED":
		cb.Status = "failed"
	default:
		cb.Status = "sent"
	}

	return cb, nil
}

func (p *Provider) ValidateCredentials(ctx context.Context, config sms.ProviderConfig) error {
	if config.Zenvia == nil {
		return fmt.Errorf("zenvia: missing config")
	}

	endpoint := fmt.Sprintf("%s/v2/channels", zenviaAPIBase)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("zenvia: create validate request: %w", err)
	}
	req.Header.Set("X-API-TOKEN", config.Zenvia.APIToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("zenvia: validate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &sms.ProviderError{StatusCode: resp.StatusCode, Message: "invalid credentials"}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &sms.ProviderError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	return nil
}

func (p *Provider) decryptConfig(ciphertext string) (*sms.ZenviaConfig, error) {
	raw, err := p.cipher.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	pc, err := sms.ParseProviderConfig("zenvia", raw)
	if err != nil {
		return nil, err
	}
	return pc.Zenvia, nil
}

type zenviaInbound struct {
	ID       string          `json:"id"`
	From     string          `json:"from"`
	To       string          `json:"to"`
	Contents []zenviaContent `json:"contents"`
}

type zenviaContent struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Payload *zenviaPayload `json:"payload,omitempty"`
}

type zenviaPayload struct {
	MediaURL  string `json:"mediaUrl"`
	MediaType string `json:"mediaType,omitempty"`
}

type zenviaSendRequest struct {
	From            string          `json:"from"`
	To              string          `json:"to"`
	Contents        []zenviaContent `json:"contents"`
	NotificationURL string          `json:"notificationUrl,omitempty"`
}

type zenviaSendResponse struct {
	ID string `json:"id"`
}

type zenviaStatus struct {
	MessageID     string              `json:"messageId"`
	MessageStatus zenviaMessageStatus `json:"messageStatus"`
}

type zenviaMessageStatus struct {
	Code string `json:"code"`
}
