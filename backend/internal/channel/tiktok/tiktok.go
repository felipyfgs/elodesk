package tiktok

import (
	"context"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	appcrypto "backend/internal/crypto"
	"backend/internal/repo"
)

type Channel struct {
	tiktokRepo       *repo.ChannelTiktokRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
	api              *APIClient
	tokens           *TokenService
	appSecret        string
}

func NewChannel(
	tiktokRepo *repo.ChannelTiktokRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	redisClient redis.Cmdable,
	asynqClient *asynq.Client,
	tokens *TokenService,
	appSecret string,
) *Channel {
	return &Channel{
		tiktokRepo:       tiktokRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            channel.NewDedupLock(redisClient),
		asynqClient:      asynqClient,
		api:              NewAPIClient(),
		tokens:           tokens,
		appSecret:        appSecret,
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindTiktok }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	businessID := req.PathParams["business_id"]
	if businessID == "" {
		businessID = req.PathParams["identifier"]
	}

	ch, err := c.tiktokRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("tiktok handle inbound: find channel: %w", err)
	}
	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("tiktok handle inbound: find inbox: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, ch, inbox, c.dedup,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.tiktokRepo.FindByIDNoScope(ctx, msg.ChannelID)
	if err != nil {
		return "", fmt.Errorf("tiktok send: find channel: %w", err)
	}

	accessToken, err := c.tokens.AccessToken(ctx, ch)
	if err != nil {
		return "", fmt.Errorf("tiktok send: access token: %w", err)
	}

	var conversationID string
	if msg.ChannelID > 0 {
		if dbMsg, findErr := c.messageRepo.FindByID(ctx, msg.ChannelID, 0); findErr == nil && dbMsg.ContentAttrs != nil {
			conversationID = ConversationIDFromAttrs(*dbMsg.ContentAttrs)
		}
	}
	if conversationID == "" {
		conversationID = msg.To
	}

	sourceID, err := SendText(ctx, c.api, accessToken, ch.BusinessID, conversationID, msg.Content, "")
	if errors.Is(err, ErrReauthRequired) {
		_ = c.tiktokRepo.SetRequiresReauth(ctx, ch.ID, true)
	}
	return sourceID, err
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

var _ channel.Channel = (*Channel)(nil)
