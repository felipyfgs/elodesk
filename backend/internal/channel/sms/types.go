package sms

import (
	"context"
	"fmt"
	"net/http"

	"backend/internal/model"
)

type InboundMessage struct {
	SourceID   string   `json:"sourceId"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Content    string   `json:"content,omitempty"`
	MediaURLs  []string `json:"mediaUrls,omitempty"`
	MediaTypes []string `json:"mediaTypes,omitempty"`
	Timestamp  int64    `json:"timestamp"`
}

type OutboundMessage struct {
	To       string   `json:"to"`
	Content  string   `json:"content,omitempty"`
	MediaURL []string `json:"mediaUrl,omitempty"`
}

type StatusCallback struct {
	SourceID      string `json:"sourceId"`
	Status        string `json:"status"`
	ExternalError string `json:"externalError,omitempty"`
}

type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("sms provider error: status=%d message=%s", e.StatusCode, e.Message)
}

func IsAuthError(err error) bool {
	pe, ok := err.(*ProviderError)
	return ok && (pe.StatusCode == 401 || pe.StatusCode == 403)
}

func IsRetryableError(err error) bool {
	pe, ok := err.(*ProviderError)
	if !ok {
		return false
	}
	return pe.StatusCode == 429 || pe.StatusCode >= 500
}

type Provider interface {
	Name() string
	VerifyWebhook(r *http.Request, channel *model.ChannelSMS) error
	ParseInbound(r *http.Request) (*InboundMessage, error)
	Send(ctx context.Context, channel *model.ChannelSMS, out *OutboundMessage, statusCallbackURL string) (sourceID string, err error)
	ParseDeliveryStatus(r *http.Request) (*StatusCallback, error)
	ValidateCredentials(ctx context.Context, config ProviderConfig) error
}
