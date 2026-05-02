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

var inboundDomain = "inbound.elodesk.io"

var (
	reUUIDReceiver  = regexp.MustCompile(`reply\+([0-9a-f-]{36})@`)
	reFallbackMsgID = regexp.MustCompile(`<account/\d+/conversation/([0-9a-f-]{36})@`)
)

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

type ConversationFinder struct {
	deps      Deps
	accountID int64
	inboxID   int64
}

func NewConversationFinder(deps Deps, accountID, inboxID int64) *ConversationFinder {
	return &ConversationFinder{deps: deps, accountID: accountID, inboxID: inboxID}
}

func (f *ConversationFinder) Resolve(ctx context.Context, env *Envelope) (*model.Conversation, bool, error) {
	if conv, err := f.byUUIDReceiver(ctx, env); err == nil && conv != nil {
		return conv, false, nil
	}

	if env.InReplyTo != "" {
		if conv, err := f.byMessageID(ctx, env.InReplyTo); err == nil && conv != nil {
			return conv, false, nil
		}
	}

	for i := len(env.References) - 1; i >= 0; i-- {
		if conv, err := f.byMessageID(ctx, env.References[i]); err == nil && conv != nil {
			return conv, false, nil
		}
	}

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
	msg, err := f.deps.MessageRepo.FindBySourceIDInbox(ctx, msgID, f.inboxID)
	if err == nil {
		conv, err := f.deps.ConversationRepo.FindByUUID(ctx, "", f.accountID, f.inboxID)
		_ = conv
		_ = err
		return f.convByMessageConvID(ctx, msg)
	}

	if m := reFallbackMsgID.FindStringSubmatch(msgID); m != nil {
		uuid := m[1]
		conv, err := f.deps.ConversationRepo.FindByUUID(ctx, uuid, f.accountID, f.inboxID)
		if err == nil {
			return conv, nil
		}
	}
	return nil, nil
}

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

func extractName(addr string) string {
	if i := strings.Index(addr, "<"); i > 0 {
		return strings.TrimSpace(addr[:i])
	}
	return ""
}
