package webhook

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
	"strconv"
	"time"

	"github.com/hibiken/asynq"

	"backend/internal/crypto"
	"backend/internal/logger"
)

const (
	TypeOutboundWebhook = "webhook:outbound"

	MaxRetries      = 5
	deliveryTimeout = 10 * time.Second
)

// OutboundRetryDelay implements the spec retry schedule: 1s, 5s, 30s, 2m, 10m.
func OutboundRetryDelay(n int, _ error, _ *asynq.Task) time.Duration {
	delays := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
	}
	if n >= 1 && n-1 < len(delays) {
		return delays[n-1]
	}
	return 10 * time.Minute
}

// OutboundPayload is the task payload enqueued by OutboundWebhookService.
// Secret holds the channel secret encrypted with BACKEND_KEK (AES-256-GCM).
// HmacCiphertext carries the per-channel HMAC key encrypted with BACKEND_KEK;
// the processor decrypts both right before signing so plaintext never lives in
// Redis. DeliveryID is generated at enqueue time and stays stable across all
// retries of the same delivery.
type OutboundPayload struct {
	EventType              string          `json:"event"`
	AccountID              int64           `json:"accountId"`
	InboxID                int64           `json:"inboxId"`
	WebhookURL             string          `json:"webhookUrl"`
	Secret                 string          `json:"secret"`
	HmacCiphertext         string          `json:"hmacCiphertext"`
	DeliveryID             string          `json:"deliveryId"`
	Conversation           json.RawMessage `json:"conversation,omitempty"`
	Message                json.RawMessage `json:"message,omitempty"`
	ConversationAttributes json.RawMessage `json:"conversation_attributes,omitempty"`
}

type OutboundProcessor struct {
	httpClient *http.Client
	cipher     *crypto.Cipher
}

func NewOutboundProcessor(cipher *crypto.Cipher) *OutboundProcessor {
	return &OutboundProcessor{
		httpClient: &http.Client{Timeout: deliveryTimeout},
		cipher:     cipher,
	}
}

// NewOutboundTask marshals the payload for asynq. Callers MUST populate
// payload.DeliveryID before calling this.
func NewOutboundTask(payload *OutboundPayload) (*asynq.Task, error) {
	if payload.DeliveryID == "" {
		return nil, fmt.Errorf("outbound webhook: DeliveryID is required")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal outbound webhook payload: %w", err)
	}
	return asynq.NewTask(TypeOutboundWebhook, data,
		asynq.MaxRetry(MaxRetries),
		asynq.Timeout(deliveryTimeout),
	), nil
}

// publicBody is what we ship to the provider (excludes infra fields like
// webhookUrl and hmacCiphertext).
type publicBody struct {
	EventType              string          `json:"event"`
	AccountID              int64           `json:"accountId"`
	InboxID                int64           `json:"inboxId"`
	DeliveryID             string          `json:"deliveryId"`
	Conversation           json.RawMessage `json:"conversation,omitempty"`
	Message                json.RawMessage `json:"message,omitempty"`
	ConversationAttributes json.RawMessage `json:"conversation_attributes,omitempty"`
}

func (p *OutboundProcessor) HandleOutboundWebhook(ctx context.Context, t *asynq.Task) error {
	var payload OutboundPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Error().Str("component", "outbound-webhook").Err(err).Msg("unmarshal payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	if payload.WebhookURL == "" {
		logger.Warn().Str("component", "outbound-webhook").
			Str("eventType", payload.EventType).
			Int64("accountId", payload.AccountID).
			Msg("skip: no webhook URL")
		return nil
	}

	body, err := json.Marshal(publicBody{
		EventType:              payload.EventType,
		AccountID:              payload.AccountID,
		InboxID:                payload.InboxID,
		DeliveryID:             payload.DeliveryID,
		Conversation:           payload.Conversation,
		Message:                payload.Message,
		ConversationAttributes: payload.ConversationAttributes,
	})
	if err != nil {
		return fmt.Errorf("marshal public body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, payload.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if payload.Secret != "" {
		secret, err := p.cipher.Decrypt(payload.Secret)
		if err != nil {
			logger.Error().Str("component", "outbound-webhook").
				Str("eventType", payload.EventType).
				Int64("accountId", payload.AccountID).
				Err(err).Msg("failed to decrypt webhook secret, dead-letter")
			return fmt.Errorf("decrypt webhook secret: %w", asynq.SkipRetry)
		}
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		signedBody := ts + "." + string(body)
		sig := computeHmacSha256([]byte(signedBody), secret)
		upstreamSig := "sha256=" + sig

		req.Header.Set("X-Chatwoot-Delivery", payload.DeliveryID)
		req.Header.Set("X-Chatwoot-Timestamp", ts)
		req.Header.Set("X-Chatwoot-Signature", upstreamSig)
		req.Header.Set("X-Elodesk-Signature", upstreamSig)
	} else {
		logger.Error().Str("component", "outbound-webhook").
			Str("eventType", payload.EventType).
			Int64("accountId", payload.AccountID).
			Msg("secret is empty, dead-letter")
		return fmt.Errorf("outbound webhook: channel.secret is empty: %w", asynq.SkipRetry)
	}

	if payload.HmacCiphertext != "" {
		key, err := p.cipher.Decrypt(payload.HmacCiphertext)
		if err != nil {
			logger.Error().Str("component", "outbound-webhook").Err(err).Msg("decrypt hmac key")
			return fmt.Errorf("decrypt hmac key: %w", err)
		}
		req.Header.Set("X-Delivery-Id", payload.DeliveryID)
		req.Header.Set("X-Chatwoot-Hmac-Sha256", computeHmacSha256(body, key))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		logger.Error().Str("component", "outbound-webhook").
			Str("eventType", payload.EventType).
			Str("webhookUrl", payload.WebhookURL).
			Str("deliveryId", payload.DeliveryID).
			Int("retryCount", getRetryCount(ctx)).
			Err(err).Msg("delivery failed")
		return fmt.Errorf("delivery failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode >= 500:
		logger.Warn().Str("component", "outbound-webhook").
			Int("statusCode", resp.StatusCode).
			Str("deliveryId", payload.DeliveryID).
			Int("retryCount", getRetryCount(ctx)).
			Msg("server error, will retry")
		return fmt.Errorf("webhook server error: status %d", resp.StatusCode)

	case resp.StatusCode >= 400:
		logger.Warn().Str("component", "outbound-webhook").
			Int("statusCode", resp.StatusCode).
			Str("deliveryId", payload.DeliveryID).
			Str("responseBody", string(respBody)).
			Msg("client error, dead-letter")
		return asynq.SkipRetry

	default:
		logger.Info().Str("component", "outbound-webhook").
			Int("statusCode", resp.StatusCode).
			Str("deliveryId", payload.DeliveryID).
			Int64("accountId", payload.AccountID).
			Msg("delivered")
		return nil
	}
}

func computeHmacSha256(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func getRetryCount(ctx context.Context) int {
	n, _ := asynq.GetRetryCount(ctx)
	return n
}
