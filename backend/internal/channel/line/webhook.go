package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const (
	dedupKeyPrefix      = "elodesk:line:"
	replyTokenAttrsKey  = "line_reply_token"
	lineChannelAttrsKey = "line_channel_id"
)

// VerifySignature computes HMAC-SHA256 over the raw request body and compares to the
// base64-encoded signature provided in the X-Line-Signature header.
// https://developers.line.biz/en/reference/messaging-api/#signature-validation
func VerifySignature(secret string, rawBody []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ProcessWebhook parses a LINE webhook payload and persists contacts/messages.
// The caller is expected to have already verified the X-Line-Signature header.
func ProcessWebhook(
	ctx context.Context,
	body []byte,
	ch *model.ChannelLine,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	api *APIClient,
	channelToken string,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("line webhook: unmarshal: %w", err)
	}

	for _, evt := range payload.Events {
		if err := processEvent(ctx, &evt, ch, inbox, dedup, api, channelToken,
			contactRepo, contactInboxRepo, conversationRepo, messageRepo); err != nil {
			logger.Warn().Str("component", "channel.line").Err(err).Str("eventType", evt.Type).Msg("line event processing error")
			continue
		}
	}
	return nil
}

func processEvent(
	ctx context.Context,
	evt *Event,
	ch *model.ChannelLine,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	api *APIClient,
	channelToken string,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	// Only process 1:1 messages from users; ignore group/room for now.
	if evt.Source.Type != "user" || evt.Source.UserID == "" {
		return nil
	}

	switch evt.Type {
	case EventTypeMessage:
		if evt.Message == nil {
			return nil
		}
		return processMessage(ctx, evt, ch, inbox, dedup, api, channelToken,
			contactRepo, contactInboxRepo, conversationRepo, messageRepo)
	case EventTypeFollow, EventTypeUnfollow:
		// ensure the contact exists but don't create a message
		_, _, err := ensureContactAndConversation(ctx, evt.Source.UserID, ch.AccountID, inbox.ID,
			api, channelToken, contactRepo, contactInboxRepo, conversationRepo)
		return err
	case EventTypePostback:
		if evt.Postback == nil {
			return nil
		}
		return processPostback(ctx, evt, ch, inbox, dedup,
			api, channelToken, contactRepo, contactInboxRepo, conversationRepo, messageRepo)
	}
	return nil
}

func processMessage(
	ctx context.Context,
	evt *Event,
	ch *model.ChannelLine,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	api *APIClient,
	channelToken string,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	sourceID := evt.Message.ID
	dedupKey := dedupKeyPrefix + sourceID
	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKey)
		if err != nil {
			logger.Warn().Str("component", "channel.line").Err(err).Msg("dedup acquire error")
		}
		if !ok {
			return nil
		}
	}

	ci, conv, err := ensureContactAndConversation(ctx, evt.Source.UserID, ch.AccountID, inbox.ID,
		api, channelToken, contactRepo, contactInboxRepo, conversationRepo)
	if err != nil {
		return err
	}
	if conv == nil {
		return nil
	}

	content, contentType, contentAttrs := extractContent(evt.Message)

	attrs := mergeContentAttrs(contentAttrs, map[string]any{
		replyTokenAttrsKey:  evt.ReplyToken,
		lineChannelAttrsKey: ch.LineChannelID,
	})

	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    contentType,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   attrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}

func processPostback(
	ctx context.Context,
	evt *Event,
	ch *model.ChannelLine,
	inbox *model.Inbox,
	dedup *channel.DedupLock,
	api *APIClient,
	channelToken string,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	sourceID := "pb_" + evt.WebhookEventID
	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKeyPrefix+sourceID)
		if err != nil {
			logger.Warn().Str("component", "channel.line").Err(err).Msg("dedup acquire postback error")
		}
		if !ok {
			return nil
		}
	}

	ci, conv, err := ensureContactAndConversation(ctx, evt.Source.UserID, ch.AccountID, inbox.ID,
		api, channelToken, contactRepo, contactInboxRepo, conversationRepo)
	if err != nil {
		return err
	}
	if conv == nil {
		return nil
	}

	content := evt.Postback.Data
	attrs := mergeContentAttrs(nil, map[string]any{
		"postback":          true,
		replyTokenAttrsKey:  evt.ReplyToken,
		lineChannelAttrsKey: ch.LineChannelID,
	})
	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   attrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func ensureContactAndConversation(
	ctx context.Context,
	userID string,
	accountID int64,
	inboxID int64,
	api *APIClient,
	channelToken string,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
) (*model.ContactInbox, *model.Conversation, error) {
	ci, err := contactInboxRepo.FindBySourceID(ctx, userID, inboxID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return nil, nil, fmt.Errorf("find contact inbox: %w", err)
		}

		displayName := userID
		if api != nil && channelToken != "" {
			if profile, perr := api.GetProfile(ctx, channelToken, userID); perr == nil && profile != nil {
				if profile.DisplayName != "" {
					displayName = profile.DisplayName
				}
			} else if perr != nil {
				logger.Warn().Str("component", "channel.line").Err(perr).Msg("line get profile failed")
			}
		}

		contact := &model.Contact{
			AccountID:  accountID,
			Name:       displayName,
			Identifier: &userID,
		}
		if err := contactRepo.Create(ctx, contact); err != nil {
			return nil, nil, fmt.Errorf("create contact: %w", err)
		}
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   inboxID,
			SourceID:  userID,
		}
		if err := contactInboxRepo.Create(ctx, ci); err != nil {
			return nil, nil, fmt.Errorf("create contact inbox: %w", err)
		}
	}

	if c, cErr := contactRepo.FindByID(ctx, ci.ContactID, accountID); cErr == nil && c.Blocked {
		logger.Warn().Str("component", "channel.line").Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return ci, nil, nil
	}

	conv, err := conversationRepo.EnsureOpen(ctx, accountID, inboxID, ci.ContactID)
	if err != nil {
		return nil, nil, fmt.Errorf("ensure open conversation: %w", err)
	}
	return ci, conv, nil
}

func extractContent(msg *EventMessage) (string, model.MessageContentType, *string) {
	switch msg.Type {
	case MessageTypeText:
		return msg.Text, model.ContentTypeText, nil
	case MessageTypeImage:
		attrs := encodeContentAttrs(map[string]any{"line_message_id": msg.ID})
		return "", model.ContentTypeImage, attrs
	case MessageTypeVideo:
		attrs := encodeContentAttrs(map[string]any{"line_message_id": msg.ID, "duration": msg.Duration})
		return "", model.ContentTypeVideo, attrs
	case MessageTypeAudio:
		attrs := encodeContentAttrs(map[string]any{"line_message_id": msg.ID, "duration": msg.Duration})
		return "", model.ContentTypeAudio, attrs
	case MessageTypeFile:
		attrs := encodeContentAttrs(map[string]any{"line_message_id": msg.ID, "file_name": msg.FileName, "file_size": msg.FileSize})
		return "", model.ContentTypeFile, attrs
	case MessageTypeSticker:
		stickerURL := fmt.Sprintf("https://stickershop.line-scdn.net/stickershop/v1/sticker/%s/android/sticker.png", msg.StickerID)
		content := fmt.Sprintf("![sticker-%s](%s)", msg.StickerID, stickerURL)
		attrs := encodeContentAttrs(map[string]any{"line_sticker_id": msg.StickerID, "package_id": msg.PackageID})
		return content, model.ContentTypeSticker, attrs
	case MessageTypeLocation:
		attrs := encodeContentAttrs(map[string]any{"latitude": msg.Latitude, "longitude": msg.Longitude, "address": msg.Address})
		return msg.Title, model.ContentTypeText, attrs
	}
	attrs := encodeContentAttrs(map[string]any{"unsupported": true})
	return "[unsupported]", model.ContentTypeText, attrs
}

func encodeContentAttrs(m map[string]any) *string {
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	s := string(data)
	return &s
}

func mergeContentAttrs(existing *string, extras map[string]any) *string {
	base := map[string]any{}
	if existing != nil && *existing != "" {
		if err := json.Unmarshal([]byte(*existing), &base); err != nil {
			base = map[string]any{}
		}
	}
	for k, v := range extras {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		base[k] = v
	}
	if len(base) == 0 {
		return nil
	}
	data, err := json.Marshal(base)
	if err != nil {
		return nil
	}
	out := string(data)
	return &out
}
