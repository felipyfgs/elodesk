package webwidget

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/channel"
	"backend/internal/model"
	"backend/internal/repo"

	"github.com/redis/go-redis/v9"
)

var _ channel.Channel = (*Channel)(nil)

type Channel struct {
	widgetRepo       *repo.ChannelWebWidgetRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	pubsub           *PubsubService
}

func NewChannel(
	widgetRepo *repo.ChannelWebWidgetRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	redisClient *redis.Client,
) *Channel {
	return &Channel{
		widgetRepo:       widgetRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		pubsub:           NewPubsubService(redisClient),
	}
}

func (ch *Channel) Kind() channel.Kind {
	return channel.KindWebWidget
}

func (ch *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	return nil, channel.ErrUnsupported
}

func (ch *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	conversation, err := ch.conversationRepo.FindByID(ctx, msg.ChannelID, 0)
	if err != nil {
		return "", fmt.Errorf("failed to find conversation: %w", err)
	}

	widget, err := ch.widgetRepo.FindByInboxID(ctx, conversation.InboxID)
	if err != nil {
		return "", fmt.Errorf("failed to find widget channel: %w", err)
	}

	messageData := map[string]any{
		"id":             conversation.ID,
		"conversationId": conversation.ID,
		"content":        msg.Content,
		"senderType":     "Agent",
		"createdAt":      conversation.LastActivityAt,
	}

	if conversation.PubsubToken != nil && *conversation.PubsubToken != "" {
		if err := ch.pubsub.PublishMessageCreated(ctx, *conversation.PubsubToken, messageData); err != nil {
			return "", fmt.Errorf("failed to publish outbound message: %w", err)
		}
	}

	_ = widget
	return fmt.Sprintf("%d", conversation.ID), nil
}

func (ch *Channel) SyncTemplates(ctx context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

func GetWidgetConfig(m *model.ChannelWebWidget) *WidgetConfig {
	return &WidgetConfig{
		WebsiteURL:     m.WebsiteURL,
		WidgetColor:    m.WidgetColor,
		WelcomeTitle:   m.WelcomeTitle,
		WelcomeTagline: m.WelcomeTagline,
		ReplyTime:      m.ReplyTime,
		FeatureFlags:   json.RawMessage(m.FeatureFlags),
	}
}
