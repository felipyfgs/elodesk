package twitter

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	"backend/internal/channel/reauth"
	appcrypto "backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/repo"
)

// Channel implements channel.Channel for Twitter/X DMs.
type Channel struct {
	twitterRepo      *repo.ChannelTwitterRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	api              *APIClient
	tracker          *reauth.Tracker
	consumerSecret   string
}

func NewChannel(
	twitterRepo *repo.ChannelTwitterRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	redisClient redis.Cmdable,
	consumerKey, consumerSecret string,
) *Channel {
	return &Channel{
		twitterRepo:      twitterRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            channel.NewDedupLock(redisClient),
		api:              NewAPIClient(consumerKey, consumerSecret),
		tracker:          reauth.NewTracker(redisClient),
		consumerSecret:   consumerSecret,
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindTwitter }

func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	profileID := req.PathParams["profile_id"]
	if profileID == "" {
		profileID = req.PathParams["identifier"]
	}

	ch, err := c.twitterRepo.FindByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("twitter handle inbound: find channel: %w", err)
	}

	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("twitter handle inbound: find inbox: %w", err)
	}

	if err := ProcessWebhook(ctx, req.Body, ch, inbox, c.dedup,
		c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := c.twitterRepo.FindByIDNoScope(ctx, msg.ChannelID)
	if err != nil {
		return "", fmt.Errorf("twitter send: find channel: %w", err)
	}
	accessToken, err := c.cipher.Decrypt(ch.TwitterAccessTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("twitter send: decrypt token: %w", err)
	}
	accessSecret, err := c.cipher.Decrypt(ch.TwitterAccessTokenSecretCiphertext)
	if err != nil {
		return "", fmt.Errorf("twitter send: decrypt secret: %w", err)
	}

	sourceID, sendErr := Send(ctx, c.api, accessToken, accessSecret, SendOptions{
		ParticipantID: msg.To,
		Content:       msg.Content,
	})
	if sendErr != nil {
		if errors.Is(sendErr, ErrReauthRequired) {
			c.handleReauth(ctx, ch.ID)
		}
		return "", sendErr
	}
	return sourceID, nil
}

func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

// handleReauth records a reauth signal and, when the per-kind threshold is
// reached, persists requires_reauth=true. Logged errors are intentionally
// swallowed — reauth is a best-effort signal, not a critical write.
func (c *Channel) handleReauth(ctx context.Context, channelID int64) {
	key := fmt.Sprintf("twitter:%d", channelID)
	prompt, err := c.tracker.RecordErrorForKind(ctx, channel.KindTwitter, key)
	if err != nil {
		logger.Warn().Str("component", "channel.twitter").Err(err).Msg("reauth tracker error")
		return
	}
	if !prompt {
		return
	}
	if err := c.twitterRepo.SetRequiresReauth(ctx, channelID, true); err != nil {
		logger.Warn().Str("component", "channel.twitter").Err(err).
			Int64("channelId", channelID).Msg("set requires_reauth failed")
	}
}

var _ channel.Channel = (*Channel)(nil)
