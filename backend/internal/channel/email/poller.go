package email

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"backend/internal/channel/reauth"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// backoffSteps defines the wait durations after consecutive IMAP errors.
var backoffSteps = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
}

// PollDeps are the dependencies injected into every EmailPoller.
type PollDeps struct {
	ChannelEmailRepo   *repo.ChannelEmailRepo
	ConversationFinder func(accountID, inboxID int64) *ConversationFinder
	MessageRepo        *repo.MessageRepo
	AttachmentHandler  *AttachmentHandler
	DecryptFn          func(string) (string, error)
	Tracker            *reauth.Tracker
	InboxID            int64
}

// EmailPoller polls a single email channel's IMAP mailbox on a ticker.
type EmailPoller struct {
	ch       model.ChannelEmail
	deps     PollDeps
	interval time.Duration
}

func NewEmailPoller(ch model.ChannelEmail, deps PollDeps, interval time.Duration) *EmailPoller {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &EmailPoller{ch: ch, deps: deps, interval: interval}
}

// Run starts polling until ctx is cancelled.
func (p *EmailPoller) Run(ctx context.Context) {
	key := fmt.Sprintf("channel:email:%d", p.ch.ID)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	backoffIdx := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if err := p.poll(ctx, key); err != nil {
			logger.Warn().Str("component", "email-poller").Err(err).Int64("channelID", p.ch.ID).Msg("email poller error")

			hitThreshold, _ := p.deps.Tracker.RecordError(ctx, key)
			if hitThreshold {
				logger.Error().Str("component", "email-poller").Int64("channelID", p.ch.ID).Msg("email channel requires reauth")
				_ = p.deps.ChannelEmailRepo.SetRequiresReauth(ctx, p.ch.ID, true)
				return
			}

			// exponential backoff
			wait := backoffSteps[backoffIdx]
			if backoffIdx < len(backoffSteps)-1 {
				backoffIdx++
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		} else {
			backoffIdx = 0
			_ = p.deps.Tracker.Reset(ctx, key)
		}
	}
}

func (p *EmailPoller) poll(ctx context.Context, _ string) error {
	client, err := Connect(ctx, &p.ch, p.deps.DecryptFn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close() //nolint:errcheck

	msgs, err := client.FetchSince(uint32(p.ch.LastUIDSeen))
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	var maxUID uint32
	for _, fm := range msgs {
		if err := p.processMessage(ctx, fm); err != nil {
			logger.Warn().Str("component", "email-poller").Err(err).Uint32("uid", fm.UID).Int64("channelID", p.ch.ID).Msg("process message failed")
		}
		if fm.UID > maxUID {
			maxUID = fm.UID
		}
	}

	if maxUID > 0 {
		p.ch.LastUIDSeen = int64(maxUID)
		if err := p.deps.ChannelEmailRepo.UpdateLastUIDSeen(ctx, p.ch.ID, int64(maxUID)); err != nil {
			return fmt.Errorf("update last_uid_seen: %w", err)
		}
	}
	return nil
}

func (p *EmailPoller) processMessage(ctx context.Context, fm FetchedMessage) error {
	env, err := ParseMIME(bytes.NewReader(fm.Raw))
	if err != nil && env == nil {
		return fmt.Errorf("parse mime uid=%d: %w", fm.UID, err)
	}

	finder := p.deps.ConversationFinder(p.ch.AccountID, p.deps.InboxID)
	conv, _, err := finder.Resolve(ctx, env)
	if err != nil {
		return fmt.Errorf("thread resolve: %w", err)
	}

	msgID := env.MessageID
	var srcID *string
	if msgID != "" {
		srcID = &msgID
	}

	body := env.Text
	if body == "" {
		body = env.HTML
	}
	senderType := "Contact"
	contactID := conv.ContactID
	msg := &model.Message{
		AccountID:      p.ch.AccountID,
		InboxID:        p.deps.InboxID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeIncomingEmail,
		Content:        &body,
		SourceID:       srcID,
		Status:         model.MessageSent,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}

	created, err := p.deps.MessageRepo.Create(ctx, msg)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	if p.deps.AttachmentHandler != nil && len(env.Attachments) > 0 {
		_ = p.deps.AttachmentHandler.ProcessAttachments(ctx, created, env.Attachments)
	}
	return nil
}
