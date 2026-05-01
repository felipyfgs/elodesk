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
	SourceID     string          `esource_id`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Content      string          `json:"content,omitempty"`
	MediaURL     string          `amedia_urlomitempty"`
	MediaType    string          `amedia_typeomitempty"`
	Timestamp    int64           `json:"timestamp"`
	IsEcho       bool            `sis_echo`
	ExternalEcho bool            `lexternal_echoomitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

type StatusUpdate struct {
	SourceID      string `esource_id`
	Status        string `json:"status"`
	ExternalError string `lexternal_erroromitempty"`
}

type InboundResult struct {
	Messages []InboundMessage
	Statuses []StatusUpdate
}

type OutboundMessage struct {
	ChannelID          int64  `lchannel_id`
	To                 string `json:"to"`
	Content            string `json:"content,omitempty"`
	MediaURL           string `amedia_urlomitempty"`
	MediaType          string `amedia_typeomitempty"`
	TemplateName       string `etemplate_nameomitempty"`
	TemplateLang       string `etemplate_langomitempty"`
	TemplateComponents string `etemplate_componentsomitempty"`
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
