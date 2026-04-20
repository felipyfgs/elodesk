package whatsapp

import (
	"context"
	"encoding/json"
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

const (
	WaMaxRetries = 5
)

type WaSendPayload struct {
	ChannelID          int64  `json:"channelId"`
	AccountID          int64  `json:"accountId"`
	InboxID            int64  `json:"inboxId"`
	ConversationID     int64  `json:"conversationId"`
	MessageID          int64  `json:"messageId"`
	To                 string `json:"to"`
	Content            string `json:"content,omitempty"`
	MediaURL           string `json:"mediaUrl,omitempty"`
	MediaType          string `json:"mediaType,omitempty"`
	TemplateName       string `json:"templateName,omitempty"`
	TemplateLang       string `json:"templateLang,omitempty"`
	TemplateComponents string `json:"templateComponents,omitempty"`
	ApiKeyCiphertext   string `json:"apiKeyCiphertext"`
	PhoneNumberID      string `json:"phoneNumberId,omitempty"`
	BusinessAccountID  string `json:"businessAccountId,omitempty"`
	Provider           string `json:"provider"`
	DeliveryID         string `json:"deliveryId"`
}

type Service struct {
	channelWhatsappRepo *repo.ChannelWhatsAppRepo
	inboxRepo           *repo.InboxRepo
	messageRepo         *repo.MessageRepo
	conversationRepo    *repo.ConversationRepo
	contactService      *service.ContactService
	realtimeSvc         *service.RealtimeService
	cipher              *crypto.Cipher
	dedup               *appchannel.DedupLock
	reauth              *reauth.Tracker
	asynqClient         *asynq.Client
	httpClient          *http.Client
}

func NewService(
	channelWhatsappRepo *repo.ChannelWhatsAppRepo,
	inboxRepo *repo.InboxRepo,
	messageRepo *repo.MessageRepo,
	conversationRepo *repo.ConversationRepo,
	contactService *service.ContactService,
	realtimeSvc *service.RealtimeService,
	cipher *crypto.Cipher,
	dedup *appchannel.DedupLock,
	reauth *reauth.Tracker,
	asynqClient *asynq.Client,
	httpClient *http.Client,
) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Service{
		channelWhatsappRepo: channelWhatsappRepo,
		inboxRepo:           inboxRepo,
		messageRepo:         messageRepo,
		conversationRepo:    conversationRepo,
		contactService:      contactService,
		realtimeSvc:         realtimeSvc,
		cipher:              cipher,
		dedup:               dedup,
		reauth:              reauth,
		asynqClient:         asynqClient,
		httpClient:          httpClient,
	}
}

func (s *Service) HandleInbound(ctx context.Context, identifier string, req *appchannel.InboundRequest) error {
	inbox, err := s.findInboxByIdentifier(ctx, identifier)
	if err != nil {
		return err
	}

	ch, err := s.channelWhatsappRepo.FindByID(ctx, inbox.ChannelID, inbox.AccountID)
	if err != nil {
		return fmt.Errorf("whatsapp service: find channel: %w", err)
	}

	provider, err := ProviderForType(ch.Provider, s.httpClient)
	if err != nil {
		return err
	}

	result, err := provider.ParsePayload(ctx, req.Body)
	if err != nil {
		return fmt.Errorf("whatsapp service: parse payload: %w", err)
	}

	for _, im := range result.Messages {
		if err := s.processInboundMessage(ctx, ch, inbox, im); err != nil {
			logger.Error().Str("component", "channel.whatsapp").Err(err).
				Str("sourceId", im.SourceID).Msg("process inbound message")
			continue
		}
	}

	for _, su := range result.Statuses {
		if err := s.processStatusUpdate(ctx, ch, su); err != nil {
			logger.Error().Str("component", "channel.whatsapp").Err(err).
				Str("sourceId", su.SourceID).Msg("process status update")
			continue
		}
	}

	return nil
}

func (s *Service) findInboxByIdentifier(ctx context.Context, identifier string) (*model.Inbox, error) {
	return s.inboxRepo.FindByIdentifier(ctx, identifier)
}

func (s *Service) processInboundMessage(ctx context.Context, ch *model.ChannelWhatsApp, inbox *model.Inbox, im appchannel.InboundMessage) error {
	dk := dedupKey(im.SourceID)
	acquired, err := s.dedup.Acquire(ctx, dk)
	if err != nil {
		return fmt.Errorf("dedup acquire: %w", err)
	}
	if !acquired {
		logger.Debug().Str("component", "channel.whatsapp").Str("sourceId", im.SourceID).Msg("duplicate message, skipping")
		return nil
	}

	contact, err := s.upsertContact(ctx, ch.AccountID, inbox.ID, im)
	if err != nil {
		return fmt.Errorf("upsert contact: %w", err)
	}

	convo, err := s.ensureConversation(ctx, ch.AccountID, inbox.ID, contact.ID)
	if err != nil {
		return fmt.Errorf("ensure conversation: %w", err)
	}

	msgType := model.MessageIncoming
	contentType := model.ContentTypeText
	if im.MediaType != "" {
		switch im.MediaType {
		case "image":
			contentType = model.ContentTypeImage
		case "video":
			contentType = model.ContentTypeVideo
		case "audio":
			contentType = model.ContentTypeAudio
		default:
			contentType = model.ContentTypeFile
		}
	}

	if im.ExternalEcho {
		msgType = model.MessageOutgoing
	}

	var contentAttrs *string
	if im.MediaURL != "" {
		attrs := map[string]interface{}{
			"external_source_urls": []string{im.MediaURL},
			"source_id":            im.SourceID,
		}
		b, _ := json.Marshal(attrs)
		s := string(b)
		contentAttrs = &s
	}

	var content *string
	if im.Content != "" {
		content = &im.Content
	}

	msg := &model.Message{
		AccountID:    ch.AccountID,
		InboxID:      inbox.ID,
		MessageType:  msgType,
		ContentType:  contentType,
		Content:      content,
		SourceID:     &im.SourceID,
		Status:       model.MessageSent,
		ContentAttrs: contentAttrs,
	}

	created, err := s.messageRepo.Create(ctx, msg)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	if err := s.messageRepo.UpdateConversationID(ctx, created.ID, ch.AccountID, convo.ID); err != nil {
		logger.Error().Str("component", "channel.whatsapp").Err(err).Msg("link message to conversation")
	}

	s.realtimeSvc.BroadcastConversationEvent(convo.ID, "message.created", created)

	return nil
}

func (s *Service) processStatusUpdate(ctx context.Context, ch *model.ChannelWhatsApp, su appchannel.StatusUpdate) error {
	msg, err := s.messageRepo.FindBySourceID(ctx, su.SourceID, ch.AccountID)
	if err != nil {
		return fmt.Errorf("find message by source_id: %w", err)
	}

	var extErr *string
	if su.ExternalError != "" {
		extErr = &su.ExternalError
	}

	updated, err := s.messageRepo.UpdateStatus(ctx, msg.ID, ch.AccountID, su.Status, extErr)
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}

	s.realtimeSvc.BroadcastConversationEvent(msg.ConversationID, "message.status_changed", updated)

	return nil
}

func (s *Service) upsertContact(ctx context.Context, accountID, inboxID int64, im appchannel.InboundMessage) (*model.Contact, error) {
	phone := normalizePhone(im.From)
	contact := &model.Contact{
		PhoneNumber: &phone,
		Identifier:  &phone,
		Name:        "",
	}
	created, err := s.contactService.Create(ctx, accountID, contact)
	if err != nil {
		return nil, err
	}
	if err := s.contactService.EnsureContactInbox(ctx, created.ID, inboxID, im.From); err != nil {
		return nil, fmt.Errorf("ensure contact inbox: %w", err)
	}
	return created, nil
}

func (s *Service) ensureConversation(ctx context.Context, accountID, inboxID, contactID int64) (*model.Conversation, error) {
	return s.conversationRepo.EnsureOpen(ctx, accountID, inboxID, contactID)
}

func (s *Service) SendOutbound(ctx context.Context, ch *model.ChannelWhatsApp, to string, content string) (string, error) {
	apiKey, err := s.cipher.Decrypt(ch.ApiKeyCiphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt api key: %w", err)
	}

	provider, err := ProviderForType(ch.Provider, s.httpClient)
	if err != nil {
		return "", err
	}

	sendCtx := ctx
	if ch.PhoneNumberID != nil {
		sendCtx = context.WithValue(ctx, ctxKeyPhoneNumberID{}, *ch.PhoneNumberID)
	}

	sourceID, err := provider.Send(sendCtx, apiKey, normalizePhone(to), content, "", "", "", "", "")
	if err != nil {
		perr, ok := err.(*ProviderError)
		if ok && (perr.StatusCode == 401 || perr.StatusCode == 403) {
			prompt, trackerErr := s.reauth.RecordError(ctx, fmt.Sprintf("wa:%d", ch.ID))
			if trackerErr != nil {
				logger.Error().Str("component", "channel.whatsapp").Err(trackerErr).Msg("reauth tracker error")
			}
			if prompt {
				_ = s.channelWhatsappRepo.SetRequiresReauth(ctx, ch.ID, true)
				s.realtimeSvc.BroadcastAccountEvent(ch.AccountID, "channel.reauth_required", map[string]interface{}{
					"channelId":   ch.ID,
					"channelType": "whatsapp",
				})
			}
		}
		return "", err
	}

	_ = s.reauth.Reset(ctx, fmt.Sprintf("wa:%d", ch.ID))
	return sourceID, nil
}

func (s *Service) EnqueueAsyncSend(ctx context.Context, payload *WaSendPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal async send payload: %w", err)
	}

	task := asynq.NewTask(TypeChannelWaSend, data, asynq.MaxRetry(WaMaxRetries))

	_, err = s.asynqClient.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("enqueue async send: %w", err)
	}

	logger.Info().Str("component", "channel.whatsapp").
		Int64("channelId", payload.ChannelID).
		Int64("messageId", payload.MessageID).
		Msg("enqueued async send")

	return nil
}

func (s *Service) VerifyHandshake(ctx context.Context, ch *model.ChannelWhatsApp, query map[string]string) (string, bool) {
	if ch.Provider != "whatsapp_cloud" {
		return "", false
	}
	if ch.WebhookVerifyTokenCiphertext == nil {
		return "", false
	}
	token, err := s.cipher.Decrypt(*ch.WebhookVerifyTokenCiphertext)
	if err != nil {
		logger.Error().Str("component", "channel.whatsapp").Err(err).Msg("decrypt verify token")
		return "", false
	}
	provider := NewCloudProvider(s.httpClient)
	return provider.VerifyHandshake(ctx, query, token)
}

func (s *Service) SyncTemplatesForChannel(ctx context.Context, ch *model.ChannelWhatsApp) ([]appchannel.Template, error) {
	apiKey, err := s.cipher.Decrypt(ch.ApiKeyCiphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}

	provider, err := ProviderForType(ch.Provider, s.httpClient)
	if err != nil {
		return nil, err
	}

	businessAccountID := ""
	if ch.BusinessAccountID != nil {
		businessAccountID = *ch.BusinessAccountID
	}
	phoneNumberID := ""
	if ch.PhoneNumberID != nil {
		phoneNumberID = *ch.PhoneNumberID
	}

	templates, err := provider.SyncTemplates(ctx, apiKey, businessAccountID, phoneNumberID)
	if err != nil {
		return nil, err
	}

	if len(templates) > 0 {
		b, _ := json.Marshal(templates)
		str := string(b)
		if err := s.channelWhatsappRepo.UpdateTemplates(ctx, ch.ID, &str); err != nil {
			logger.Error().Str("component", "channel.whatsapp").Err(err).Msg("update templates")
		}
	}

	return templates, nil
}
