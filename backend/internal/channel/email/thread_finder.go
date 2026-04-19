package email

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"backend/internal/model"
	"backend/internal/repo"
)

// inboundDomain is the base domain used for reply+<uuid>@inbound.<domain>
// receiver addresses. Loaded from env via Config.
var inboundDomain = "inbound.elodesk.io"

var (
	reUUIDReceiver  = regexp.MustCompile(`reply\+([0-9a-f-]{36})@`)
	reFallbackMsgID = regexp.MustCompile(`<account/\d+/conversation/([0-9a-f-]{36})@`)
)

// Deps holds the repo dependencies needed by ConversationFinder.
type Deps struct {
	ConversationRepo interface {
		FindByUUID(ctx context.Context, uuid string, accountID, inboxID int64) (*model.Conversation, error)
	}
	MessageRepo interface {
		FindBySourceIDInbox(ctx context.Context, sourceID string, inboxID int64) (*model.Message, error)
	}
	ContactRepo interface {
		FindByEmail(ctx context.Context, email string, accountID int64) (*model.Contact, error)
		Create(ctx context.Context, m *model.Contact) error
	}
	ContactInboxRepo interface {
		FindByContactAndInbox(ctx context.Context, contactID, inboxID int64) (*model.ContactInbox, error)
		Create(ctx context.Context, m *model.ContactInbox) error
	}
	ConversationCreate func(ctx context.Context, conv *model.Conversation) error
}

// ConversationFinder resolves which conversation an inbound email belongs to.
type ConversationFinder struct {
	deps      Deps
	accountID int64
	inboxID   int64
}

func NewConversationFinder(deps Deps, accountID, inboxID int64) *ConversationFinder {
	return &ConversationFinder{deps: deps, accountID: accountID, inboxID: inboxID}
}

// Resolve tries four strategies in order and returns the conversation plus
// whether it was newly created.
func (f *ConversationFinder) Resolve(ctx context.Context, env *Envelope) (*model.Conversation, bool, error) {
	// Strategy 1 — UUID receiver
	if conv, err := f.byUUIDReceiver(ctx, env); err == nil && conv != nil {
		return conv, false, nil
	}

	// Strategy 2 — In-Reply-To
	if env.InReplyTo != "" {
		if conv, err := f.byMessageID(ctx, env.InReplyTo); err == nil && conv != nil {
			return conv, false, nil
		}
	}

	// Strategy 3 — References chain (last 50, newest first for speed)
	for i := len(env.References) - 1; i >= 0; i-- {
		if conv, err := f.byMessageID(ctx, env.References[i]); err == nil && conv != nil {
			return conv, false, nil
		}
	}

	// Strategy 4 — new conversation
	conv, err := f.newConversation(ctx, env)
	if err != nil {
		return nil, false, fmt.Errorf("thread_finder: create new conversation: %w", err)
	}
	return conv, true, nil
}

func (f *ConversationFinder) byUUIDReceiver(ctx context.Context, env *Envelope) (*model.Conversation, error) {
	for _, addr := range append(env.To, env.Cc...) {
		m := reUUIDReceiver.FindStringSubmatch(addr)
		if m == nil {
			continue
		}
		uuid := m[1]
		conv, err := f.deps.ConversationRepo.FindByUUID(ctx, uuid, f.accountID, f.inboxID)
		if err == nil {
			return conv, nil
		}
	}
	return nil, nil
}

func (f *ConversationFinder) byMessageID(ctx context.Context, msgID string) (*model.Conversation, error) {
	// Try direct source_id match (anti-hijack: scoped to this inbox).
	msg, err := f.deps.MessageRepo.FindBySourceIDInbox(ctx, msgID, f.inboxID)
	if err == nil {
		conv, err := f.deps.ConversationRepo.FindByUUID(ctx, "", f.accountID, f.inboxID)
		_ = conv
		// we need the conversation id from the message
		_ = err
		// Re-fetch by conversation id via a simpler approach: return the conversation
		// using the message's conversation_id - we'll store the convRepo as a wider iface.
		return f.convByMessageConvID(ctx, msg)
	}

	// Try fallback pattern <account/<id>/conversation/<uuid>@<domain>>
	if m := reFallbackMsgID.FindStringSubmatch(msgID); m != nil {
		uuid := m[1]
		conv, err := f.deps.ConversationRepo.FindByUUID(ctx, uuid, f.accountID, f.inboxID)
		if err == nil {
			return conv, nil
		}
	}
	return nil, nil
}

// convByMessageConvID looks up the conversation that owns msg.
// We do this by carrying a broader ConversationRepo interface.
func (f *ConversationFinder) convByMessageConvID(ctx context.Context, msg *model.Message) (*model.Conversation, error) {
	type convByIDer interface {
		FindByConvID(ctx context.Context, convID, accountID int64) (*model.Conversation, error)
	}
	if r, ok := f.deps.ConversationRepo.(convByIDer); ok {
		return r.FindByConvID(ctx, msg.ConversationID, f.accountID)
	}
	return nil, errors.New("conversation repo does not implement FindByConvID")
}

func (f *ConversationFinder) newConversation(ctx context.Context, env *Envelope) (*model.Conversation, error) {
	fromAddr := extractEmail(env.From)

	contact, err := f.deps.ContactRepo.FindByEmail(ctx, fromAddr, f.accountID)
	if err != nil {
		if !repo.IsErrNotFound(err) {
			return nil, err
		}
		name := extractName(env.From)
		if name == "" {
			name = fromAddr
		}
		contact = &model.Contact{
			AccountID: f.accountID,
			Name:      name,
			Email:     &fromAddr,
		}
		if err := f.deps.ContactRepo.Create(ctx, contact); err != nil {
			return nil, err
		}
	}

	ci, _ := f.deps.ContactInboxRepo.FindByContactAndInbox(ctx, contact.ID, f.inboxID)
	if ci == nil {
		ci = &model.ContactInbox{
			ContactID: contact.ID,
			InboxID:   f.inboxID,
			SourceID:  fromAddr,
		}
		if err := f.deps.ContactInboxRepo.Create(ctx, ci); err != nil {
			return nil, err
		}
	}

	conv := &model.Conversation{
		AccountID:      f.accountID,
		InboxID:        f.inboxID,
		ContactID:      contact.ID,
		ContactInboxID: &ci.ID,
		Status:         model.ConversationOpen,
	}
	if err := f.deps.ConversationCreate(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// extractEmail returns the bare email address from "Name <addr>" or "addr".
func extractEmail(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.Index(addr, "<"); i >= 0 {
		addr = addr[i+1:]
		if j := strings.Index(addr, ">"); j >= 0 {
			addr = addr[:j]
		}
	}
	return strings.ToLower(strings.TrimSpace(addr))
}

// extractName returns the display name portion of "Name <addr>".
func extractName(addr string) string {
	if i := strings.Index(addr, "<"); i > 0 {
		return strings.TrimSpace(addr[:i])
	}
	return ""
}
