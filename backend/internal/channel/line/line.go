package line

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
	lineRepo         *repo.ChannelLineRepo
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
	lineRepo *repo.ChannelLineRepo,
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
		lineRepo:         lineRepo,
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

func (c *Channel) Kind() channel.Kind { return channel.KindLine }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	lineChannelID := req.PathParams["line_channel_id"]
	if lineChannelID == "" {
		lineChannelID = req.PathParams["identifier"]
	}

	ch, err := c.lineRepo.FindByLineChannelID(ctx, lineChannelID)
	if err != nil {
		return nil, fmt.Errorf("line handle inbound: find channel: %w", err)
	}

	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("line handle inbound: find inbox: %w", err)
	}

	token, err := c.cipher.Decrypt(ch.LineChannelTokenCiphertext)
	if err != nil {
		return nil, fmt.Errorf("line handle inbound: decrypt token: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, ch, inbox, c.dedup, c.api, token,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.lineRepo.FindByIDNoScope(ctx, msg.ChannelID)
	if err != nil {
		return "", fmt.Errorf("line send: find channel: %w", err)
	}
	token, err := c.cipher.Decrypt(ch.LineChannelTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("line send: decrypt token: %w", err)
	}

	var contentAttrsJSON string
	if msg.ChannelID > 0 {
		if dbMsg, findErr := c.messageRepo.FindByID(ctx, msg.ChannelID, 0); findErr == nil && dbMsg.ContentAttrs != nil {
			contentAttrsJSON = *dbMsg.ContentAttrs
		}
	}

	return Send(ctx, c.api, token, SendOptions{
		To:           msg.To,
		Content:      msg.Content,
		MediaURL:     msg.MediaURL,
		MediaType:    msg.MediaType,
		ContentAttrs: contentAttrsJSON,
	})
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

var _ channel.Channel = (*Channel)(nil)
