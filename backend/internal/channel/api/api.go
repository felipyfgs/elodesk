package api

import (
	"context"

	"backend/internal/channel"
)

type Channel struct{}

func NewChannel() *Channel {
	return &Channel{}
}

func (c *Channel) Kind() channel.Kind { return channel.KindApi }

// /public/api/v1/inboxes/{identifier}/... and those handlers persist messages
func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	return nil, channel.ErrUnsupported
}

func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	return "", channel.ErrUnsupported
}

func (c *Channel) SyncTemplates(ctx context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}
