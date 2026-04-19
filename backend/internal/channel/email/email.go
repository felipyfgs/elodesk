package email

import (
	"context"
	"fmt"

	"backend/internal/model"
)

// Channel implements the Channel::Email outbound path, dispatching by provider.
type Channel struct {
	DecryptFn func(string) (string, error)
}

func NewChannel(decryptFn func(string) (string, error)) *Channel {
	return &Channel{DecryptFn: decryptFn}
}

// SendOutbound sends msg via the appropriate transport based on ch.Provider.
// Returns the Message-ID used (stored as messages.source_id on the outbound row).
func (c *Channel) SendOutbound(ctx context.Context, ch *model.ChannelEmail, msg *OutboundEmail) (sourceID string, err error) {
	switch ch.Provider {
	case "generic":
		return SendSMTP(ch, msg, c.DecryptFn)
	case "google":
		return SendGmail(ctx, ch, msg, c.DecryptFn)
	case "microsoft":
		return SendGraph(ctx, ch, msg, c.DecryptFn)
	default:
		return "", fmt.Errorf("channel email: unknown provider %q", ch.Provider)
	}
}

// ReplyAddress builds the deterministic reply address for a conversation UUID.
// Relay must route reply+*@<inboundDomain> to POST /webhooks/email/inbound.
func ReplyAddress(convUUID string) string {
	return fmt.Sprintf("reply+%s@%s", convUUID, inboundDomain)
}

// InboundDomain returns the configured inbound relay domain.
func InboundDomain() string { return inboundDomain }
