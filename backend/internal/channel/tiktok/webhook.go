package tiktok

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// ErrReauthRequired is returned by outbound/sink operations when TikTok responds
// with 401/403, indicating the current credentials are no longer valid.
var ErrReauthRequired = errors.New("tiktok: reauth required")

const (
	dedupKeyPrefix   = "elodesk:tiktok:"
	maxSignatureSkew = 5 * time.Second
)

// VerifySignature validates the `Tiktok-Signature: t=TIMESTAMP,s=HEX-HMAC`
// header against the raw body using the TikTok app secret.
// https://business-api.tiktok.com/portal/docs?id=1832190670631937
func VerifySignature(secret string, rawBody []byte, signatureHeader string, now time.Time) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}
	parts := strings.Split(signatureHeader, ",")
	var ts, sig string
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "s":
			sig = kv[1]
		}
	}
	if ts == "" || sig == "" {
		return false
	}
	timestamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if delta := now.Unix() - timestamp; delta < 0 || time.Duration(delta)*time.Second > maxSignatureSkew {
		return false
	}

	signingPayload := ts + "." + string(rawBody)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingPayload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// ProcessWebhook parses a TikTok webhook event and upserts inbound messages.
// outgoing echo (im_send_msg) is still recorded but marked as external_echo so
// the UI can display operator replies typed directly in the TikTok app.
func ProcessWebhook(
	ctx context.Context,
	rawEvent []byte,
	ch *model.ChannelTiktok,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	var evt WebhookEvent
	if err := json.Unmarshal(rawEvent, &evt); err != nil {
		return fmt.Errorf("tiktok webhook: unmarshal event: %w", err)
	}

	switch evt.Event {
	case EventReceiveMsg, EventSendMsg:
		// supported below
	case EventMarkRead:
		// read status not modelled yet; ignore silently
		return nil
	default:
		logger.Info().Str("component", "channel.tiktok").Str("event", evt.Event).Msg("ignored unsupported event")
		return nil
	}

	var content EventContent
	if err := json.Unmarshal([]byte(evt.Content), &content); err != nil {
		return fmt.Errorf("tiktok webhook: unmarshal content: %w", err)
	}

	outgoingEcho := evt.Event == EventSendMsg
	messageType := model.MessageIncoming
	if outgoingEcho {
		messageType = model.MessageOutgoing
	}

	// For incoming events the contact is `from_user`; for outgoing echoes it's `to_user`.
	var sourceUserID, displayName string
	if outgoingEcho {
		if content.ToUser != nil {
			sourceUserID = content.ToUser.ID
			displayName = content.ToUser.Username
		}
	} else {
		if content.FromUser != nil {
			sourceUserID = content.FromUser.ID
			displayName = content.FromUser.Username
		}
	}
	if sourceUserID == "" {
		return nil
	}

	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKeyPrefix+content.MessageID)
		if err != nil {
			logger.Warn().Str("component", "channel.tiktok").Err(err).Msg("dedup acquire error")
		}
		if !ok {
			return nil
		}
	}

	ci, err := contactInboxRepo.FindBySourceID(ctx, sourceUserID, inbox.ID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return fmt.Errorf("find contact inbox: %w", err)
		}
		if displayName == "" {
			displayName = sourceUserID
		}
		contact := &model.Contact{
			AccountID:  ch.AccountID,
			Name:       displayName,
			Identifier: &sourceUserID,
		}
		if err := contactRepo.Create(ctx, contact); err != nil {
			return fmt.Errorf("create contact: %w", err)
		}
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   inbox.ID,
			SourceID:  sourceUserID,
		}
		if err := contactInboxRepo.Create(ctx, ci); err != nil {
			return fmt.Errorf("create contact inbox: %w", err)
		}
	}

	if c, cErr := contactRepo.FindByID(ctx, ci.ContactID, ch.AccountID); cErr == nil && c.Blocked {
		logger.Warn().Str("component", "channel.tiktok").Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return nil
	}

	conv, err := conversationRepo.EnsureOpen(ctx, ch.AccountID, inbox.ID, ci.ContactID)
	if err != nil {
		return fmt.Errorf("ensure open conversation: %w", err)
	}

	messageContent, contentType, attrs := extractContent(&content, outgoingEcho, ch.BusinessID)

	dbMsg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    messageType,
		ContentType:    contentType,
		Content:        &messageContent,
		SourceID:       &content.MessageID,
		ContentAttrs:   attrs,
	}
	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func extractContent(c *EventContent, outgoingEcho bool, businessID string) (string, model.MessageContentType, *string) {
	base := map[string]any{
		"tiktok_conversation_id": c.ConversationID,
	}
	if c.Referenced != nil && c.Referenced.ReferencedMessageID != "" {
		base["in_reply_to_external_id"] = c.Referenced.ReferencedMessageID
	}
	if outgoingEcho {
		base["external_echo"] = true
	}
	_ = businessID

	switch c.Type {
	case MessageTypeText:
		if c.Text != nil {
			attrs := encodeAttrs(base)
			return c.Text.Body, model.ContentTypeText, attrs
		}
	case MessageTypeImage:
		if c.Image != nil {
			base["media_id"] = c.Image.MediaID
		}
		attrs := encodeAttrs(base)
		return "", model.ContentTypeImage, attrs
	}

	base["is_unsupported"] = true
	attrs := encodeAttrs(base)
	return "[unsupported]", model.ContentTypeText, attrs
}

func encodeAttrs(attrs map[string]any) *string {
	if len(attrs) == 0 {
		return nil
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		return nil
	}
	out := string(data)
	return &out
}
