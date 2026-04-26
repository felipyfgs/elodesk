package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/realtime"
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
