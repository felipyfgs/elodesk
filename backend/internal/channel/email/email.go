package email

import (
	"context"
	"fmt"

	"backend/internal/model"
)

type Channel struct {
	DecryptFn func(string) (string, error)
}

func NewChannel(decryptFn func(string) (string, error)) *Channel {
	return &Channel{DecryptFn: decryptFn}
}

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

func ReplyAddress(convUUID string) string {
	return fmt.Sprintf("reply+%s@%s", convUUID, inboundDomain)
}

func InboundDomain() string { return inboundDomain }
