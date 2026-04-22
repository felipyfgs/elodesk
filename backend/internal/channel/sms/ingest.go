package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	appchannel "backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type IngestService struct {
	channelSMSRepo   *repo.ChannelSMSRepo
	inboxRepo        *repo.InboxRepo
	contactService   *service.ContactService
	contactRepo      *repo.ContactRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	dedup            *appchannel.DedupLock
	media            *MediaHandler
	realtimeSvc      *service.RealtimeService
}

func NewIngestService(
	channelSMSRepo *repo.ChannelSMSRepo,
	inboxRepo *repo.InboxRepo,
	contactService *service.ContactService,
	contactRepo *repo.ContactRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	dedup *appchannel.DedupLock,
	media *MediaHandler,
	realtimeSvc *service.RealtimeService,
) *IngestService {
	return &IngestService{
		channelSMSRepo:   channelSMSRepo,
		inboxRepo:        inboxRepo,
		contactService:   contactService,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		dedup:            dedup,
		media:            media,
		realtimeSvc:      realtimeSvc,
	}
}

func (s *IngestService) IngestInbound(ctx context.Context, channel *model.ChannelSMS, inbound *InboundMessage) error {
	if channel.InboxID == nil {
		return fmt.Errorf("sms ingest: channel has no inbox_id")
	}

	inboxID := *channel.InboxID

	dk := fmt.Sprintf("sms:ingest:%s", inbound.SourceID)
	acquired, err := s.dedup.Acquire(ctx, dk)
	if err != nil {
		return fmt.Errorf("sms ingest: dedup acquire: %w", err)
	}
	if !acquired {
		logger.Debug().Str("component", "sms.ingest").Str("sourceId", inbound.SourceID).Msg("duplicate message, skipping")
		return nil
	}

	contact, err := s.upsertContact(ctx, channel.AccountID, inboxID, inbound)
	if err != nil {
		return fmt.Errorf("sms ingest: upsert contact: %w", err)
	}

	if contact.Blocked {
		logger.Warn().Str("component", "sms.ingest").Int64("contact_id", contact.ID).Str("sourceId", inbound.SourceID).Msg("contact_blocked_inbound_dropped")
		return nil
	}

	convo, err := s.conversationRepo.EnsureOpen(ctx, channel.AccountID, inboxID, contact.ID)
	if err != nil {
		return fmt.Errorf("sms ingest: ensure conversation: %w", err)
	}

	contentType := model.ContentTypeText
	if len(inbound.MediaURLs) > 0 {
		if len(inbound.MediaTypes) > 0 && inbound.MediaTypes[0] != "" {
			switch {
			case isImageMIME(inbound.MediaTypes[0]):
				contentType = model.ContentTypeImage
			case isVideoMIME(inbound.MediaTypes[0]):
				contentType = model.ContentTypeVideo
			case isAudioMIME(inbound.MediaTypes[0]):
				contentType = model.ContentTypeAudio
			default:
				contentType = model.ContentTypeFile
			}
		} else {
			contentType = model.ContentTypeFile
		}
	}

	var contentAttrs *string
	if len(inbound.MediaURLs) > 0 {
		attrs := map[string]interface{}{
			"external_source_urls": inbound.MediaURLs,
			"source_id":            inbound.SourceID,
		}
		b, _ := json.Marshal(attrs)
		s := string(b)
		contentAttrs = &s
	}

	var content *string
	if inbound.Content != "" {
		content = &inbound.Content
	}

	msg := &model.Message{
		AccountID:    channel.AccountID,
		InboxID:      inboxID,
		MessageType:  model.MessageIncoming,
		ContentType:  contentType,
		Content:      content,
		SourceID:     &inbound.SourceID,
		Status:       model.MessageSent,
		ContentAttrs: contentAttrs,
	}

	created, err := s.messageRepo.Create(ctx, msg)
	if err != nil {
		return fmt.Errorf("sms ingest: create message: %w", err)
	}

	if err := s.messageRepo.UpdateConversationID(ctx, created.ID, channel.AccountID, convo.ID); err != nil {
		logger.Error().Str("component", "sms.ingest").Err(err).Msg("link message to conversation")
	}

	s.realtimeSvc.BroadcastConversationEvent(convo.ID, "message.created", created)

	if len(inbound.MediaURLs) > 0 {
		go func() {
			bgCtx := context.Background()
			s.media.DownloadAndStoreAll(bgCtx, inbound.MediaURLs, inbound.MediaTypes,
				strconv.FormatInt(channel.AccountID, 10),
				strconv.FormatInt(inboxID, 10),
				strconv.FormatInt(created.ID, 10))
		}()
	}

	return nil
}

func (s *IngestService) upsertContact(ctx context.Context, accountID, inboxID int64, inbound *InboundMessage) (*model.Contact, error) {
	e164, valid := NormalizeE164(inbound.From)
	phone := inbound.From
	var phoneE164 *string
	if valid {
		phone = e164
		phoneE164 = &e164
	}

	existing, err := s.contactRepo.FindByPhoneE164(ctx, e164, accountID)
	if err == nil {
		contact := existing
		if err := s.contactService.EnsureContactInbox(ctx, contact.ID, inboxID, inbound.From); err != nil {
			logger.Warn().Str("component", "sms.ingest").Err(err).Msg("ensure contact inbox")
		}
		return contact, nil
	}

	contact := &model.Contact{
		PhoneNumber: &phone,
		PhoneE164:   phoneE164,
		Identifier:  &inbound.From,
		Name:        "",
	}

	created, err := s.contactService.Create(ctx, accountID, contact)
	if err != nil {
		return nil, err
	}

	if err := s.contactService.EnsureContactInbox(ctx, created.ID, inboxID, inbound.From); err != nil {
		return nil, fmt.Errorf("sms ingest: ensure contact inbox: %w", err)
	}

	return created, nil
}

func isImageMIME(m string) bool {
	return m == "image/jpeg" || m == "image/png" || m == "image/gif" || m == "image/webp"
}

func isVideoMIME(m string) bool {
	return m == "video/mp4" || m == "video/3gpp"
}

func isAudioMIME(m string) bool {
	return m == "audio/aac" || m == "audio/ogg" || m == "audio/mpeg"
}
