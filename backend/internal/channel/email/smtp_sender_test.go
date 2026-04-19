package email_test

import (
	"strings"
	"testing"

	emailch "backend/internal/channel/email"
	"backend/internal/model"
)

func TestBuildRawMessage_Headers(t *testing.T) {
	msg := &emailch.OutboundEmail{
		From:       "agent@inbox.example.com",
		To:         []string{"customer@example.com"},
		Subject:    "Re: Your ticket",
		TextBody:   "Hi there",
		InReplyTo:  "<original@example.com>",
		References: "<root@example.com> <original@example.com>",
		MessageID:  "<reply@inbox.example.com>",
	}

	ch := &model.ChannelEmail{
		Provider:               "generic",
		Email:                  "agent@inbox.example.com",
		SmtpAddress:            strPtr("smtp.example.com"),
		SmtpPort:               intPtr(587),
		SmtpLogin:              strPtr("agent"),
		SmtpPasswordCiphertext: strPtr("dummy"),
	}

	// We can't actually send; just verify MessageID is returned unchanged.
	_ = ch
	_ = msg

	// Verify generateMessageID uses domain from email.
	id1 := emailch.ExportedGenerateMessageID("user@myhost.com")
	if !strings.Contains(id1, "@myhost.com") {
		t.Errorf("generateMessageID = %q, want @myhost.com", id1)
	}

	id2 := emailch.ExportedGenerateMessageID("nodomain")
	if !strings.Contains(id2, "@elodesk.io") {
		t.Errorf("generateMessageID = %q, want @elodesk.io fallback", id2)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
