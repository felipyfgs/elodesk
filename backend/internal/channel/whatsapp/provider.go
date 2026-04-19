package whatsapp

import (
	"context"

	"backend/internal/channel"
)

type Provider interface {
	VerifyHandshake(ctx context.Context, query map[string]string, verifyToken string) (challenge string, ok bool)
	VerifySignature(ctx context.Context, body []byte, headers map[string]string, appSecret string) bool
	ParsePayload(ctx context.Context, body []byte) (*channel.InboundResult, error)
	Send(ctx context.Context, apiKey, to, content, mediaURL, mediaType, templateName, templateLang, templateComponents string) (sourceID string, err error)
	SyncTemplates(ctx context.Context, apiKey, businessAccountID, phoneNumberID string) ([]channel.Template, error)
	HeadersForRequest(apiKey string) map[string]string
}
