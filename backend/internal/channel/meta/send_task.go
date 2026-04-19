package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	appcrypto "backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/repo"
)

const (
	TypeMetaSend    = "channel:meta:send"
	metaMaxRetries  = 5
	metaSendTimeout = 10 * time.Second
)

// MetaSendPayload is the asynq task payload for async Meta outbound sends.
type MetaSendPayload struct {
	ChannelType string `json:"channelType"` // "instagram" or "facebook"
	ChannelID   int64  `json:"channelId"`
	MessageID   int64  `json:"messageId"`
	AccountID   int64  `json:"accountId"`
}

// MetaSendRetryDelay implements the spec backoff schedule: 1s, 5s, 30s, 2m, 10m.
func MetaSendRetryDelay(n int, _ error, _ *asynq.Task) time.Duration {
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

// NewMetaSendTask creates an asynq task for async Meta message delivery.
func NewMetaSendTask(payload *MetaSendPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal meta send payload: %w", err)
	}
	return asynq.NewTask(TypeMetaSend, data,
		asynq.MaxRetry(metaMaxRetries),
		asynq.Timeout(metaSendTimeout),
	), nil
}

// MetaSendProcessor processes channel:meta:send tasks.
type MetaSendProcessor struct {
	igRepo      *repo.ChannelInstagramRepo
	fbRepo      *repo.ChannelFacebookRepo
	messageRepo *repo.MessageRepo
	cipher      *appcrypto.Cipher
	appSecret   string
}

func NewMetaSendProcessor(
	igRepo *repo.ChannelInstagramRepo,
	fbRepo *repo.ChannelFacebookRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	appSecret string,
) *MetaSendProcessor {
	return &MetaSendProcessor{
		igRepo:      igRepo,
		fbRepo:      fbRepo,
		messageRepo: messageRepo,
		cipher:      cipher,
		appSecret:   appSecret,
	}
}

func (p *MetaSendProcessor) HandleMetaSend(ctx context.Context, t *asynq.Task) error {
	var payload MetaSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal meta send payload: %w", err)
	}

	msg, err := p.messageRepo.FindByID(ctx, payload.MessageID, payload.AccountID)
	if err != nil {
		return fmt.Errorf("meta send: find message %d: %w", payload.MessageID, err)
	}

	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}

	var sourceID string
	var sendErr error

	switch payload.ChannelType {
	case "instagram":
		sourceID, sendErr = p.sendInstagram(ctx, payload.ChannelID, payload.AccountID, msg.InboxID, content)
	case "facebook":
		sourceID, sendErr = p.sendFacebook(ctx, payload.ChannelID, payload.AccountID, msg.InboxID, content)
	default:
		return fmt.Errorf("%w: unknown channel type %s", asynq.SkipRetry, payload.ChannelType)
	}

	if sendErr != nil {
		if errors.Is(sendErr, ErrMetaAuthFailed) || errors.Is(sendErr, ErrMetaPermanent) {
			logger.Warn().Str("component", "meta.send_task").
				Str("channelType", payload.ChannelType).
				Int64("messageId", payload.MessageID).
				Err(sendErr).Msg("permanent send failure, dead-lettering")
			return asynq.SkipRetry
		}
		return fmt.Errorf("meta send: %w", sendErr)
	}

	if sourceID != "" {
		_ = p.messageRepo.UpdateSourceID(ctx, payload.MessageID, payload.AccountID, &sourceID)
	}
	return nil
}

func (p *MetaSendProcessor) sendInstagram(ctx context.Context, channelID, accountID, inboxID int64, content string) (string, error) {
	ch, err := p.igRepo.FindByID(ctx, channelID, accountID)
	if err != nil {
		return "", fmt.Errorf("find instagram channel: %w", err)
	}
	token, err := p.cipher.Decrypt(ch.AccessTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt instagram token: %w", err)
	}
	client := NewClient("https://graph.instagram.com")
	var resp struct {
		MessageID string `json:"message_id"`
	}
	path := fmt.Sprintf("/%s/messages?appsecret_proof=%s", ch.InstagramID, AppSecretProof(token, p.appSecret))
	body := map[string]any{
		"recipient": map[string]any{"id": fmt.Sprintf("%d", inboxID)},
		"message":   map[string]any{"text": content},
	}
	if err := client.Post(ctx, path, token, body, &resp); err != nil {
		return "", err
	}
	return resp.MessageID, nil
}

func (p *MetaSendProcessor) sendFacebook(ctx context.Context, channelID, accountID, inboxID int64, content string) (string, error) {
	ch, err := p.fbRepo.FindByID(ctx, channelID, accountID)
	if err != nil {
		return "", fmt.Errorf("find facebook channel: %w", err)
	}
	token, err := p.cipher.Decrypt(ch.PageAccessTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt facebook token: %w", err)
	}
	client := NewClient("https://graph.facebook.com")
	var resp struct {
		MessageID string `json:"message_id"`
	}
	path := fmt.Sprintf("/%s/messages?appsecret_proof=%s", ch.PageID, AppSecretProof(token, p.appSecret))
	body := map[string]any{
		"recipient": map[string]any{"id": fmt.Sprintf("%d", inboxID)},
		"message":   map[string]any{"text": content},
	}
	if err := client.Post(ctx, path, token, body, &resp); err != nil {
		return "", err
	}
	return resp.MessageID, nil
}
