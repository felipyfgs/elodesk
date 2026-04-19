package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/webhook"
)

const (
	EventTypeMessageCreated           = "message_created"
	EventTypeMessageUpdated           = "message_updated"
	EventTypeConversationStatusChanged = "conversation_status_changed"
	EventTypeConversationUpdated      = "conversation_updated"
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
	payload := &webhook.OutboundPayload{
		EventType:              eventType,
		AccountID:              ch.AccountID,
		InboxID:                inboxID,
		WebhookURL:             ch.WebhookURL,
		HmacToken:              ch.HmacToken,
		ConversationAttributes: convAttrs,
	}

	if conv != nil {
		convData, err := json.Marshal(conv)
		if err != nil {
			return fmt.Errorf("failed to marshal conversation: %w", err)
		}
		payload.Conversation = convData
	}

	if msg != nil {
		msgData, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		payload.Message = msgData
	}

	task, err := webhook.NewOutboundTask(payload)
	if err != nil {
		return fmt.Errorf("failed to create outbound webhook task: %w", err)
	}

	info, err := s.asynqClient.EnqueueContext(ctx, task)
	if err != nil {
		logger.Error().
			Str("component", "outbound-webhook").
			Err(err).
			Str("eventType", eventType).
			Int64("accountId", ch.AccountID).
			Str("webhookUrl", ch.WebhookURL).
			Msg("Failed to enqueue outbound webhook")
		return fmt.Errorf("failed to enqueue outbound webhook: %w", err)
	}

	logger.Info().
		Str("component", "outbound-webhook").
		Str("eventType", eventType).
		Int64("accountId", ch.AccountID).
		Str("webhookUrl", ch.WebhookURL).
		Str("taskId", info.ID).
		Str("queue", info.Queue).
		Msg("Outbound webhook enqueued")

	return nil
}
