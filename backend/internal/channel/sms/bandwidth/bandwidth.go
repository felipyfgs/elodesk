package bandwidth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"backend/internal/channel/sms"
	"backend/internal/crypto"
	"backend/internal/model"
)

const (
	bandwidthAPIBase = "https://messaging.bandwidth.com/api/v2"
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
	return "bandwidth"
}

func (p *Provider) VerifyWebhook(r *http.Request, channel *model.ChannelSMS) error {
	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return fmt.Errorf("bandwidth: decrypt config: %w", err)
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("bandwidth: missing basic auth")
	}
	if user != config.BasicAuthUser || pass != config.BasicAuthPass {
		return fmt.Errorf("bandwidth: basic auth mismatch")
	}

	return nil
}

func (p *Provider) ParseInbound(r *http.Request) (*sms.InboundMessage, error) {
	var payload []bandwidthEvent
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("bandwidth: decode body: %w", err)
	}

	for _, evt := range payload {
		if evt.Type == "message-received" && evt.Message != nil {
			msg := &sms.InboundMessage{
				SourceID: evt.Message.ID,
				From:     evt.Message.From,
				To:       firstString(evt.Message.To),
				Content:  evt.Message.Text,
			}

			if evt.Message.Time != "" {
				t, _ := http.ParseTime(evt.Message.Time)
				msg.Timestamp = t.Unix()
			}

			if len(evt.Message.Media) > 0 {
				msg.MediaURLs = make([]string, len(evt.Message.Media))
				copy(msg.MediaURLs, evt.Message.Media)
			}

			return msg, nil
		}
	}

	return nil, fmt.Errorf("bandwidth: no message-received event found")
}

func (p *Provider) Send(ctx context.Context, channel *model.ChannelSMS, out *sms.OutboundMessage, statusCallbackURL string) (string, error) {
	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return "", fmt.Errorf("bandwidth: decrypt config: %w", err)
	}

	endpoint := fmt.Sprintf("%s/users/%s/messages", bandwidthAPIBase, config.AccountID)

	payload := bandwidthSendRequest{
		ApplicationID: config.ApplicationID,
		To:            []string{out.To},
		From:          channel.PhoneNumber,
		Text:          out.Content,
		Tag:           statusCallbackURL,
	}

	if len(out.MediaURL) > 0 {
		payload.Media = out.MediaURL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("bandwidth: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("bandwidth: create request: %w", err)
	}
	req.SetBasicAuth(config.AccountID, config.BasicAuthPass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("bandwidth: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return "", &sms.ProviderError{StatusCode: 429, Message: "rate limited"}
	}
	if resp.StatusCode >= 400 {
		return "", &sms.ProviderError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	var result bandwidthSendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("bandwidth: parse response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("bandwidth: empty id in response")
	}

	return result.ID, nil
}

func (p *Provider) ParseDeliveryStatus(r *http.Request) (*sms.StatusCallback, error) {
	var payload []bandwidthEvent
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("bandwidth: decode status body: %w", err)
	}

	for _, evt := range payload {
		switch evt.Type {
		case "message-delivered":
			if evt.Message != nil {
				return &sms.StatusCallback{
					SourceID: evt.Message.ID,
					Status:   "delivered",
				}, nil
			}
		case "message-failed":
			if evt.Message != nil {
				return &sms.StatusCallback{
					SourceID: evt.Message.ID,
					Status:   "failed",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("bandwidth: no delivery status event found")
}

func (p *Provider) ValidateCredentials(ctx context.Context, config sms.ProviderConfig) error {
	if config.Bandwidth == nil {
		return fmt.Errorf("bandwidth: missing config")
	}

	endpoint := fmt.Sprintf("%s/users/%s/applications/%s",
		bandwidthAPIBase, config.Bandwidth.AccountID, config.Bandwidth.ApplicationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("bandwidth: create validate request: %w", err)
	}
	req.SetBasicAuth(config.Bandwidth.AccountID, config.Bandwidth.BasicAuthPass)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bandwidth: validate request: %w", err)
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

func (p *Provider) decryptConfig(ciphertext string) (*sms.BandwidthConfig, error) {
	raw, err := p.cipher.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	pc, err := sms.ParseProviderConfig("bandwidth", raw)
	if err != nil {
		return nil, err
	}
	return pc.Bandwidth, nil
}

type bandwidthEvent struct {
	Type    string            `json:"type"`
	Time    string            `json:"time"`
	Message *bandwidthMessage `json:"message,omitempty"`
}

type bandwidthMessage struct {
	ID    string   `json:"id"`
	From  string   `json:"from"`
	To    []string `json:"to"`
	Text  string   `json:"text"`
	Media []string `json:"media"`
	Time  string   `json:"time"`
	Tag   string   `json:"tag"`
}

type bandwidthSendRequest struct {
	ApplicationID string   `json:"applicationId"`
	To            []string `json:"to"`
	From          string   `json:"from"`
	Text          string   `json:"text"`
	Media         []string `json:"media,omitempty"`
	Tag           string   `json:"tag,omitempty"`
}

type bandwidthSendResponse struct {
	ID string `json:"id"`
}

func firstString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
