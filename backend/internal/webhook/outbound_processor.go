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
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"backend/internal/logger"
)

const (
	TypeOutboundWebhook = "webhook:outbound"

	MaxRetries     = 5
	deliveryTimeout = 10 * time.Second
)

func OutboundRetryDelay(n int, e error, t *asynq.Task) time.Duration {
	delays := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
	}
	if n-1 < len(delays) {
		return delays[n-1]
	}
	return 10 * time.Minute
}

type OutboundPayload struct {
	EventType             string          `json:"event"`
	AccountID             int64           `json:"accountId"`
	InboxID               int64           `json:"inboxId"`
	WebhookURL            string          `json:"-"`
	HmacToken             string          `json:"-"`
	DeliveryID            string          `json:"deliveryId"`
	Conversation          json.RawMessage `json:"conversation,omitempty"`
	Message               json.RawMessage `json:"message,omitempty"`
	ConversationAttributes json.RawMessage `json:"conversation_attributes,omitempty"`
}

type OutboundProcessor struct {
	httpClient *http.Client
}

func NewOutboundProcessor() *OutboundProcessor {
	return &OutboundProcessor{
		httpClient: &http.Client{
			Timeout: deliveryTimeout,
		},
	}
}

func NewOutboundTask(payload *OutboundPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbound webhook payload: %w", err)
	}
	return asynq.NewTask(TypeOutboundWebhook, data,
		asynq.MaxRetry(MaxRetries),
		asynq.Timeout(deliveryTimeout),
	), nil
}

func (p *OutboundProcessor) HandleOutboundWebhook(ctx context.Context, t *asynq.Task) error {
	var payload OutboundPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Error().
			Str("component", "outbound-webhook").
			Err(err).
			Msg("Failed to unmarshal outbound webhook payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	if payload.WebhookURL == "" {
		logger.Warn().
			Str("component", "outbound-webhook").
			Str("eventType", payload.EventType).
			Int64("accountId", payload.AccountID).
			Msg("Skipping webhook: no URL configured")
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error().
			Str("component", "outbound-webhook").
			Err(err).
			Msg("Failed to marshal webhook body")
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, payload.WebhookURL, bytes.NewReader(body))
	if err != nil {
		logger.Error().
			Str("component", "outbound-webhook").
			Err(err).
			Str("webhookUrl", payload.WebhookURL).
			Msg("Failed to create webhook request")
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if payload.HmacToken != "" {
		signature := computeHmacSha256(body, payload.HmacToken)
		req.Header.Set("X-Chatwoot-Hmac-Sha256", signature)
	}

	if payload.DeliveryID == "" {
		payload.DeliveryID = uuid.New().String()
	}
	req.Header.Set("X-Delivery-Id", payload.DeliveryID)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		logger.Error().
			Str("component", "outbound-webhook").
			Err(err).
			Str("eventType", payload.EventType).
			Str("webhookUrl", payload.WebhookURL).
			Str("deliveryId", payload.DeliveryID).
			Int("retryCount", getRetryCount(ctx)).
			Msg("Webhook delivery failed")
		return fmt.Errorf("webhook delivery failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 500 {
		logger.Warn().
			Str("component", "outbound-webhook").
			Int("statusCode", resp.StatusCode).
			Str("eventType", payload.EventType).
			Str("webhookUrl", payload.WebhookURL).
			Str("deliveryId", payload.DeliveryID).
			Int("retryCount", getRetryCount(ctx)).
			Msg("Webhook server error, will retry")
		return fmt.Errorf("webhook server error: status %d", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		logger.Warn().
			Str("component", "outbound-webhook").
			Int("statusCode", resp.StatusCode).
			Str("eventType", payload.EventType).
			Str("webhookUrl", payload.WebhookURL).
			Str("deliveryId", payload.DeliveryID).
			Str("responseBody", string(respBody)).
			Msg("Webhook client error, not retrying")
		return nil
	}

	logger.Info().
		Str("component", "outbound-webhook").
		Int("statusCode", resp.StatusCode).
		Str("eventType", payload.EventType).
		Str("webhookUrl", payload.WebhookURL).
		Str("deliveryId", payload.DeliveryID).
		Int64("accountId", payload.AccountID).
		Msg("Webhook delivered successfully")

	return nil
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
