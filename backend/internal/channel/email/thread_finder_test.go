package email_test

import (
	"context"
	"testing"

	emailch "backend/internal/channel/email"
	"backend/internal/model"
	"backend/internal/repo"
)

// --- stubs ---

type stubConvRepo struct {
	byUUID map[string]*model.Conversation
	byID   map[int64]*model.Conversation
}

func (r *stubConvRepo) FindByUUID(_ context.Context, uuid string, _, _ int64) (*model.Conversation, error) {
	if c, ok := r.byUUID[uuid]; ok {
		return c, nil
	}
	return nil, repo.ErrConversationNotFound
}
func (r *stubConvRepo) FindByConvID(_ context.Context, convID, _ int64) (*model.Conversation, error) {
	if c, ok := r.byID[convID]; ok {
		return c, nil
	}
	return nil, repo.ErrConversationNotFound
}

type stubMsgRepo struct {
	bySourceInbox map[string]*model.Message
}

func (r *stubMsgRepo) FindBySourceIDInbox(_ context.Context, sourceID string, _ int64) (*model.Message, error) {
	if m, ok := r.bySourceInbox[sourceID]; ok {
		return m, nil
	}
	return nil, repo.ErrMessageNotFound
}

type stubContactRepo struct {
	byEmail map[string]*model.Contact
	created []*model.Contact
}

func (r *stubContactRepo) FindByEmail(_ context.Context, email string, _ int64) (*model.Contact, error) {
	if c, ok := r.byEmail[email]; ok {
		return c, nil
	}
	return nil, repo.ErrContactNotFound
}
func (r *stubContactRepo) Create(_ context.Context, m *model.Contact) error {
	m.ID = int64(len(r.created) + 100)
	r.created = append(r.created, m)
	return nil
}

type stubCIRepo struct {
	existing *model.ContactInbox
	created  []*model.ContactInbox
}

func (r *stubCIRepo) FindByContactAndInbox(_ context.Context, _, _ int64) (*model.ContactInbox, error) {
	if r.existing != nil {
		return r.existing, nil
	}
	return nil, repo.ErrContactInboxNotFound
}
func (r *stubCIRepo) Create(_ context.Context, m *model.ContactInbox) error {
	m.ID = int64(len(r.created) + 200)
	r.created = append(r.created, m)
	return nil
}

func makeDeps(convRepo *stubConvRepo, msgRepo *stubMsgRepo, contactRepo *stubContactRepo, ciRepo *stubCIRepo) emailch.Deps {
	var convID int64
	return emailch.Deps{
		ConversationRepo: convRepo,
		MessageRepo:      msgRepo,
		ContactRepo:      contactRepo,
		ContactInboxRepo: ciRepo,
		ConversationCreate: func(_ context.Context, conv *model.Conversation) error {
			convID++
			conv.ID = convID
			conv.UUID = "new-uuid"
			return nil
		},
	}
}

// --- tests ---

func TestThreadFinder_UUIDReceiver(t *testing.T) {
	conv := &model.Conversation{ID: 1, UUID: "aaaabbbb-cccc-dddd-eeee-ffffffffffff"}
	convRepo := &stubConvRepo{byUUID: map[string]*model.Conversation{conv.UUID: conv}}
	deps := makeDeps(convRepo, &stubMsgRepo{}, &stubContactRepo{byEmail: map[string]*model.Contact{}}, &stubCIRepo{})
	finder := emailch.NewConversationFinder(deps, 1, 1)

	env := &emailch.Envelope{
		To:      []string{"reply+" + conv.UUID + "@inbound.elodesk.io"},
		From:    "sender@example.com",
		Subject: "Re: test",
	}
	result, created, err := finder.Resolve(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected existing conversation, got created=true")
	}
	if result.ID != conv.ID {
		t.Errorf("conversation ID = %d, want %d", result.ID, conv.ID)
	}
}

func TestThreadFinder_InReplyTo(t *testing.T) {
	conv := &model.Conversation{ID: 5, UUID: "conv-uuid"}
	convRepo := &stubConvRepo{byID: map[int64]*model.Conversation{5: conv}}
	msgRepo := &stubMsgRepo{
		bySourceInbox: map[string]*model.Message{
			"<original@example.com>": {ID: 10, ConversationID: 5},
		},
	}
	deps := makeDeps(convRepo, msgRepo, &stubContactRepo{byEmail: map[string]*model.Contact{}}, &stubCIRepo{})
	finder := emailch.NewConversationFinder(deps, 1, 1)

	env := &emailch.Envelope{
		From:      "sender@example.com",
		InReplyTo: "<original@example.com>",
	}
	result, created, err := finder.Resolve(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected existing conversation")
	}
	if result.ID != 5 {
		t.Errorf("conversation ID = %d, want 5", result.ID)
	}
}

func TestThreadFinder_CrossInboxRejected(t *testing.T) {
	// Message exists but belongs to a different inbox — FindBySourceIDInbox returns not-found.
	convRepo := &stubConvRepo{byUUID: map[string]*model.Conversation{}}
	msgRepo := &stubMsgRepo{bySourceInbox: map[string]*model.Message{}} // empty = anti-hijack
	contactRepo := &stubContactRepo{byEmail: map[string]*model.Contact{}}
	deps := makeDeps(convRepo, msgRepo, contactRepo, &stubCIRepo{})
	finder := emailch.NewConversationFinder(deps, 1, 1)

	env := &emailch.Envelope{
		From:      "attacker@evil.com",
		InReplyTo: "<victim@example.com>",
	}
	_, created, err := finder.Resolve(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected new conversation to be created (no cross-inbox match)")
	}
}

func TestThreadFinder_NewConversation(t *testing.T) {
	convRepo := &stubConvRepo{byUUID: map[string]*model.Conversation{}}
	contactRepo := &stubContactRepo{byEmail: map[string]*model.Contact{}}
	deps := makeDeps(convRepo, &stubMsgRepo{}, contactRepo, &stubCIRepo{})
	finder := emailch.NewConversationFinder(deps, 1, 1)

	env := &emailch.Envelope{From: "newuser@example.com", Subject: "Help!"}
	result, created, err := finder.Resolve(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected new conversation")
	}
	if result == nil {
		t.Error("expected non-nil conversation")
	}
	if len(contactRepo.created) != 1 {
		t.Errorf("expected 1 contact created, got %d", len(contactRepo.created))
	}
}
