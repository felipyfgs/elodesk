package instagram

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	"backend/internal/channel/reauth"
	appcrypto "backend/internal/crypto"
	"backend/internal/repo"
)

// Channel implements channel.Channel for Instagram DMs.
type Channel struct {
	igRepo           *repo.ChannelInstagramRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	appSecret        string
	verifyToken      string
	dedup            *channel.DedupLock
	reauthTracker    *reauth.Tracker
	asynqClient      *asynq.Client
}

func NewChannel(
	igRepo *repo.ChannelInstagramRepo,
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
		igRepo:           igRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		appSecret:        appSecret,
		verifyToken:      verifyToken,
		dedup:            channel.NewDedupLock(redisClient),
		reauthTracker:    reauth.NewTracker(redisClient),
		asynqClient:      asynqClient,
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindInstagram }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	instagramID := req.PathParams["identifier"]
	ch, err := c.igRepo.FindByInstagramID(ctx, instagramID)
	if err != nil {
		return nil, fmt.Errorf("instagram handle inbound: find channel: %w", err)
	}

	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("instagram handle inbound: find inbox: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, inbox, ch.AccountID, c.dedup, c.asynqClient,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.igRepo.FindByID(ctx, msg.ChannelID, 0)
	if err != nil {
		// try without account scope (channel registry doesn't have accountID)
		return "", fmt.Errorf("instagram send: find channel: %w", err)
	}

	accessToken, decErr := c.cipher.Decrypt(ch.AccessTokenCiphertext)
	if decErr != nil {
		return "", fmt.Errorf("instagram send: decrypt token: %w", decErr)
	}

	accessToken, err = RefreshIfNeeded(ctx, ch, accessToken, c.igRepo, c.cipher, c.reauthTracker)
	if err != nil {
		return "", err
	}

	return Send(ctx, ch, accessToken, c.appSecret, msg.To, msg.Content, msg.MediaURL, msg.MediaType)
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}
