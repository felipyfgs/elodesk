package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
)

type capturedBroadcast struct {
	conversationID int64
	accountID      int64
	event          string
	payload        any
}

type mockRealtimeNotifier struct {
	calls []capturedBroadcast
}

func (m *mockRealtimeNotifier) Broadcast(conversationID, accountID int64, event string, payload any) {
	m.calls = append(m.calls, capturedBroadcast{
		conversationID: conversationID,
		accountID:      accountID,
		event:          event,
		payload:        payload,
	})
}

// TestMessageService_Create_EmitsRealtimeEvent covers the path after the
// message has been persisted: broadcastMessageEvent is what Create/SoftDelete/
// UpdateStatus call once the DB write succeeds, so exercising it directly
// verifies the event contract without spinning up pgx.
func TestMessageService_Create_EmitsRealtimeEvent(t *testing.T) {
	notifier := &mockRealtimeNotifier{}
	svc := &MessageService{realtime: notifier}

	msg := &model.Message{
		ID:             42,
		AccountID:      10,
		InboxID:        5,
		ConversationID: 77,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeText,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc.broadcastMessageEvent(context.Background(), realtime.EventMessageCreated, msg)

	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(notifier.calls))
	}
	got := notifier.calls[0]
	if got.event != realtime.EventMessageCreated {
		t.Errorf("event = %q, want %q", got.event, realtime.EventMessageCreated)
	}
	if got.conversationID != 77 {
		t.Errorf("conversationID = %d, want 77", got.conversationID)
	}
	if got.accountID != 10 {
		t.Errorf("accountID = %d, want 10", got.accountID)
	}
	payload, ok := got.payload.(dto.MessageResp)
	if !ok {
		t.Fatalf("payload type = %T, want dto.MessageResp", got.payload)
	}
	if payload.ID != 42 {
		t.Errorf("payload.ID = %d, want 42", payload.ID)
	}
}

func TestMessageService_Create_EchoIDRoundtrip(t *testing.T) {
	notifier := &mockRealtimeNotifier{}
	svc := &MessageService{realtime: notifier}

	echoAttrs := `{"echo_id":"ui-abc"}`
	msg := &model.Message{
		ID:             1,
		AccountID:      10,
		ConversationID: 77,
		ContentAttrs:   &echoAttrs,
	}

	svc.broadcastMessageEvent(context.Background(), realtime.EventMessageCreated, msg)

	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(notifier.calls))
	}
	payload := notifier.calls[0].payload.(dto.MessageResp)
	if payload.EchoID == nil {
		t.Fatalf("payload.EchoID is nil, want \"ui-abc\"")
	}
	if *payload.EchoID != "ui-abc" {
		t.Errorf("EchoID = %q, want %q", *payload.EchoID, "ui-abc")
	}
}

func TestMessageService_SoftDelete_EmitsDeleted(t *testing.T) {
	notifier := &mockRealtimeNotifier{}
	svc := &MessageService{realtime: notifier}

	msg := &model.Message{
		ID:             99,
		AccountID:      10,
		ConversationID: 77,
	}

	svc.broadcastMessageEvent(context.Background(), realtime.EventMessageDeleted, msg)

	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(notifier.calls))
	}
	if notifier.calls[0].event != realtime.EventMessageDeleted {
		t.Errorf("event = %q, want %q", notifier.calls[0].event, realtime.EventMessageDeleted)
	}
}

func TestMessageService_broadcastMessageEvent_NoOpWithoutNotifier(t *testing.T) {
	svc := &MessageService{}
	msg := &model.Message{ID: 1, ConversationID: 2, AccountID: 3}
	svc.broadcastMessageEvent(context.Background(), realtime.EventMessageCreated, msg)
}

func TestMessageService_resolveSender_PreservesPresetSender(t *testing.T) {
	svc := &MessageService{}
	st := "User"
	id := int64(7)
	msg := &model.Message{MessageType: model.MessageOutgoing, SenderType: &st, SenderID: &id}

	if err := svc.resolveSender(context.Background(), msg); err != nil {
		t.Fatalf("resolveSender err = %v, want nil", err)
	}
	if msg.SenderType == nil || *msg.SenderType != "User" {
		t.Errorf("SenderType mutated: %v", msg.SenderType)
	}
	if msg.SenderID == nil || *msg.SenderID != 7 {
		t.Errorf("SenderID mutated: %v", msg.SenderID)
	}
}

func TestMessageService_resolveSender_OutgoingWithoutSender(t *testing.T) {
	svc := &MessageService{}
	msg := &model.Message{MessageType: model.MessageOutgoing}

	err := svc.resolveSender(context.Background(), msg)
	if !errors.Is(err, ErrMessageMissingSender) {
		t.Fatalf("err = %v, want ErrMessageMissingSender", err)
	}
}

// mockConversationStore satisfies the conversationStore interface used by
// MessageService. Tracks calls so tests can assert on what was invoked.
type mockConversationStore struct {
	conv             *model.Conversation
	hydrated         *repo.ConversationHydrated
	findByIDErr      error
	findByIDFullErr  error
	toggleStatusErr  error
	toggleStatusArgs *struct {
		id, accountID int64
		status        model.ConversationStatus
	}
	toggleStatusCalls int
	findByIDFullCalls int
}

func (m *mockConversationStore) FindByID(_ context.Context, id, accountID int64) (*model.Conversation, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.conv, nil
}

func (m *mockConversationStore) FindByIDFull(_ context.Context, accountID, id int64) (*repo.ConversationHydrated, error) {
	m.findByIDFullCalls++
	if m.findByIDFullErr != nil {
		return nil, m.findByIDFullErr
	}
	return m.hydrated, nil
}

func (m *mockConversationStore) ToggleStatus(_ context.Context, id, accountID int64, status model.ConversationStatus) (*model.Conversation, error) {
	m.toggleStatusCalls++
	m.toggleStatusArgs = &struct {
		id, accountID int64
		status        model.ConversationStatus
	}{id, accountID, status}
	if m.toggleStatusErr != nil {
		return nil, m.toggleStatusErr
	}
	updated := *m.conv
	updated.Status = status
	return &updated, nil
}

func (m *mockConversationStore) UpdateLastActivity(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (m *mockConversationStore) CountUnread(_ context.Context, _, _ int64) (int, error) {
	return 0, nil
}

func newReopenFixture(status model.ConversationStatus) (*MessageService, *mockConversationStore, *mockRealtimeNotifier) {
	conv := &model.Conversation{ID: 77, AccountID: 10, InboxID: 5, Status: status}
	store := &mockConversationStore{
		conv: conv,
		hydrated: &repo.ConversationHydrated{
			Conversation: model.Conversation{ID: 77, AccountID: 10, InboxID: 5, Status: model.ConversationOpen},
			Contact:      model.Contact{ID: 1, AccountID: 10},
			Inbox:        model.Inbox{ID: 5, AccountID: 10},
		},
	}
	notifier := &mockRealtimeNotifier{}
	svc := &MessageService{conversationRepo: store, realtime: notifier}
	return svc, store, notifier
}

func incomingMsg() *model.Message {
	return &model.Message{
		ID:             42,
		AccountID:      10,
		InboxID:        5,
		ConversationID: 77,
		MessageType:    model.MessageIncoming,
	}
}

func TestMessageService_reopenIfClosed_ReopensResolvedAndBroadcasts(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationResolved)

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 1 {
		t.Fatalf("ToggleStatus calls = %d, want 1", store.toggleStatusCalls)
	}
	if store.toggleStatusArgs.status != model.ConversationOpen {
		t.Errorf("toggle status = %d, want ConversationOpen", store.toggleStatusArgs.status)
	}
	if store.toggleStatusArgs.id != 77 || store.toggleStatusArgs.accountID != 10 {
		t.Errorf("toggle args = (%d, %d), want (77, 10)", store.toggleStatusArgs.id, store.toggleStatusArgs.accountID)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(notifier.calls))
	}
	if notifier.calls[0].event != realtime.EventConversationUpdated {
		t.Errorf("event = %q, want %q", notifier.calls[0].event, realtime.EventConversationUpdated)
	}
	if notifier.calls[0].conversationID != 77 || notifier.calls[0].accountID != 10 {
		t.Errorf("broadcast scope = (%d, %d), want (77, 10)", notifier.calls[0].conversationID, notifier.calls[0].accountID)
	}
	if _, ok := notifier.calls[0].payload.(dto.ConversationResp); !ok {
		t.Errorf("payload type = %T, want dto.ConversationResp", notifier.calls[0].payload)
	}
}

func TestMessageService_reopenIfClosed_ReopensSnoozed(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationSnoozed)

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 1 {
		t.Errorf("ToggleStatus calls = %d, want 1", store.toggleStatusCalls)
	}
	if len(notifier.calls) != 1 {
		t.Errorf("broadcasts = %d, want 1", len(notifier.calls))
	}
}

func TestMessageService_reopenIfClosed_SkipsAlreadyOpen(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationOpen)

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 0 {
		t.Errorf("ToggleStatus calls = %d, want 0", store.toggleStatusCalls)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcasts = %d, want 0", len(notifier.calls))
	}
}

func TestMessageService_reopenIfClosed_SkipsPending(t *testing.T) {
	// Pending is a triage state, not a closed state — inbound messages should
	// not auto-promote it to Open. Only Resolved and Snoozed reopen.
	svc, store, notifier := newReopenFixture(model.ConversationPending)

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 0 {
		t.Errorf("ToggleStatus calls = %d, want 0 (Pending must not auto-reopen)", store.toggleStatusCalls)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcasts = %d, want 0", len(notifier.calls))
	}
}

func TestMessageService_reopenIfClosed_ReopensOutgoing(t *testing.T) {
	// Diverge do Chatwoot: mensagens outgoing também reabrem (cobre echo
	// externo do wzap quando o operador envia do próprio WhatsApp).
	svc, store, notifier := newReopenFixture(model.ConversationResolved)
	msg := incomingMsg()
	msg.MessageType = model.MessageOutgoing

	svc.reopenIfClosed(context.Background(), msg)

	if store.toggleStatusCalls != 1 {
		t.Errorf("ToggleStatus calls = %d, want 1", store.toggleStatusCalls)
	}
	if store.toggleStatusArgs.status != model.ConversationOpen {
		t.Errorf("toggle status = %d, want ConversationOpen", store.toggleStatusArgs.status)
	}
	if len(notifier.calls) != 1 {
		t.Errorf("broadcasts = %d, want 1", len(notifier.calls))
	}
}

func TestMessageService_reopenIfClosed_SkipsActivity(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationResolved)
	msg := incomingMsg()
	msg.MessageType = model.MessageActivity

	svc.reopenIfClosed(context.Background(), msg)

	if store.toggleStatusCalls != 0 {
		t.Errorf("ToggleStatus called for activity message")
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcast emitted for activity message")
	}
}

func TestMessageService_reopenIfClosed_SkipsPrivateNote(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationResolved)
	msg := incomingMsg()
	msg.Private = true

	svc.reopenIfClosed(context.Background(), msg)

	if store.toggleStatusCalls != 0 {
		t.Errorf("ToggleStatus called for private note")
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcast emitted for private note")
	}
}

func TestMessageService_reopenIfClosed_NoOpWithoutRepo(t *testing.T) {
	notifier := &mockRealtimeNotifier{}
	svc := &MessageService{realtime: notifier}

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if len(notifier.calls) != 0 {
		t.Errorf("broadcasts = %d, want 0", len(notifier.calls))
	}
}

func TestMessageService_reopenIfClosed_NoBroadcastOnToggleError(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationResolved)
	store.toggleStatusErr = errors.New("db down")

	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 1 {
		t.Errorf("ToggleStatus calls = %d, want 1", store.toggleStatusCalls)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcast emitted despite toggle error")
	}
	if store.findByIDFullCalls != 0 {
		t.Errorf("FindByIDFull called despite toggle error")
	}
}

func TestMessageService_reopenIfClosed_HydrationErrorSwallowed(t *testing.T) {
	svc, store, notifier := newReopenFixture(model.ConversationResolved)
	store.findByIDFullErr = errors.New("hydrate failed")

	// Must not panic and must not broadcast — hydration failure swallows.
	svc.reopenIfClosed(context.Background(), incomingMsg())

	if store.toggleStatusCalls != 1 {
		t.Errorf("ToggleStatus calls = %d, want 1", store.toggleStatusCalls)
	}
	if len(notifier.calls) != 0 {
		t.Errorf("broadcast emitted despite hydration error")
	}
}
