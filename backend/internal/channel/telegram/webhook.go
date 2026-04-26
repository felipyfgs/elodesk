package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hibiken/asynq"

	"backend/internal/channel"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const dedupKeyPrefix = "elodesk:telegram:"

func ProcessWebhook(
	ctx context.Context,
	body []byte,
	inbox *model.Inbox,
	accountID int64,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		return fmt.Errorf("telegram webhook: unmarshal: %w", err)
	}

	if update.Message != nil {
		return processMessage(ctx, update.Message, inbox, accountID, dedup, asynqClient, contactRepo, contactInboxRepo, conversationRepo, messageRepo)
	}

	if update.EditedMessage != nil {
		logger.Info().Str("component", "telegram.webhook").Int64("messageId", update.EditedMessage.MessageID).Msg("edited message ignored (MVP)")
		return nil
	}

	if update.CallbackQuery != nil {
		return processCallbackQuery(ctx, update.CallbackQuery, inbox, accountID, dedup, asynqClient, contactRepo, contactInboxRepo, conversationRepo, messageRepo)
	}

	return nil
}

func processMessage(
	ctx context.Context,
	msg *Message,
	inbox *model.Inbox,
	accountID int64,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	if msg.Chat.Type != "private" {
		logger.Info().Str("component", "telegram.webhook").Str("chatType", msg.Chat.Type).Msg("non-private chat ignored")
		return nil
	}

	sourceID := strconv.FormatInt(msg.MessageID, 10)
	dedupKey := dedupKeyPrefix + sourceID
	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKey)
		if err != nil {
			logger.Warn().Str("component", "telegram.webhook").Err(err).Msg("dedup acquire error")
		}
		if !ok {
			return nil
		}
	}

	senderID := strconv.FormatInt(msg.From.ID, 10)
	displayName := msg.From.FirstName
	if msg.From.LastName != "" {
		displayName += " " + msg.From.LastName
	}

	ci, err := contactInboxRepo.FindBySourceID(ctx, senderID, inbox.ID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return fmt.Errorf("find contact inbox: %w", err)
		}
		contact := &model.Contact{
			AccountID:  accountID,
			Name:       displayName,
			Identifier: &senderID,
		}
		if msg.From.Username != "" {
			username := "@" + msg.From.Username
			contact.Identifier = &username
		}
		if err := contactRepo.Create(ctx, contact); err != nil {
			return fmt.Errorf("create contact: %w", err)
		}
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   inbox.ID,
			SourceID:  senderID,
		}
		if err := contactInboxRepo.Create(ctx, ci); err != nil {
			return fmt.Errorf("create contact inbox: %w", err)
		}
	}

	if c, cErr := contactRepo.FindByID(ctx, ci.ContactID, accountID); cErr == nil && c.Blocked {
		logger.Warn().Str("component", "telegram.webhook").Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return nil
	}

	conv, err := conversationRepo.EnsureOpen(ctx, accountID, inbox.ID, ci.ContactID)
	if err != nil {
		return fmt.Errorf("ensure open conversation: %w", err)
	}

	content, contentType, contentAttrs := extractContent(msg)

	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      accountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    contentType,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   contentAttrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}

	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}

func extractContent(msg *Message) (string, model.MessageContentType, *string) {
	if msg.Text != nil && *msg.Text != "" {
		return *msg.Text, model.ContentTypeText, nil
	}

	if len(msg.Photo) > 0 {
		largest := msg.Photo[0]
		for _, p := range msg.Photo {
			if p.Width > largest.Width {
				largest = p
			}
		}
		attrs := fmt.Sprintf(`{"file_id":"%s","file_size":%d}`, largest.FileID, largest.FileSize)
		return "", model.ContentTypeImage, &attrs
	}

	if msg.Video != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","mime_type":"%s","file_size":%d}`, msg.Video.FileID, msg.Video.MimeType, msg.Video.FileSize)
		return "", model.ContentTypeVideo, &attrs
	}

	if msg.Audio != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","mime_type":"%s","file_size":%d}`, msg.Audio.FileID, msg.Audio.MimeType, msg.Audio.FileSize)
		return "", model.ContentTypeAudio, &attrs
	}

	if msg.Voice != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","mime_type":"%s","file_size":%d}`, msg.Voice.FileID, msg.Voice.MimeType, msg.Voice.FileSize)
		return "", model.ContentTypeAudio, &attrs
	}

	if msg.VideoNote != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","file_size":%d}`, msg.VideoNote.FileID, msg.VideoNote.FileSize)
		return "", model.ContentTypeVideo, &attrs
	}

	if msg.Document != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","file_name":"%s","mime_type":"%s","file_size":%d}`, msg.Document.FileID, msg.Document.FileName, msg.Document.MimeType, msg.Document.FileSize)
		return "", model.ContentTypeFile, &attrs
	}

	if msg.Sticker != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","emoji":"sticker","file_size":%d}`, msg.Sticker.FileID, msg.Sticker.FileSize)
		return "", model.ContentTypeSticker, &attrs
	}

	if msg.Animation != nil {
		attrs := fmt.Sprintf(`{"file_id":"%s","mime_type":"%s","file_size":%d}`, msg.Animation.FileID, msg.Animation.MimeType, msg.Animation.FileSize)
		return "", model.ContentTypeFile, &attrs
	}

	if msg.Location != nil {
		attrs := fmt.Sprintf(`{"longitude":%f,"latitude":%f}`, msg.Location.Longitude, msg.Location.Latitude)
		return "", model.ContentTypeText, &attrs
	}

	if msg.Contact != nil {
		attrs := fmt.Sprintf(`{"phone_number":"%s","first_name":"%s"}`, msg.Contact.PhoneNumber, msg.Contact.FirstName)
		return "", model.ContentTypeText, &attrs
	}

	attrs := `{"unsupported":true}`
	return "[unsupported]", model.ContentTypeText, &attrs
}

func processCallbackQuery(
	ctx context.Context,
	cq *CallbackQuery,
	inbox *model.Inbox,
	accountID int64,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	if cq.Message == nil {
		return nil
	}

	msg := cq.Message
	if msg.Chat.Type != "private" {
		return nil
	}

	sourceID := "cq_" + cq.ID
	dedupKey := dedupKeyPrefix + sourceID
	if dedup != nil {
		ok, err := dedup.Acquire(ctx, dedupKey)
		if err != nil {
			logger.Warn().Str("component", "telegram.webhook").Err(err).Msg("dedup acquire error for callback")
		}
		if !ok {
			return nil
		}
	}

	senderID := strconv.FormatInt(cq.From.ID, 10)
	displayName := cq.From.FirstName
	if cq.From.LastName != "" {
		displayName += " " + cq.From.LastName
	}

	ci, err := contactInboxRepo.FindBySourceID(ctx, senderID, inbox.ID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return fmt.Errorf("find contact inbox: %w", err)
		}
		contact := &model.Contact{
			AccountID:  accountID,
			Name:       displayName,
			Identifier: &senderID,
		}
		if cq.From.Username != "" {
			username := "@" + cq.From.Username
			contact.Identifier = &username
		}
		if err := contactRepo.Create(ctx, contact); err != nil {
			return fmt.Errorf("create contact: %w", err)
		}
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   inbox.ID,
			SourceID:  senderID,
		}
		if err := contactInboxRepo.Create(ctx, ci); err != nil {
			return fmt.Errorf("create contact inbox: %w", err)
		}
	}

	if c, cErr := contactRepo.FindByID(ctx, ci.ContactID, accountID); cErr == nil && c.Blocked {
		logger.Warn().Str("component", "telegram.webhook").Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return nil
	}

	conv, err := conversationRepo.EnsureOpen(ctx, accountID, inbox.ID, ci.ContactID)
	if err != nil {
		return fmt.Errorf("ensure open conversation: %w", err)
	}

	content := cq.Data
	attrs := fmt.Sprintf(`{"callback_query_id":"%s","button_text":"%s"}`, cq.ID, cq.Data)

	senderType := "Contact"
	contactID := ci.ContactID
	dbMsg := &model.Message{
		AccountID:      accountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   &attrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}

	_ = time.Now()

	if _, err := messageRepo.Create(ctx, dbMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}
