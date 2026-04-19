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
)

var ErrUnsupported = errors.New("channel: operation not supported")

type InboundRequest struct {
	Body       []byte
	Headers    map[string]string
	Query      map[string]string
	PathParams map[string]string
}

type InboundMessage struct {
	SourceID     string          `json:"sourceId"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Content      string          `json:"content,omitempty"`
	MediaURL     string          `json:"mediaUrl,omitempty"`
	MediaType    string          `json:"mediaType,omitempty"`
	Timestamp    int64           `json:"timestamp"`
	IsEcho       bool            `json:"isEcho"`
	ExternalEcho bool            `json:"externalEcho,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

type StatusUpdate struct {
	SourceID      string `json:"sourceId"`
	Status        string `json:"status"`
	ExternalError string `json:"externalError,omitempty"`
}

type InboundResult struct {
	Messages []InboundMessage
	Statuses []StatusUpdate
}

type OutboundMessage struct {
	ChannelID          int64  `json:"channelId"`
	To                 string `json:"to"`
	Content            string `json:"content,omitempty"`
	MediaURL           string `json:"mediaUrl,omitempty"`
	MediaType          string `json:"mediaType,omitempty"`
	TemplateName       string `json:"templateName,omitempty"`
	TemplateLang       string `json:"templateLang,omitempty"`
	TemplateComponents string `json:"templateComponents,omitempty"`
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
