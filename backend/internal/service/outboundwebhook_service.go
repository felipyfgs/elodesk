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

// AttachmentURLBuilder devolve a URL pública do attachment hospedada pelo
// próprio elodesk (padrão Chatwoot/ActiveStorage). O integrador (wzap, n8n,
// etc.) só precisa saber de uma hostname — a do elodesk — pra alcançar a
// mídia, sem nunca falar direto com MinIO/S3.
type AttachmentURLBuilder func(accountID, attachmentID int64) string

// ContactInboxLookup é o contrato mínimo que o webhook outbound precisa pra
// resolver o source_id do contato em uma inbox. Implementado por
// *repo.ContactInboxRepo. Definido aqui (não em /repo) pra evitar ciclo
// import e manter o serviço testável com fakes. accountID é exigido pelo
// guard multi-tenant — o caller já tem o accountID do channel/conversation.
type ContactInboxLookup interface {
	FindByID(ctx context.Context, id, accountID int64) (*model.ContactInbox, error)
}

type OutboundWebhookService struct {
	asynqClient      *asynq.Client
	cipher           *crypto.Cipher
	urlBuilder       AttachmentURLBuilder
	contactInboxRepo ContactInboxLookup
}

func NewOutboundWebhookService(asynqClient *asynq.Client, cipher *crypto.Cipher) *OutboundWebhookService {
	return &OutboundWebhookService{asynqClient: asynqClient, cipher: cipher}
}

// WithAttachmentURLBuilder injeta o builder de URL hospedada pelo elodesk.
// Sem isso o webhook outbound serializa attachments sem `dataUrl` e o
// integrador não consegue baixar a mídia.
func (s *OutboundWebhookService) WithAttachmentURLBuilder(b AttachmentURLBuilder) *OutboundWebhookService {
	s.urlBuilder = b
	return s
}

// WithContactInboxRepo injeta o repo usado pra resolver o source_id do
// contact_inbox e enriquecer o payload outbound. Sem isso, o integrador (wzap)
// não tem como descobrir pra qual destinatário enviar quando a conversa é
// recém-criada (forward para contato sem histórico) — wzap só consegue mapear
// elodesk_conv_id → chat_jid via mensagens incoming, então o source_id
// fornecido aqui é o único caminho pra entrega no primeiro envio.
func (s *OutboundWebhookService) WithContactInboxRepo(r ContactInboxLookup) *OutboundWebhookService {
	s.contactInboxRepo = r
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
		data, err := s.marshalConversation(ctx, conv)
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

// outboundContactInboxView é o subset de model.ContactInbox que o integrador
// precisa: id (referência) e sourceId (telefone/email/identifier do canal).
// Para um forward para contato sem histórico, este é o único caminho pelo
// qual o wzap descobre o JID de destino — antes desse enriquecimento, ele
// caía no FindChatJIDByElodeskConvID que retorna vazio nesse cenário.
type outboundContactInboxView struct {
	ID       int64  `json:"id,omitempty"`
	SourceID string `json:"sourceId"`
}

// outboundConversationView embeda model.Conversation e adiciona o
// contactInbox resolvido. encoding/json usa o ContactInbox embutido aqui
// (menor profundidade) em vez do que viesse de Conversation.
type outboundConversationView struct {
	*model.Conversation
	ContactInbox *outboundContactInboxView `json:"contactInbox,omitempty"`
}

// marshalConversation enriquece a conversa com o contact_inbox quando o repo
// está disponível. Falha em resolver é silenciosa — o webhook ainda sai com a
// conversa "magra" (comportamento anterior), só que sem o source_id de
// fallback. Mantém retrocompat para integradores que não dependem do campo.
func (s *OutboundWebhookService) marshalConversation(ctx context.Context, conv *model.Conversation) ([]byte, error) {
	view := outboundConversationView{Conversation: conv}
	if s.contactInboxRepo != nil && conv.ContactInboxID != nil && *conv.ContactInboxID > 0 {
		ci, err := s.contactInboxRepo.FindByID(ctx, *conv.ContactInboxID, conv.AccountID)
		if err == nil && ci != nil && ci.SourceID != "" {
			view.ContactInbox = &outboundContactInboxView{ID: ci.ID, SourceID: ci.SourceID}
		}
	}
	return json.Marshal(view)
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

func (s *OutboundWebhookService) marshalMessage(_ context.Context, msg *model.Message) ([]byte, error) {
	if len(msg.Attachments) == 0 {
		return json.Marshal(msg)
	}

	views := make([]outboundAttachmentView, len(msg.Attachments))
	for i, att := range msg.Attachments {
		views[i] = outboundAttachmentView{Attachment: att}
		if s.urlBuilder != nil && att.ID > 0 {
			views[i].DataURL = s.urlBuilder(att.AccountID, att.ID)
		}
	}

	return json.Marshal(outboundMessageView{Message: msg, Attachments: views})
}
