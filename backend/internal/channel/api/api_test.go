package api

import (
	"context"
	"errors"
	"testing"

	"backend/internal/channel"
)

func TestChannel_Kind(t *testing.T) {
	c := NewChannel()
	if got := c.Kind(); got != channel.KindApi {
		t.Fatalf("Kind() = %q, want %q", got, channel.KindApi)
	}
}

func TestChannel_HandleInboundUnsupported(t *testing.T) {
	c := NewChannel()
	_, err := c.HandleInbound(context.Background(), &channel.InboundRequest{})
	if !errors.Is(err, channel.ErrUnsupported) {
		t.Fatalf("HandleInbound err = %v, want ErrUnsupported", err)
	}
}

func TestChannel_SendOutboundUnsupported(t *testing.T) {
	c := NewChannel()
	_, err := c.SendOutbound(context.Background(), &channel.OutboundMessage{})
	if !errors.Is(err, channel.ErrUnsupported) {
		t.Fatalf("SendOutbound err = %v, want ErrUnsupported", err)
	}
}

func TestChannel_SyncTemplatesUnsupported(t *testing.T) {
	c := NewChannel()
	_, err := c.SyncTemplates(context.Background())
	if !errors.Is(err, channel.ErrUnsupported) {
		t.Fatalf("SyncTemplates err = %v, want ErrUnsupported", err)
	}
}
