package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	TypeChannelSmsIngest = "channel:sms:ingest"
	TypeChannelSmsSend   = "channel:sms:send"
	smsMaxRetries        = 5
	smsTimeout           = 30 * time.Second
)

type SmsIngestPayload struct {
	ChannelID int64             `json:"channelId"`
	Provider  string            `json:"provider"`
	RawBody   string            `json:"rawBody"`
	Headers   map[string]string `json:"headers"`
}

type SmsSendPayload struct {
	ChannelID                int64    `json:"channelId"`
	AccountID                int64    `json:"accountId"`
	InboxID                  int64    `json:"inboxId"`
	ConversationID           int64    `json:"conversationId"`
	MessageID                int64    `json:"messageId"`
	To                       string   `json:"to"`
	Content                  string   `json:"content,omitempty"`
	MediaURL                 []string `json:"mediaUrl,omitempty"`
	Provider                 string   `json:"provider"`
	ProviderConfigCiphertext string   `json:"providerConfigCiphertext"`
	PhoneNumber              string   `json:"phoneNumber"`
	MessagingServiceSid      string   `json:"messagingServiceSid,omitempty"`
	StatusCallbackURL        string   `json:"statusCallbackUrl,omitempty"`
}

type SmsIngestProcessor struct {
	ingestSvc  *IngestService
	registry   *Registry
	httpClient *http.Client
}

func NewSmsIngestProcessor(ingestSvc *IngestService, registry *Registry) *SmsIngestProcessor {
	return &SmsIngestProcessor{
		ingestSvc:  ingestSvc,
		registry:   registry,
		httpClient: &http.Client{Timeout: smsTimeout},
	}
}

func NewSmsIngestTask(payload *SmsIngestPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal sms ingest payload: %w", err)
	}
	return asynq.NewTask(TypeChannelSmsIngest, data,
		asynq.MaxRetry(smsMaxRetries),
		asynq.Timeout(smsTimeout),
	), nil
}

func (p *SmsIngestProcessor) HandleSmsIngest(ctx context.Context, t *asynq.Task) error {
	var payload SmsIngestPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Error().Str("component", "sms-ingest").Err(err).Msg("unmarshal payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	return fmt.Errorf("sms-ingest: use HandleInboundHTTP instead")
}

func (p *SmsIngestProcessor) HandleInboundHTTP(ctx context.Context, channelID int64, provider string, rawBody []byte, r *http.Request) error {
	prov, err := p.registry.Get(provider)
	if err != nil {
		return fmt.Errorf("sms ingest: get provider: %w", err)
	}

	inbound, err := prov.ParseInbound(r)
	if err != nil {
		return fmt.Errorf("sms ingest: parse inbound: %w", err)
	}

	ch := &model.ChannelSMS{
		ID:       channelID,
		Provider: provider,
	}

	return p.ingestSvc.IngestInbound(ctx, ch, inbound)
}

type SmsSendProcessor struct {
	channelSMSRepo *repo.ChannelSMSRepo
	messageRepo    *repo.MessageRepo
	registry       *Registry
	httpClient     *http.Client
}

func NewSmsSendProcessor(channelSMSRepo *repo.ChannelSMSRepo, messageRepo *repo.MessageRepo, registry *Registry) *SmsSendProcessor {
	return &SmsSendProcessor{
		channelSMSRepo: channelSMSRepo,
		messageRepo:    messageRepo,
		registry:       registry,
		httpClient:     &http.Client{Timeout: smsTimeout},
	}
}

func NewSmsSendTask(payload *SmsSendPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal sms send payload: %w", err)
	}
	return asynq.NewTask(TypeChannelSmsSend, data,
		asynq.MaxRetry(smsMaxRetries),
		asynq.Timeout(smsTimeout),
	), nil
}

func (p *SmsSendProcessor) HandleSmsSend(ctx context.Context, t *asynq.Task) error {
	var payload SmsSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Error().Str("component", "sms-send").Err(err).Msg("unmarshal payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	ch, err := p.channelSMSRepo.FindByID(ctx, payload.ChannelID, payload.AccountID)
	if err != nil {
		logger.Error().Str("component", "sms-send").Err(err).Msg("find channel")
		return fmt.Errorf("find channel: %w", err)
	}

	prov, err := p.registry.Get(payload.Provider)
	if err != nil {
		return fmt.Errorf("sms send: get provider: %w", err)
	}

	out := &OutboundMessage{
		To:       payload.To,
		Content:  payload.Content,
		MediaURL: payload.MediaURL,
	}

	sourceID, err := prov.Send(ctx, ch, out, payload.StatusCallbackURL)
	if err != nil {
		if IsAuthError(err) {
			logger.Warn().Str("component", "sms-send").
				Int64("channelId", payload.ChannelID).
				Msg("auth error, dead-letter")
			return asynq.SkipRetry
		}
		if IsRetryableError(err) {
			logger.Warn().Str("component", "sms-send").
				Int64("messageId", payload.MessageID).
				Msg("retryable error, will retry")
			return fmt.Errorf("sms send failed: %w", err)
		}
		logger.Warn().Str("component", "sms-send").
			Int64("messageId", payload.MessageID).
			Msg("client error, dead-letter")
		return asynq.SkipRetry
	}

	if sourceID != "" {
		sid := sourceID
		if err := p.messageRepo.UpdateSourceID(ctx, payload.MessageID, payload.AccountID, &sid); err != nil {
			logger.Warn().Str("component", "sms-send").Err(err).Msg("update source_id")
		}
	}

	logger.Info().Str("component", "sms-send").
		Str("sourceId", sourceID).
		Int64("messageId", payload.MessageID).
		Msg("sent")
	return nil
}

func SmsRetryDelay(n int, _ error, _ *asynq.Task) time.Duration {
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
