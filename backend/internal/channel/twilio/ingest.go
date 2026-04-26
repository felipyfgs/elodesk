package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	appchannel "backend/internal/channel"
	"backend/internal/channel/sms"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// InboundParams captures the subset of Twilio webhook form fields Elodesk
// cares about, extracted once from the incoming form.
type InboundParams struct {
	MessageSid string
	From       string
	To         string
	Body       string
	MediaURLs  []string
	MediaTypes []string
}

// ParseInbound reads the Messages.json-compatible form shape Twilio posts to
// our webhook and normalizes media URLs/types.
func ParseInbound(form url.Values) *InboundParams {
	p := &InboundParams{
		MessageSid: form.Get("MessageSid"),
		From:       form.Get("From"),
		To:         form.Get("To"),
		Body:       form.Get("Body"),
	}
	if n, err := strconv.Atoi(form.Get("NumMedia")); err == nil && n > 0 {
		p.MediaURLs = make([]string, 0, n)
		p.MediaTypes = make([]string, 0, n)
		for i := 0; i < n; i++ {
			p.MediaURLs = append(p.MediaURLs, form.Get(fmt.Sprintf("MediaUrl%d", i)))
			p.MediaTypes = append(p.MediaTypes, form.Get(fmt.Sprintf("MediaContentType%d", i)))
		}
	}
	return p
}

type Ingester struct {
	channelRepo      *repo.ChannelTwilioRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	dedup            *appchannel.DedupLock
}

func NewIngester(
	channelRepo *repo.ChannelTwilioRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	dedup *appchannel.DedupLock,
) *Ingester {
	return &Ingester{
		channelRepo:      channelRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		dedup:            dedup,
	}
}

func (i *Ingester) Ingest(ctx context.Context, ch *model.ChannelTwilio, inbox *model.Inbox, p *InboundParams) error {
	if p.MessageSid == "" {
		return fmt.Errorf("twilio ingest: missing MessageSid")
	}

	if i.dedup != nil {
		ok, err := i.dedup.Acquire(ctx, "twilio:ingest:"+p.MessageSid)
		if err != nil {
			logger.Warn().Str("component", "channel.twilio").Err(err).Msg("dedup acquire")
		}
		if !ok {
			return nil
		}
	}

	from := stripWhatsappPrefix(p.From)
	sourceKey := from
	phoneE164, valid := sms.NormalizeE164(from)
	if valid {
		sourceKey = phoneE164
	}

	contact, err := i.upsertContact(ctx, ch.AccountID, inbox.ID, from, phoneE164, valid, sourceKey)
	if err != nil {
		return fmt.Errorf("twilio ingest: upsert contact: %w", err)
	}

	if contact.Blocked {
		logger.Warn().Str("component", "channel.twilio").Int64("contact_id", contact.ID).Msg("contact_blocked_inbound_dropped")
		return nil
	}

	conv, err := i.conversationRepo.EnsureOpen(ctx, ch.AccountID, inbox.ID, contact.ID)
	if err != nil {
		return fmt.Errorf("twilio ingest: ensure conversation: %w", err)
	}

	contentType := model.ContentTypeText
	if len(p.MediaURLs) > 0 {
		if len(p.MediaTypes) > 0 && p.MediaTypes[0] != "" {
			switch {
			case strings.HasPrefix(p.MediaTypes[0], "image/"):
				contentType = model.ContentTypeImage
			case strings.HasPrefix(p.MediaTypes[0], "video/"):
				contentType = model.ContentTypeVideo
			case strings.HasPrefix(p.MediaTypes[0], "audio/"):
				contentType = model.ContentTypeAudio
			default:
				contentType = model.ContentTypeFile
			}
		} else {
			contentType = model.ContentTypeFile
		}
	}

	var contentAttrs *string
	if len(p.MediaURLs) > 0 {
		attrs := map[string]any{
			"external_source_urls": p.MediaURLs,
			"source_id":            p.MessageSid,
		}
		if b, err := json.Marshal(attrs); err == nil {
			s := string(b)
			contentAttrs = &s
		}
	}

	var content *string
	if p.Body != "" {
		b := p.Body
		content = &b
	}

	senderType := "Contact"
	contactID := contact.ID
	msg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    contentType,
		Content:        content,
		SourceID:       &p.MessageSid,
		Status:         model.MessageSent,
		ContentAttrs:   contentAttrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	if _, err := i.messageRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("twilio ingest: create message: %w", err)
	}
	return nil
}

func (i *Ingester) upsertContact(ctx context.Context, accountID, inboxID int64, phone, phoneE164 string, validE164 bool, sourceKey string) (*model.Contact, error) {
	if validE164 {
		if existing, err := i.contactRepo.FindByPhoneE164(ctx, phoneE164, accountID); err == nil {
			if err := i.ensureContactInbox(ctx, existing.ID, inboxID, sourceKey); err != nil {
				return nil, err
			}
			return existing, nil
		}
	}
	if existing, err := i.contactInboxRepo.FindBySourceID(ctx, sourceKey, inboxID); err == nil {
		contact, findErr := i.contactRepo.FindByID(ctx, existing.ContactID, accountID)
		if findErr == nil {
			return contact, nil
		}
	}

	contact := &model.Contact{
		AccountID:   accountID,
		Name:        phone,
		PhoneNumber: &phone,
	}
	if validE164 {
		contact.PhoneE164 = &phoneE164
	}
	if err := i.contactRepo.Create(ctx, contact); err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}
	if err := i.ensureContactInbox(ctx, contact.ID, inboxID, sourceKey); err != nil {
		return nil, err
	}
	return contact, nil
}

func (i *Ingester) ensureContactInbox(ctx context.Context, contactID, inboxID int64, sourceID string) error {
	if _, err := i.contactInboxRepo.FindBySourceID(ctx, sourceID, inboxID); err == nil {
		return nil
	}
	ci := &model.ContactInbox{ContactID: contactID, InboxID: inboxID, SourceID: sourceID}
	if err := i.contactInboxRepo.Create(ctx, ci); err != nil {
		return fmt.Errorf("create contact inbox: %w", err)
	}
	return nil
}

func stripWhatsappPrefix(s string) string {
	return strings.TrimPrefix(s, WhatsappPrefix)
}

// DetectMedium looks at the inbound "From" field and reports the medium the
// provider intended. whatsapp: prefix signals WhatsApp; anything else is SMS.
func DetectMedium(from string) model.TwilioMedium {
	if strings.HasPrefix(from, WhatsappPrefix) {
		return model.TwilioMediumWhatsApp
	}
	return model.TwilioMediumSMS
}
