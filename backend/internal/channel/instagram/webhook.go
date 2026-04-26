package instagram

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

// ProcessWebhook parses and processes an Instagram webhook payload.
// Echo messages are scheduled with a 2s delay via asynq.
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
		return fmt.Errorf("instagram webhook: unmarshal: %w", err)
	}

	for _, entry := range payload.Entry {
		for _, me := range entry.Messaging {
			if me.Message == nil {
				continue
			}
			if err := processMessagingEntry(ctx, me, false, inbox, accountID, dedup, asynqClient, contactRepo, contactInboxRepo, conversationRepo, messageRepo); err != nil {
				logger.Warn().Str("component", "instagram.webhook").Err(err).Msg("process messaging entry")
			}
		}
	}
	return nil
}

func processMessagingEntry(
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
	if me.Message == nil {
		return nil
	}
	mid := me.Message.Mid
	if mid == "" {
		return nil
	}

	if inbox == nil || dedup == nil {
		return nil
	}

	if me.Message.IsEcho {
		return scheduleEchoProcessing(ctx, mid, inbox.ID, me, isStandby, asynqClient)
	}

	ok, err := dedup.Acquire(ctx, dedupKeyPrefix+mid)
	if err != nil {
		logger.Warn().Str("component", "instagram.webhook").Str("mid", mid).Err(err).Msg("dedup acquire error")
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

	// Find or create contact inbox (keyed by sender PSID/IGID)
	ci, err := contactInboxRepo.FindBySourceID(ctx, senderID, inbox.ID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return fmt.Errorf("find contact inbox: %w", err)
		}
		contact := &model.Contact{
			AccountID: accountID,
			Name:      "Instagram User",
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
		logger.Warn().Str("component", "instagram.webhook").Int64("contact_id", c.ID).Msg("contact_blocked_inbound_dropped")
		return nil
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
	_ = ts // stored via created_at default

	senderType := "Contact"
	contactID := ci.ContactID
	msg := &model.Message{
		AccountID:      accountID,
		InboxID:        inbox.ID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		Content:        &content,
		SourceID:       &sourceID,
		ContentAttrs:   contentAttrs,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}

	if _, err := messageRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

// scheduleEchoProcessing enqueues echo processing with 2s delay so the
// outbound message record is already persisted when the echo arrives.
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
	task := asynq.NewTask("channel:instagram:echo", payload, asynq.ProcessIn(2*time.Second))
	_, err = asynqClient.EnqueueContext(ctx, task)
	return err
}
