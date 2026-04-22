// Package api implements channel.Channel for Channel::Api inboxes.
//
// Unlike other kinds, the Api channel does not have dedicated inbound webhooks
// or a direct outbound client. Integrators consume:
//
//   - Inbound: POST /public/api/v1/inboxes/{identifier}/... authenticated with
//     the api_access_token header (see middleware/api_token.go). The regular
//     public message handlers handle persistence — no kind-specific parsing.
//   - Outbound: OutboundWebhookService produces a webhook delivery using the
//     channel's own webhook_url + hmac_token (see service/outbound_webhook_service.go).
//
// The type here exists to register Channel::Api in channel.Registry so the
// discovery/enumeration APIs (frontend inbox listing, swagger docs) include it
// as a first-class kind, and to keep the interface consistent across all kinds.
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

// HandleInbound is not used for Channel::Api. Integrators POST directly to
// /public/api/v1/inboxes/{identifier}/... and those handlers persist messages
// without routing through channel.Channel.HandleInbound.
func (c *Channel) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	return nil, channel.ErrUnsupported
}

// SendOutbound is not used for Channel::Api. Outbound delivery goes through
// OutboundWebhookService → asynq → webhook.OutboundProcessor with the
// channel-specific webhook_url and hmac_token.
func (c *Channel) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	return "", channel.ErrUnsupported
}

// SyncTemplates is not supported for Channel::Api.
func (c *Channel) SyncTemplates(ctx context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}
