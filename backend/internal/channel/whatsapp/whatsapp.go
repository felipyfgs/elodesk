package whatsapp

import (
	"context"
	"fmt"
	"net/http"

	"backend/internal/channel"
	"backend/internal/crypto"
	"backend/internal/model"
	"backend/internal/repo"
)

type WhatsApp struct {
	channelWhatsappRepo *repo.ChannelWhatsAppRepo
	inboxRepo           *repo.InboxRepo
	cipher              *crypto.Cipher
	httpClient          *http.Client
}

func NewWhatsApp(
	channelWhatsappRepo *repo.ChannelWhatsAppRepo,
	inboxRepo *repo.InboxRepo,
	cipher *crypto.Cipher,
	httpClient *http.Client,
) *WhatsApp {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}
	return &WhatsApp{
		channelWhatsappRepo: channelWhatsappRepo,
		inboxRepo:           inboxRepo,
		cipher:              cipher,
		httpClient:          httpClient,
	}
}

func (w *WhatsApp) Kind() channel.Kind {
	return channel.KindWhatsapp
}

func (w *WhatsApp) HandleInbound(ctx context.Context, req *channel.InboundRequest) (*channel.InboundResult, error) {
	identifier := req.PathParams["identifier"]
	if identifier == "" {
		return nil, fmt.Errorf("whatsapp: identifier required")
	}

	inbox, err := w.inboxRepo.FindByChannelID(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: find inbox by identifier: %w", err)
	}

	ch, err := w.channelWhatsappRepo.FindByID(ctx, inbox.ChannelID, inbox.AccountID)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: find channel: %w", err)
	}

	provider, err := ProviderForType(ch.Provider, w.httpClient)
	if err != nil {
		return nil, err
	}

	return provider.ParsePayload(ctx, req.Body)
}

func (w *WhatsApp) SendOutbound(ctx context.Context, msg *channel.OutboundMessage) (string, error) {
	ch, err := w.channelWhatsappRepo.FindByID(ctx, msg.ChannelID, 0)
	if err != nil {
		return "", fmt.Errorf("whatsapp: find channel for outbound: %w", err)
	}

	apiKey, err := w.cipher.Decrypt(ch.ApiKeyCiphertext)
	if err != nil {
		return "", fmt.Errorf("whatsapp: decrypt api key: %w", err)
	}

	provider, err := ProviderForType(ch.Provider, w.httpClient)
	if err != nil {
		return "", err
	}

	if ch.PhoneNumberID != nil {
		ctx = context.WithValue(ctx, ctxKeyPhoneNumberID{}, *ch.PhoneNumberID)
	}

	return provider.Send(ctx, apiKey, msg.To, msg.Content, msg.MediaURL, msg.MediaType,
		msg.TemplateName, msg.TemplateLang, msg.TemplateComponents)
}

func (w *WhatsApp) SyncTemplates(ctx context.Context) ([]channel.Template, error) {
	return nil, channel.ErrUnsupported
}

func (w *WhatsApp) GetProvider(ch *model.ChannelWhatsApp) (Provider, error) {
	return ProviderForType(ch.Provider, w.httpClient)
}

func (w *WhatsApp) DecryptApiKey(ciphertext string) (string, error) {
	return w.cipher.Decrypt(ciphertext)
}

func (w *WhatsApp) DecryptVerifyToken(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return w.cipher.Decrypt(ciphertext)
}

func defaultHTTPClient() *http.Client {
	return &http.Client{}
}
