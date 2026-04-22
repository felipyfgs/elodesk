package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"backend/internal/channel"
	"backend/internal/channel/reauth"
	appcrypto "backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// Channel implements channel.Channel for Twilio (sms + whatsapp mediums).
type Channel struct {
	channelRepo      *repo.ChannelTwilioRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	client           *Client
	tracker          *reauth.Tracker
}

func NewChannel(
	channelRepo *repo.ChannelTwilioRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	redisClient redis.Cmdable,
	client *Client,
) *Channel {
	return &Channel{
		channelRepo:      channelRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            channel.NewDedupLock(redisClient),
		client:           client,
		tracker:          reauth.NewTracker(redisClient),
	}
}

func (c *Channel) Kind() channel.Kind { return channel.KindTwilio }

// HandleInbound is called when the webhook handler has already verified the
// X-Twilio-Signature and JSON-encoded the form values as the request body.
func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	identifier := req.PathParams["identifier"]
	if identifier == "" {
		return nil, fmt.Errorf("twilio handle inbound: missing identifier")
	}

	ch, err := c.channelRepo.FindByWebhookIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("twilio handle inbound: find channel: %w", err)
	}
	inbox, err := c.inboxRepo.FindByChannelID(ctx, ch.ID)
	if err != nil {
		return nil, fmt.Errorf("twilio handle inbound: find inbox: %w", err)
	}

	vals := map[string]string{}
	if len(req.Body) > 0 {
		_ = json.Unmarshal(req.Body, &vals)
	}
	p := &InboundParams{
		MessageSid: vals["MessageSid"],
		From:       vals["From"],
		To:         vals["To"],
		Body:       vals["Body"],
	}
	ingester := NewIngester(c.channelRepo, c.inboxRepo, c.contactRepo, c.contactInboxRepo, c.conversationRepo, c.messageRepo, c.dedup)
	if err := ingester.Ingest(ctx, ch, inbox, p); err != nil {
		return nil, err
	}
	return &channel.InboundResult{}, nil
}

// SendOutbound resolves the channel record, decrypts the auth token and
// dispatches to Twilio. On 401/403 it nudges the reauth tracker; on 429 it
// returns the error so the caller can back off.
func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	if msg == nil || msg.ChannelID == 0 {
		return "", fmt.Errorf("twilio send: missing channel id")
	}
	ch, err := c.channelRepo.FindByIDNoScope(ctx, msg.ChannelID)
	if err != nil {
		return "", fmt.Errorf("twilio send: find channel: %w", err)
	}

	authToken, err := c.cipher.Decrypt(ch.AuthTokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("twilio send: decrypt auth token: %w", err)
	}

	var mediaURLs []string
	if msg.MediaURL != "" {
		mediaURLs = []string{msg.MediaURL}
	}

	sid, err := Send(ctx, c.client, OutboundInput{
		Channel:    ch,
		AuthToken:  authToken,
		To:         msg.To,
		Body:       msg.Content,
		MediaURLs:  mediaURLs,
		ContentSID: msg.TemplateName,
	})
	if err != nil {
		if IsAuthError(err) {
			c.markReauthOnThreshold(ctx, ch)
		}
		return "", err
	}
	return sid, nil
}

// SyncTemplates on the Channel interface is a no-op without channel context.
// Callers with a resolved channel use SyncTemplatesForChannel directly.
func (c *Channel) SyncTemplates(_ context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

// SyncTemplatesForChannel pages over the Twilio Content API for a single
// channel and persists the result. WhatsApp-only.
func (c *Channel) SyncTemplatesForChannel(ctx context.Context, ch *model.ChannelTwilio) ([]ContentTemplate, error) {
	if ch.Medium != model.TwilioMediumWhatsApp {
		return nil, channel.ErrUnsupported
	}
	authToken, err := c.cipher.Decrypt(ch.AuthTokenCiphertext)
	if err != nil {
		return nil, fmt.Errorf("twilio sync templates: decrypt auth token: %w", err)
	}
	apiKey := ""
	if ch.APIKeySID != nil {
		apiKey = *ch.APIKeySID
	}
	templates, err := c.client.ListContentTemplates(ctx, ch.AccountSID, apiKey, authToken)
	if err != nil {
		return nil, fmt.Errorf("twilio sync templates: list content: %w", err)
	}
	data, err := json.Marshal(templates)
	if err != nil {
		return nil, fmt.Errorf("twilio sync templates: marshal: %w", err)
	}
	if err := c.channelRepo.UpdateContentTemplates(ctx, ch.ID, string(data), nowFn()); err != nil {
		return nil, fmt.Errorf("twilio sync templates: persist: %w", err)
	}
	return templates, nil
}

func (c *Channel) markReauthOnThreshold(ctx context.Context, ch *model.ChannelTwilio) {
	key := fmt.Sprintf("channel:twilio:%d", ch.ID)
	prompt, err := c.tracker.RecordErrorForKind(ctx, channel.KindTwilio, key)
	if err != nil {
		logger.Warn().Str("component", "channel.twilio").Err(err).Msg("reauth tracker record error")
		return
	}
	if prompt {
		if err := c.channelRepo.SetRequiresReauth(ctx, ch.ID, true); err != nil {
			logger.Error().Str("component", "channel.twilio").Err(err).Msg("mark requires_reauth")
		}
	}
}

// nowFn is a package-level clock override hook so tests can freeze time.
var nowFn = func() time.Time { return time.Now() }

var _ channel.Channel = (*Channel)(nil)
