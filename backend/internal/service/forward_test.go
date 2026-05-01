package service

import (
	"errors"
	"testing"

	"backend/internal/model"
)

// strPtr returns a pointer to the given string. Test-only helper.
func strPtr(s string) *string { return &s }

func TestSourceIDForChannel_PhoneChannelsPreferE164(t *testing.T) {
	contact := &model.Contact{
		PhoneE164:   strPtr("+5511999998888"),
		PhoneNumber: strPtr("11 99999-8888"),
	}
	for _, ch := range []string{"Channel::Whatsapp", "Channel::Sms", "Channel::Twilio"} {
		got, err := sourceIDForChannel(ch, contact)
		if err != nil {
			t.Errorf("%s: unexpected err %v", ch, err)
			continue
		}
		if got != "+5511999998888" {
			t.Errorf("%s: got %q, want %q", ch, got, "+5511999998888")
		}
	}
}

func TestSourceIDForChannel_PhoneChannelsFallBackToPhoneNumber(t *testing.T) {
	contact := &model.Contact{PhoneNumber: strPtr("11999998888")}
	got, err := sourceIDForChannel("Channel::Whatsapp", contact)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "11999998888" {
		t.Errorf("got %q, want %q", got, "11999998888")
	}
}

func TestSourceIDForChannel_PhoneChannelMissingNumberErrors(t *testing.T) {
	got, err := sourceIDForChannel("Channel::Whatsapp", &model.Contact{})
	if err == nil {
		t.Fatalf("expected error, got %q", got)
	}
}

func TestSourceIDForChannel_EmailUsesEmail(t *testing.T) {
	contact := &model.Contact{Email: strPtr("foo@bar.com")}
	got, err := sourceIDForChannel("Channel::Email", contact)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "foo@bar.com" {
		t.Errorf("got %q, want %q", got, "foo@bar.com")
	}
}

func TestSourceIDForChannel_EmailMissingErrors(t *testing.T) {
	if _, err := sourceIDForChannel("Channel::Email", &model.Contact{}); err == nil {
		t.Fatal("expected error for email channel without email")
	}
}

func TestSourceIDForChannel_DefaultUsesIdentifier(t *testing.T) {
	contact := &model.Contact{Identifier: strPtr("tg-12345")}
	for _, ch := range []string{"Channel::Telegram", "Channel::Line", "Channel::Instagram", "Channel::Api"} {
		got, err := sourceIDForChannel(ch, contact)
		if err != nil {
			t.Errorf("%s: unexpected err %v", ch, err)
			continue
		}
		if got != "tg-12345" {
			t.Errorf("%s: got %q, want %q", ch, got, "tg-12345")
		}
	}
}

func TestSourceIDForChannel_DefaultMissingIdentifierErrors(t *testing.T) {
	if _, err := sourceIDForChannel("Channel::Telegram", &model.Contact{}); err == nil {
		t.Fatal("expected error for identifier channel without identifier")
	}
}

func TestResolveRootForwardID_ReturnsSelfWhenNotForward(t *testing.T) {
	got := resolveRootForwardID(model.Message{ID: 7})
	if got != 7 {
		t.Errorf("got %d, want 7", got)
	}
}

func TestResolveRootForwardID_ReturnsRootWhenForwardChain(t *testing.T) {
	root := int64(42)
	got := resolveRootForwardID(model.Message{ID: 100, ForwardedFromMessageID: &root})
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestForwardService_ForwardMessages_RejectsEmptySource(t *testing.T) {
	svc := &ForwardService{}
	_, err := svc.ForwardMessages(t.Context(), 1, 1, nil, []ForwardTarget{{ConversationID: 1}})
	if !errors.Is(err, ErrForwardEmptySource) {
		t.Errorf("err = %v, want ErrForwardEmptySource", err)
	}
}

func TestForwardService_ForwardMessages_RejectsTooManyMessages(t *testing.T) {
	svc := &ForwardService{}
	ids := []int64{1, 2, 3, 4, 5, 6}
	_, err := svc.ForwardMessages(t.Context(), 1, 1, ids, []ForwardTarget{{ConversationID: 1}})
	if !errors.Is(err, ErrForwardLimitExceeded) {
		t.Errorf("err = %v, want ErrForwardLimitExceeded", err)
	}
}

func TestForwardService_ForwardMessages_RejectsEmptyTargets(t *testing.T) {
	svc := &ForwardService{}
	_, err := svc.ForwardMessages(t.Context(), 1, 1, []int64{1}, nil)
	if !errors.Is(err, ErrForwardNoTargets) {
		t.Errorf("err = %v, want ErrForwardNoTargets", err)
	}
}

func TestForwardService_ForwardMessages_RejectsTooManyTargets(t *testing.T) {
	svc := &ForwardService{}
	targets := []ForwardTarget{
		{ConversationID: 1}, {ConversationID: 2}, {ConversationID: 3},
		{ConversationID: 4}, {ConversationID: 5}, {ConversationID: 6},
	}
	_, err := svc.ForwardMessages(t.Context(), 1, 1, []int64{1}, targets)
	if !errors.Is(err, ErrForwardTargetsLimit) {
		t.Errorf("err = %v, want ErrForwardTargetsLimit", err)
	}
}
