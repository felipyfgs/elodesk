package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/webhook"
)

const (
	EventTypeMessageCreated            = "message_created"
	EventTypeMessageUpdated            = "message_updated"
	EventTypeConversationStatusChanged = "conversation_status_changed"
	EventTypeConversationUpdated       = "conversation_updated"
)

type OutboundWebhookService struct {
	asynqClient *asynq.Client
}

func NewOutboundWebhookService(asynqClient *asynq.Client) *OutboundWebhookService {
	return &OutboundWebhookService{asynqClient: asynqClient}
}

func (s *OutboundWebhookService) DispatchMessageCreated(ctx context.Context, ch *model.ChannelApi, inboxID int64, conv *model.Conversation, msg *model.Message) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeMessageCreated, conv, msg, nil)
}

func (s *OutboundWebhookService) DispatchMessageUpdated(ctx context.Context, ch *model.ChannelApi, inboxID int64, conv *model.Conversation, msg *model.Message) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeMessageUpdated, conv, msg, nil)
}

func (s *OutboundWebhookService) DispatchConversationStatusChanged(ctx context.Context, ch *model.ChannelApi, inboxID int64, conv *model.Conversation) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeConversationStatusChanged, conv, nil, nil)
}

func (s *OutboundWebhookService) DispatchConversationUpdated(ctx context.Context, ch *model.ChannelApi, inboxID int64, conv *model.Conversation, attributes json.RawMessage) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeConversationUpdated, conv, nil, attributes)
}

func (s *OutboundWebhookService) dispatch(ctx context.Context, ch *model.ChannelApi, inboxID int64, eventType string, conv *model.Conversation, msg *model.Message, convAttrs json.RawMessage) error {
	// DeliveryID is generated HERE (once per delivery) and stored in the task
	// payload so it survives retries. The processor never regenerates it.
	payload := &webhook.OutboundPayload{
		EventType:              eventType,
		AccountID:              ch.AccountID,
		InboxID:                inboxID,
		WebhookURL:             ch.WebhookURL,
		HmacCiphertext:         ch.HmacToken,
		DeliveryID:             uuid.NewString(),
		ConversationAttributes: convAttrs,
	}

	if conv != nil {
		data, err := json.Marshal(conv)
		if err != nil {
			return fmt.Errorf("marshal conversation: %w", err)
		}
		payload.Conversation = data
	}
	if msg != nil {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}
		payload.Message = data
	}

	task, err := webhook.NewOutboundTask(payload)
	if err != nil {
		return fmt.Errorf("create outbound task: %w", err)
	}

	info, err := s.asynqClient.EnqueueContext(ctx, task)
	if err != nil {
		logger.Error().Str("component", "outbound-webhook").Err(err).
			Str("eventType", eventType).
			Int64("accountId", ch.AccountID).
			Str("webhookUrl", ch.WebhookURL).
			Msg("enqueue failed")
		return fmt.Errorf("enqueue outbound webhook: %w", err)
	}

	logger.Info().Str("component", "outbound-webhook").
		Str("eventType", eventType).
		Int64("accountId", ch.AccountID).
		Str("webhookUrl", ch.WebhookURL).
		Str("deliveryId", payload.DeliveryID).
		Str("taskId", info.ID).
		Str("queue", info.Queue).
		Msg("enqueued")

	return nil
}
