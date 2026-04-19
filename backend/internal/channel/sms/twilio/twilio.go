package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"backend/internal/channel/sms"
	"backend/internal/crypto"
	"backend/internal/model"
)

const (
	twilioAPIBase = "https://api.twilio.com/2010-04-01"
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
	return "twilio"
}

func (p *Provider) VerifyWebhook(r *http.Request, channel *model.ChannelSMS) error {
	sig := r.Header.Get("X-Twilio-Signature")
	if sig == "" {
		return fmt.Errorf("twilio: missing X-Twilio-Signature header")
	}

	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return fmt.Errorf("twilio: decrypt config: %w", err)
	}

	expected := p.computeSignature(r.URL.String(), r.Form, config.AuthToken)
	if !hmacEqual([]byte(expected), []byte(sig)) {
		return fmt.Errorf("twilio: signature mismatch")
	}

	return nil
}

func (p *Provider) computeSignature(fullURL string, vals url.Values, authToken string) string {
	params := make(url.Values)
	for k, vs := range vals {
		for _, v := range vs {
			params.Add(k, v)
		}
	}

	sortedKeys := make([]string, 0, len(params))
	for k := range params {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var sb strings.Builder
	sb.WriteString(fullURL)
	for _, k := range sortedKeys {
		for _, v := range params[k] {
			sb.WriteString(k)
			sb.WriteString(v)
		}
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (p *Provider) ParseInbound(r *http.Request) (*sms.InboundMessage, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("twilio: parse form: %w", err)
	}

	msg := &sms.InboundMessage{
		SourceID: r.FormValue("MessageSid"),
		From:     r.FormValue("From"),
		To:       r.FormValue("To"),
		Content:  r.FormValue("Body"),
	}

	if ts := r.FormValue("Timestamp"); ts != "" {
		t, _ := http.ParseTime(ts)
		msg.Timestamp = t.Unix()
	}

	numMedia, _ := strconv.Atoi(r.FormValue("NumMedia"))
	if numMedia > 0 {
		msg.MediaURLs = make([]string, 0, numMedia)
		msg.MediaTypes = make([]string, 0, numMedia)
		for i := 0; i < numMedia; i++ {
			msg.MediaURLs = append(msg.MediaURLs, r.FormValue(fmt.Sprintf("MediaUrl%d", i)))
			msg.MediaTypes = append(msg.MediaTypes, r.FormValue(fmt.Sprintf("MediaContentType%d", i)))
		}
	}

	return msg, nil
}

func (p *Provider) Send(ctx context.Context, channel *model.ChannelSMS, out *sms.OutboundMessage, statusCallbackURL string) (string, error) {
	config, err := p.decryptConfig(channel.ProviderConfigCiphertext)
	if err != nil {
		return "", fmt.Errorf("twilio: decrypt config: %w", err)
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", twilioAPIBase, config.AccountSID)

	form := url.Values{}
	if channel.MessagingServiceSid != nil && *channel.MessagingServiceSid != "" {
		form.Set("MessagingServiceSid", *channel.MessagingServiceSid)
	} else {
		form.Set("From", channel.PhoneNumber)
	}
	form.Set("To", out.To)
	form.Set("Body", out.Content)
	if statusCallbackURL != "" {
		form.Set("StatusCallback", statusCallbackURL)
	}
	for _, m := range out.MediaURL {
		form.Add("MediaUrl", m)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("twilio: create request: %w", err)
	}
	req.SetBasicAuth(config.AccountSID, config.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("twilio: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return "", &sms.ProviderError{StatusCode: 429, Message: "rate limited"}
	}
	if resp.StatusCode >= 400 {
		return "", &sms.ProviderError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	var result twilioSendResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("twilio: parse response: %w", err)
	}

	if len(result.SID) == 0 {
		return "", fmt.Errorf("twilio: empty sid in response")
	}

	return result.SID, nil
}

func (p *Provider) ParseDeliveryStatus(r *http.Request) (*sms.StatusCallback, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("twilio: parse status form: %w", err)
	}

	cb := &sms.StatusCallback{
		SourceID: r.FormValue("MessageSid"),
	}

	status := r.FormValue("MessageStatus")
	switch status {
	case "queued", "sent", "delivered", "undelivered", "failed":
		cb.Status = status
	default:
		cb.Status = "sent"
	}

	if errorCode := r.FormValue("ErrorCode"); errorCode != "" {
		cb.ExternalError = errorCode
	}

	return cb, nil
}

func (p *Provider) ValidateCredentials(ctx context.Context, config sms.ProviderConfig) error {
	if config.Twilio == nil {
		return fmt.Errorf("twilio: missing config")
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s.json", twilioAPIBase, config.Twilio.AccountSID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("twilio: create validate request: %w", err)
	}
	req.SetBasicAuth(config.Twilio.AccountSID, config.Twilio.AuthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio: validate request: %w", err)
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

func (p *Provider) decryptConfig(ciphertext string) (*sms.TwilioConfig, error) {
	raw, err := p.cipher.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	pc, err := sms.ParseProviderConfig("twilio", raw)
	if err != nil {
		return nil, err
	}
	return pc.Twilio, nil
}

type twilioSendResponse struct {
	SID string `json:"sid"`
}

func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return hmac.Equal(a, b)
}
