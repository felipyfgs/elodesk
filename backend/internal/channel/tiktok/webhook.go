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

var ErrReauthRequired = errors.New("tiktok: reauth required")

const (
	dedupKeyPrefix   = "elodesk:tiktok:"
	maxSignatureSkew = 5 * time.Second
)

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
	case EventMarkRead:
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

	messageContent, contentType, attrs := extractContent(&content, outgoingEcho)

	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    messageType,
		ContentType:    contentType,
		Content:        &messageContent,
		SourceID:       &content.MessageID,
		ContentAttrs:   attrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func extractContent(c *EventContent, outgoingEcho bool) (string, model.MessageContentType, *string) {
	base := map[string]any{
		"tiktok_conversation_id": c.ConversationID,
	}
	if c.Referenced != nil && c.Referenced.ReferencedMessageID != "" {
		base["in_reply_to_external_id"] = c.Referenced.ReferencedMessageID
	}
	if outgoingEcho {
		base["external_echo"] = true
	}

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
