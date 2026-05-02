package channel

import (
	"context"
	"encoding/json"
	"errors"
)

type Kind string

const (
	KindApi          Kind = "Channel::Api"
	KindSms          Kind = "Channel::Sms"
	KindWhatsapp     Kind = "Channel::Whatsapp"
	KindInstagram    Kind = "Channel::Instagram"
	KindFacebookPage Kind = "Channel::FacebookPage"
	KindWebWidget    Kind = "Channel::WebWidget"
	KindTelegram     Kind = "Channel::Telegram"
	KindLine         Kind = "Channel::Line"
	KindTiktok       Kind = "Channel::Tiktok"
	KindTwilio       Kind = "Channel::Twilio"
	KindTwitter      Kind = "Channel::Twitter"
)

var ErrUnsupported = errors.New("channel: operation not supported")

type InboundRequest struct {
	Body       []byte
	Headers    map[string]string
	Query      map[string]string
	PathParams map[string]string
}

type InboundMessage struct {
	SourceID     string          `json:"source_id"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Content      string          `json:"content,omitempty"`
	MediaURL     string          `json:"media_url,omitempty"`
	MediaType    string          `json:"media_type,omitempty"`
	Timestamp    int64           `json:"timestamp"`
	IsEcho       bool            `json:"is_echo"`
	ExternalEcho bool            `json:"external_echo,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

type StatusUpdate struct {
	SourceID      string `json:"source_id"`
	Status        string `json:"status"`
	ExternalError string `json:"external_error,omitempty"`
}

type InboundResult struct {
	Messages []InboundMessage
	Statuses []StatusUpdate
}

type OutboundMessage struct {
	ChannelID          int64  `json:"channel_id"`
	To                 string `json:"to"`
	Content            string `json:"content,omitempty"`
	MediaURL           string `json:"media_url,omitempty"`
	MediaType          string `json:"media_type,omitempty"`
	TemplateName       string `json:"template_name,omitempty"`
	TemplateLang       string `json:"template_lang,omitempty"`
	TemplateComponents string `json:"template_components,omitempty"`
}

type Template struct {
	Name       string          `json:"name"`
	Language   string          `json:"language"`
	Status     string          `json:"status"`
	Components json.RawMessage `json:"components,omitempty"`
}

type Channel interface {
	Kind() Kind
	HandleInbound(ctx context.Context, req *InboundRequest) (*InboundResult, error)
	SendOutbound(ctx context.Context, msg *OutboundMessage) (sourceID string, err error)
	SyncTemplates(ctx context.Context) ([]Template, error)
}
