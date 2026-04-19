package sms

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	appchannel "backend/internal/channel"
	"backend/internal/channel/reauth"
	"backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

const smsSendTimeout = 10 * time.Second

type Channel struct {
	channelSMSRepo *repo.ChannelSMSRepo
	inboxRepo      *repo.InboxRepo
	messageRepo    *repo.MessageRepo
	registry       *Registry
	cipher         *crypto.Cipher
	dedup          *appchannel.DedupLock
	reauth         *reauth.Tracker
	asynqClient    *asynq.Client
	media          *MediaHandler
	realtimeSvc    *service.RealtimeService
	httpClient     *http.Client
}

func NewChannel(
	channelSMSRepo *repo.ChannelSMSRepo,
	inboxRepo *repo.InboxRepo,
	messageRepo *repo.MessageRepo,
	registry *Registry,
	cipher *crypto.Cipher,
	dedup *appchannel.DedupLock,
	reauthTracker *reauth.Tracker,
	asynqClient *asynq.Client,
	media *MediaHandler,
	realtimeSvc *service.RealtimeService,
	httpClient *http.Client,
) *Channel {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: smsSendTimeout}
	}
	return &Channel{
		channelSMSRepo: channelSMSRepo,
		inboxRepo:      inboxRepo,
		messageRepo:    messageRepo,
		registry:       registry,
		cipher:         cipher,
		dedup:          dedup,
		reauth:         reauthTracker,
		asynqClient:    asynqClient,
		media:          media,
		realtimeSvc:    realtimeSvc,
		httpClient:     httpClient,
	}
}

func (c *Channel) Kind() appchannel.Kind {
	return appchannel.KindSms
}

func (c *Channel) HandleInbound(ctx context.Context, req *appchannel.InboundRequest) (*appchannel.InboundResult, error) {
	identifier := req.PathParams["identifier"]
	if identifier == "" {
		return nil, fmt.Errorf("sms: identifier required")
	}

	ch, err := c.channelSMSRepo.FindByWebhookIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("sms: find channel by identifier: %w", err)
	}

	providerName := req.PathParams["provider"]
	if providerName == "" {
		providerName = ch.Provider
	}
	if providerName != ch.Provider {
		return nil, fmt.Errorf("sms: provider mismatch: path=%s channel=%s", providerName, ch.Provider)
	}

	prov, err := c.registry.Get(ch.Provider)
	if err != nil {
		return nil, err
	}

	httpReq := buildHTTPRequest(req)
	if err := prov.VerifyWebhook(httpReq, ch); err != nil {
		return nil, fmt.Errorf("sms: webhook verification failed: %w", err)
	}

	inbound, err := prov.ParseInbound(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sms: parse inbound: %w", err)
	}

	return &appchannel.InboundResult{
		Messages: []appchannel.InboundMessage{
			{
				SourceID: inbound.SourceID,
				From:     inbound.From,
				To:       inbound.To,
				Content:  inbound.Content,
			},
		},
	}, nil
}

func (c *Channel) SendOutbound(ctx context.Context, msg *appchannel.OutboundMessage) (string, error) {
	ch, err := c.channelSMSRepo.FindByID(ctx, msg.ChannelID, 0)
	if err != nil {
		return "", fmt.Errorf("sms: find channel for outbound: %w", err)
	}

	prov, err := c.registry.Get(ch.Provider)
	if err != nil {
		return "", err
	}

	out := &OutboundMessage{
		To:       msg.To,
		Content:  msg.Content,
		MediaURL: []string{msg.MediaURL},
	}

	sourceID, err := prov.Send(ctx, ch, out, "")
	if err != nil {
		if IsAuthError(err) {
			prompt, trackerErr := c.reauth.RecordError(ctx, fmt.Sprintf("sms:%d", ch.ID))
			if trackerErr != nil {
				logger.Error().Str("component", "channel.sms").Err(trackerErr).Msg("reauth tracker error")
			}
			if prompt {
				_ = c.channelSMSRepo.SetRequiresReauth(ctx, ch.ID, true)
				c.realtimeSvc.BroadcastAccountEvent(ch.AccountID, "channel.reauth_required", map[string]interface{}{
					"channelId":   ch.ID,
					"channelType": "sms",
				})
			}
		}
		return "", err
	}

	_ = c.reauth.Reset(ctx, fmt.Sprintf("sms:%d", ch.ID))
	return sourceID, nil
}

func (c *Channel) SyncTemplates(ctx context.Context) ([]appchannel.Template, error) {
	return nil, appchannel.ErrUnsupported
}

func (c *Channel) GetProvider(ch *model.ChannelSMS) (Provider, error) {
	return c.registry.Get(ch.Provider)
}

func (c *Channel) DecryptConfig(ciphertext string) (string, error) {
	return c.cipher.Decrypt(ciphertext)
}

func buildHTTPRequest(req *appchannel.InboundRequest) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "", nil)
	r.Body = http.NoBody
	if len(req.Body) > 0 {
		r.Body = http.NoBody
	}
	for k, v := range req.Headers {
		r.Header.Set(k, v)
	}
	return r
}
