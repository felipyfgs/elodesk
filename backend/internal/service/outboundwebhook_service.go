package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"backend/internal/crypto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/webhook"
)

const (
	EventTypeMessageCreated            = "message_created"
	EventTypeMessageUpdated            = "message_updated"
	EventTypeConversationStatusChanged = "conversation_status_changed"
	EventTypeConversationUpdated       = "conversation_updated"
	EventTypeConversationCreated       = "conversation_created"
	EventTypeConversationTypingOn       = "conversation_typing_on"
	EventTypeConversationTypingOff      = "conversation_typing_off"
)

// AttachmentPresigner gera uma URL temporária (presigned GET) para o object
// path do attachment. É usado pelo dispatch do webhook outbound para que
// integradores externos consigam baixar a mídia sem credenciais do MinIO.
type AttachmentPresigner func(ctx context.Context, fileKey string) (string, error)

type OutboundWebhookService struct {
	asynqClient *asynq.Client
	cipher      *crypto.Cipher
	presigner   AttachmentPresigner
}

func NewOutboundWebhookService(asynqClient *asynq.Client, cipher *crypto.Cipher) *OutboundWebhookService {
	return &OutboundWebhookService{asynqClient: asynqClient, cipher: cipher}
}

// WithAttachmentPresigner injeta o gerador de URL presigned do MinIO. Sem
// isso o webhook outbound serializa attachments sem `dataUrl`, e o integrador
// não consegue baixar a mídia.
func (s *OutboundWebhookService) WithAttachmentPresigner(p AttachmentPresigner) *OutboundWebhookService {
	s.presigner = p
	return s
}

func (s *OutboundWebhookService) DispatchMessageCreated(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation, msg *model.Message) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeMessageCreated, conv, msg, nil)
}

func (s *OutboundWebhookService) DispatchMessageUpdated(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation, msg *model.Message) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeMessageUpdated, conv, msg, nil)
}

func (s *OutboundWebhookService) DispatchConversationStatusChanged(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeConversationStatusChanged, conv, nil, nil)
}

func (s *OutboundWebhookService) DispatchConversationUpdated(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation, attributes json.RawMessage) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeConversationUpdated, conv, nil, attributes)
}

func (s *OutboundWebhookService) DispatchConversationCreated(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation) error {
	return s.dispatch(ctx, ch, inboxID, EventTypeConversationCreated, conv, nil, nil)
}

func (s *OutboundWebhookService) DispatchTypingEvent(ctx context.Context, ch *model.ChannelAPI, inboxID int64, conv *model.Conversation, eventName string) error {
	return s.dispatch(ctx, ch, inboxID, eventName, conv, nil, nil)
}

func (s *OutboundWebhookService) dispatch(ctx context.Context, ch *model.ChannelAPI, inboxID int64, eventType string, conv *model.Conversation, msg *model.Message, convAttrs json.RawMessage) error {
	// DeliveryID is generated HERE (once per delivery) and stored in the task
	// payload so it survives retries. The processor never regenerates it.
	// Encripta o secret antes de enfileirar — plaintext nunca toca o Redis.
	// Falha aqui aborta o dispatch: enfileirar com Secret vazio só geraria
	// dead-letter no processor (branch "secret is empty"), perdendo o evento
	// silenciosamente para o integrador.
	secretCiphertext := ""
	if ch.Secret != "" {
		enc, err := s.cipher.Encrypt(ch.Secret)
		if err != nil {
			logger.Error().Str("component", "outbound-webhook").
				Str("eventType", eventType).
				Int64("accountId", ch.AccountID).
				Err(err).Msg("failed to encrypt webhook secret, aborting dispatch")
			return fmt.Errorf("encrypt webhook secret: %w", err)
		}
		secretCiphertext = enc
	}

	payload := &webhook.OutboundPayload{
		EventType:              eventType,
		AccountID:              ch.AccountID,
		InboxID:                inboxID,
		WebhookURL:             ch.WebhookURL,
		Secret:                 secretCiphertext,
		HmacCiphertext:         ch.HmacToken,
		DeliveryID:             uuid.NewString(),
		ConversationAttributes: convAttrs,
	}

	if conv != nil {
		data, err := json.Marshal(conv)
		if err != nil {
			return fmt.Errorf("marshal conversation: %w", err)
		}
		payload.Conversation = data
	}
	if msg != nil {
		data, err := s.marshalMessage(ctx, msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}
		payload.Message = data
	}

	task, err := webhook.NewOutboundTask(payload)
	if err != nil {
		return fmt.Errorf("create outbound task: %w", err)
	}

	info, err := s.asynqClient.EnqueueContext(ctx, task)
	if err != nil {
		logger.Error().Str("component", "outbound-webhook").Err(err).
			Str("eventType", eventType).
			Int64("accountId", ch.AccountID).
			Str("webhookUrl", ch.WebhookURL).
			Msg("enqueue failed")
		return fmt.Errorf("enqueue outbound webhook: %w", err)
	}

	logger.Info().Str("component", "outbound-webhook").
		Str("eventType", eventType).
		Int64("accountId", ch.AccountID).
		Str("webhookUrl", ch.WebhookURL).
		Str("deliveryId", payload.DeliveryID).
		Str("taskId", info.ID).
		Str("queue", info.Queue).
		Msg("enqueued")

	return nil
}

// outboundAttachmentView espelha model.Attachment com `dataUrl` adicional
// (URL presigned do MinIO). Vai serializado no payload pra que integradores
// externos consigam baixar a mídia sem credenciais.
type outboundAttachmentView struct {
	model.Attachment
	DataURL string `json:"dataUrl,omitempty"`
}

// outboundMessageView reusa todos os campos de model.Message via embed e
// shadowa Attachments com a versão enriquecida. encoding/json escolhe o
// campo de menor profundidade quando há colisão de tag — então este
// Attachments sobrepõe o promovido.
type outboundMessageView struct {
	*model.Message
	Attachments []outboundAttachmentView `json:"attachments"`
}

func (s *OutboundWebhookService) marshalMessage(ctx context.Context, msg *model.Message) ([]byte, error) {
	if len(msg.Attachments) == 0 {
		return json.Marshal(msg)
	}

	views := make([]outboundAttachmentView, len(msg.Attachments))
	for i, att := range msg.Attachments {
		views[i] = outboundAttachmentView{Attachment: att}
		if s.presigner == nil || att.FileKey == nil || *att.FileKey == "" {
			continue
		}
		signedURL, err := s.presigner(ctx, *att.FileKey)
		if err != nil {
			logger.Warn().Str("component", "outbound-webhook").
				Int64("attachmentId", att.ID).Err(err).
				Msg("failed to presign attachment URL — dataUrl will be empty")
			continue
		}
		views[i].DataURL = signedURL
	}

	return json.Marshal(outboundMessageView{Message: msg, Attachments: views})
}
