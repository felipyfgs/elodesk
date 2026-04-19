package telegram

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	appcrypto "backend/internal/crypto"
	"backend/internal/repo"
)

type Channel struct {
	tgRepo           *repo.ChannelTelegramRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
	api              *APIClient
}

func NewChannel(
	tgRepo *repo.ChannelTelegramRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	redisClient redis.Cmdable,
	asynqClient *asynq.Client,
) *Channel {
	return &Channel{
		tgRepo:           tgRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            channel.NewDedupLock(redisClient),
		asynqClient:      asynqClient,
		api:              NewAPIClient(),
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindTelegram }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	identifier := req.PathParams["identifier"]
	ch, err := c.tgRepo.FindByWebhookIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("telegram handle inbound: find channel: %w", err)
	}

	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("telegram handle inbound: find inbox: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, inbox, ch.AccountID, c.dedup, c.asynqClient,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.tgRepo.FindByID(ctx, msg.ChannelID, 0)
	if err != nil {
		return "", fmt.Errorf("telegram send: find channel: %w", err)
	}

	botToken, err := c.cipher.Decrypt(ch.BotTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("telegram send: decrypt token: %w", err)
	}

	var contentAttrsJSON string
	if msg.ChannelID > 0 {
		dbMsg, findErr := c.messageRepo.FindByID(ctx, msg.ChannelID, 0)
		if findErr == nil && dbMsg.ContentAttrs != nil {
			contentAttrsJSON = *dbMsg.ContentAttrs
		}
	}

	return Send(ctx, ch, botToken, msg.To, msg.Content, msg.MediaURL, msg.MediaType, contentAttrsJSON)
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

var _ channel.Channel = (*Channel)(nil)
