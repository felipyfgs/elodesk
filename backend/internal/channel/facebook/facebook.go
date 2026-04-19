package facebook

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	appcrypto "backend/internal/crypto"
	"backend/internal/repo"
)

// Channel implements channel.Channel for Facebook Messenger (Page).
type Channel struct {
	fbRepo           *repo.ChannelFacebookRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	appSecret        string
	verifyToken      string
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
}

func NewChannel(
	fbRepo *repo.ChannelFacebookRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	redisClient redis.Cmdable,
	asynqClient *asynq.Client,
	appSecret, verifyToken string,
) *Channel {
	return &Channel{
		fbRepo:           fbRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		appSecret:        appSecret,
		verifyToken:      verifyToken,
		dedup:            channel.NewDedupLock(redisClient),
		asynqClient:      asynqClient,
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindFacebookPage }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	pageID := req.PathParams["identifier"]
	ch, err := c.fbRepo.FindByPageID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("facebook handle inbound: find channel: %w", err)
	}

	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("facebook handle inbound: find inbox: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, inbox, ch.AccountID, c.dedup, c.asynqClient,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.fbRepo.FindByPageID(ctx, msg.To)
	if err != nil {
		// fallback: treat msg.To as PSID and ChannelID as the channel record id
		ch2, err2 := c.fbRepo.FindByID(ctx, msg.ChannelID, 0)
		if err2 != nil {
			return "", fmt.Errorf("facebook send: find channel: %w", err)
		}
		ch = ch2
	}

	accessToken, err := c.cipher.Decrypt(ch.PageAccessTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("facebook send: decrypt token: %w", err)
	}

	return Send(ctx, ch, accessToken, c.appSecret, SendRequest{
		To:        msg.To,
		Content:   msg.Content,
		MediaURL:  msg.MediaURL,
		MediaType: msg.MediaType,
	})
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}
