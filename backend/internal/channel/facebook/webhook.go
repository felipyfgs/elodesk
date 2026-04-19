package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"backend/internal/channel"
	"backend/internal/channel/meta"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

const dedupKeyPrefix = "elodesk:meta:"

// ProcessWebhook parses and processes a Facebook Messenger webhook payload.
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
	var payload meta.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("facebook webhook: unmarshal: %w", err)
	}

	for _, entry := range payload.Entry {
		// Process regular messaging entries
		for _, me := range entry.Messaging {
			if err := processEntry(ctx, me, false, inbox, accountID, dedup, asynqClient, contactRepo, contactInboxRepo, conversationRepo, messageRepo); err != nil {
				logger.Warn().Str("component", "facebook.webhook").Err(err).Msg("process messaging entry")
			}
		}
		// Process standby entries (another app has primary receiver control)
		for _, me := range entry.Standby {
			if err := processEntry(ctx, me, true, inbox, accountID, dedup, asynqClient, contactRepo, contactInboxRepo, conversationRepo, messageRepo); err != nil {
				logger.Warn().Str("component", "facebook.webhook").Err(err).Msg("process standby entry")
			}
		}
	}
	return nil
}

func processEntry(
	ctx context.Context,
	me meta.MessagingEntry,
	isStandby bool,
	inbox *model.Inbox,
	accountID int64,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	// Handle delivery watermark updates
	if me.Delivery != nil {
		if inbox == nil || contactInboxRepo == nil {
			return nil
		}
		ci, err := contactInboxRepo.FindBySourceID(ctx, me.Sender.ID, inbox.ID)
		if err == nil {
			conv, convErr := conversationRepo.EnsureOpen(ctx, accountID, inbox.ID, ci.ContactID)
			if convErr == nil {
				_ = ProcessDelivery(ctx, conv.ID, accountID, me.Delivery.Watermark, messageRepo)
			}
		}
		return nil
	}

	if me.Message == nil {
		return nil
	}

	mid := me.Message.Mid
	if mid == "" {
		return nil
	}

	if me.Message.IsEcho {
		return scheduleEchoProcessing(ctx, mid, inbox.ID, me, isStandby, asynqClient)
	}

	if dedup == nil || inbox == nil {
		return nil
	}

	ok, err := dedup.Acquire(ctx, dedupKeyPrefix+mid)
	if err != nil {
		logger.Warn().Str("component", "facebook.webhook").Str("mid", mid).Err(err).Msg("dedup acquire error")
	}
	if !ok {
		return nil // duplicate
	}

	return persistInboundMessage(ctx, me, mid, isStandby, inbox, accountID, contactRepo, contactInboxRepo, conversationRepo, messageRepo)
}

func persistInboundMessage(
	ctx context.Context,
	me meta.MessagingEntry,
	mid string,
	isStandby bool,
	inbox *model.Inbox,
	accountID int64,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
) error {
	senderID := me.Sender.ID

	ci, err := contactInboxRepo.FindBySourceID(ctx, senderID, inbox.ID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return fmt.Errorf("find contact inbox: %w", err)
		}
		contact := &model.Contact{
			AccountID: accountID,
			Name:      "Facebook User",
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

	conv, err := conversationRepo.EnsureOpen(ctx, accountID, inbox.ID, ci.ContactID)
	if err != nil {
		return fmt.Errorf("ensure open conversation: %w", err)
	}

	content := me.Message.Text
	sourceID := mid

	var contentAttrs *string
	if isStandby {
		s := `{"source":"standby"}`
		contentAttrs = &s
	}

	ts := time.Unix(me.Timestamp/1000, 0)
	_ = ts

	msg := &model.Message{
		AccountID:      accountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   contentAttrs,
	}

	if _, err := messageRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func scheduleEchoProcessing(ctx context.Context, mid string, inboxID int64, me meta.MessagingEntry, isStandby bool, asynqClient *asynq.Client) error {
	if asynqClient == nil {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"mid":       mid,
		"inboxId":   inboxID,
		"isStandby": isStandby,
		"entry":     me,
	})
	if err != nil {
		return err
	}
	task := asynq.NewTask("channel:facebook:echo", payload, asynq.ProcessIn(2*time.Second))
	_, err = asynqClient.EnqueueContext(ctx, task)
	return err
}
