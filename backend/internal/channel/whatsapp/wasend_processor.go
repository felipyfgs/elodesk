package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"backend/internal/crypto"
	"backend/internal/logger"
)

const (
	TypeChannelWaSend = "channel:wa:send"
	waSendTimeout     = 30 * time.Second
)

type WaSendProcessor struct {
	httpClient *http.Client
	cipher     *crypto.Cipher
}

func NewWaSendProcessor(cipher *crypto.Cipher) *WaSendProcessor {
	return &WaSendProcessor{
		httpClient: &http.Client{Timeout: waSendTimeout},
		cipher:     cipher,
	}
}

func NewWaSendTask(payload *WaSendPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal wa send payload: %w", err)
	}
	return asynq.NewTask(TypeChannelWaSend, data,
		asynq.MaxRetry(WaMaxRetries),
		asynq.Timeout(waSendTimeout),
	), nil
}

func (p *WaSendProcessor) HandleWaSend(ctx context.Context, t *asynq.Task) error {
	var payload WaSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Error().Str("component", "wa-send").Err(err).Msg("unmarshal payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	apiKey, err := p.cipher.Decrypt(payload.ApiKeyCiphertext)
	if err != nil {
		logger.Error().Str("component", "wa-send").Err(err).Msg("decrypt api key")
		return fmt.Errorf("decrypt api key: %w", err)
	}

	provider, err := ProviderForType(payload.Provider, p.httpClient)
	if err != nil {
		return err
	}

	sendCtx := ctx
	if payload.PhoneNumberID != "" {
		sendCtx = WithPhoneNumberID(ctx, payload.PhoneNumberID)
	}

	sourceID, err := provider.Send(sendCtx, apiKey, payload.To, payload.Content, payload.MediaURL, payload.MediaType,
		payload.TemplateName, payload.TemplateLang, payload.TemplateComponents)
	if err != nil {
		perr, ok := err.(*ProviderError)
		if ok && perr.StatusCode >= 400 && perr.StatusCode < 500 {
			logger.Warn().Str("component", "wa-send").
				Int("statusCode", perr.StatusCode).
				Int64("messageId", payload.MessageID).
				Msg("client error, dead-letter")
			return asynq.SkipRetry
		}
		logger.Warn().Str("component", "wa-send").
			Int64("messageId", payload.MessageID).
			Msg("send failed, will retry")
		return fmt.Errorf("wa send failed: %w", err)
	}

	logger.Info().Str("component", "wa-send").
		Str("sourceId", sourceID).
		Int64("messageId", payload.MessageID).
		Msg("sent")
	return nil
}

func WaRetryDelay(n int, _ error, _ *asynq.Task) time.Duration {
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
