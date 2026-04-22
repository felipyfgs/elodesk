package service

import (
	"context"
	"encoding/json"

	appchannel "backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type OutboundWebhookNotifier struct {
	outboundWebhookSvc *OutboundWebhookService
	channelApiRepo     *repo.ChannelAPIRepo
	inboxRepo          *repo.InboxRepo
	conversationRepo   *repo.ConversationRepo
}

func NewOutboundWebhookNotifier(
	outboundWebhookSvc *OutboundWebhookService,
	channelApiRepo *repo.ChannelAPIRepo,
	inboxRepo *repo.InboxRepo,
	conversationRepo *repo.ConversationRepo,
) *OutboundWebhookNotifier {
	return &OutboundWebhookNotifier{
		outboundWebhookSvc: outboundWebhookSvc,
		channelApiRepo:     channelApiRepo,
		inboxRepo:          inboxRepo,
		conversationRepo:   conversationRepo,
	}
}

func (n *OutboundWebhookNotifier) resolveChannelAPI(ctx context.Context, inboxID, accountID int64) (*model.ChannelAPI, *model.Inbox, error) {
	inbox, err := n.inboxRepo.FindByID(ctx, inboxID, accountID)
	if err != nil {
		return nil, nil, err
	}
	if inbox.ChannelType != string(appchannel.KindApi) {
		return nil, inbox, nil
	}
	ch, err := n.channelApiRepo.FindByID(ctx, inbox.ChannelID)
	if err != nil {
		return nil, inbox, err
	}
	if ch.WebhookURL == "" {
		return nil, inbox, nil
	}
	return ch, inbox, nil
}

func (n *OutboundWebhookNotifier) OnOutboundMessageCreated(ctx context.Context, accountID, inboxID int64, msg *model.Message) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		if err != nil {
			logger.Warn().Str("component", "outbound-webhook-notifier").
				Int64("inboxId", inboxID).Err(err).Msg("find channel api")
		}
		return
	}

	conv, err := n.conversationRepo.FindByID(ctx, msg.ConversationID, accountID)
	if err != nil {
		logger.Warn().Str("component", "outbound-webhook-notifier").
			Int64("conversationId", msg.ConversationID).Err(err).Msg("find conversation")
		return
	}

	if err := n.outboundWebhookSvc.DispatchMessageCreated(ctx, ch, inboxID, conv, msg); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("messageId", msg.ID).Err(err).Msg("dispatch webhook")
	}
}

func (n *OutboundWebhookNotifier) OnConversationCreated(ctx context.Context, accountID, inboxID int64, conv *model.Conversation) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		return
	}
	if err := n.outboundWebhookSvc.DispatchConversationCreated(ctx, ch, inboxID, conv); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("conversationId", conv.ID).Err(err).Msg("dispatch conversation_created webhook")
	}
}

func (n *OutboundWebhookNotifier) OnConversationUpdated(ctx context.Context, accountID, inboxID int64, conv *model.Conversation, attributes json.RawMessage) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		return
	}
	if err := n.outboundWebhookSvc.DispatchConversationUpdated(ctx, ch, inboxID, conv, attributes); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("conversationId", conv.ID).Err(err).Msg("dispatch conversation_updated webhook")
	}
}

func (n *OutboundWebhookNotifier) OnConversationStatusChanged(ctx context.Context, accountID, inboxID int64, conv *model.Conversation) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		return
	}
	if err := n.outboundWebhookSvc.DispatchConversationStatusChanged(ctx, ch, inboxID, conv); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("conversationId", conv.ID).Err(err).Msg("dispatch conversation_status_changed webhook")
	}
}

func (n *OutboundWebhookNotifier) OnMessageUpdated(ctx context.Context, accountID, inboxID int64, msg *model.Message) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		return
	}
	conv, err := n.conversationRepo.FindByID(ctx, msg.ConversationID, accountID)
	if err != nil {
		return
	}
	if err := n.outboundWebhookSvc.DispatchMessageUpdated(ctx, ch, inboxID, conv, msg); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("messageId", msg.ID).Err(err).Msg("dispatch message_updated webhook")
	}
}

func (n *OutboundWebhookNotifier) OnTypingEvent(ctx context.Context, accountID, inboxID int64, conv *model.Conversation, eventName string) {
	ch, _, err := n.resolveChannelAPI(ctx, inboxID, accountID)
	if err != nil || ch == nil {
		return
	}
	if err := n.outboundWebhookSvc.DispatchTypingEvent(ctx, ch, inboxID, conv, eventName); err != nil {
		logger.Error().Str("component", "outbound-webhook-notifier").
			Int64("conversationId", conv.ID).Err(err).Msg("dispatch typing event webhook")
	}
}
