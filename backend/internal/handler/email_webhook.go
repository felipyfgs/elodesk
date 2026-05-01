package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"

	emailch "backend/internal/channel/email"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// EmailWebhookHandler handles inbound email delivery from external relays.
type EmailWebhookHandler struct {
	channelEmailRepo *repo.ChannelEmailRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	attachHandler    *emailch.AttachmentHandler
}

func NewEmailWebhookHandler(
	channelEmailRepo *repo.ChannelEmailRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	attachHandler *emailch.AttachmentHandler,
) *EmailWebhookHandler {
	return &EmailWebhookHandler{
		channelEmailRepo: channelEmailRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		attachHandler:    attachHandler,
	}
}

// Inbound handles POST /webhooks/email/inbound
//
//	@Summary     Inbound email webhook
//	@Description Receives raw MIME email from a relay (SES, SendGrid, Postfix). Authenticated via HMAC.
//	@Tags        webhooks
//	@Param       X-Elodesk-Inbound-Signature header string true "HMAC-SHA256 of body"
//	@Param       X-Elodesk-Inbox-Id          header string true "Target inbox ID"
//	@Router      /webhooks/email/inbound [post]
func (h *EmailWebhookHandler) Inbound(c *fiber.Ctx) error {
	raw := c.Body()

	if err := validateInboundHMAC(c.Get("X-Elodesk-Inbound-Signature"), raw); err != nil {
		logger.Warn().Str("component", "email-webhook").Err(err).Str("ip", c.IP()).Msg("inbound email: invalid signature")
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	var inboxID int64
	if _, err := fmt.Sscanf(c.Get("X-Elodesk-Inbox-Id"), "%d", &inboxID); err != nil || inboxID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "missing or invalid X-Elodesk-Inbox-Id"))
	}

	ctx := context.Background()

	ch, err := h.channelEmailRepo.FindByInboxID(ctx, inboxID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "inbox not found"))
	}

	env, parseErr := emailch.ParseMIME(bytes.NewReader(raw))
	if parseErr != nil && env == nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Parse Error", parseErr.Error()))
	}

	deps := emailch.Deps{
		ConversationRepo: h.conversationRepo,
		MessageRepo:      h.messageRepo,
		ContactRepo:      h.contactRepo,
		ContactInboxRepo: h.contactInboxRepo,
		ConversationCreate: func(ctx2 context.Context, conv *model.Conversation) error {
			return h.conversationRepo.Create(ctx2, conv)
		},
	}
	finder := emailch.NewConversationFinder(deps, ch.AccountID, inboxID)
	conv, _, err := finder.Resolve(ctx, env)
	if err != nil {
		logger.Error().Str("component", "email-webhook").Err(err).Msg("thread resolve failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "thread resolve failed"))
	}

	var srcID *string
	if env.MessageID != "" {
		srcID = &env.MessageID
	}
	body := env.Text
	if body == "" {
		body = env.HTML
	}
	senderType := "Contact"
	contactID := conv.ContactID
	msg := &model.Message{
		AccountID:      ch.AccountID,
		InboxID:        inboxID,
		ConversationID: conv.ID,
		MessageType:    model.MessageIncoming,
		ContentType:    model.ContentTypeIncomingEmail,
		Content:        &body,
		SourceID:       srcID,
		Status:         model.MessageSent,
		SenderType:     &senderType,
		SenderID:       &contactID,
	}
	created, err := h.messageRepo.Create(ctx, msg)
	if err != nil {
		logger.Error().Str("component", "email-webhook").Err(err).Msg("create message failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "create message"))
	}

	if h.attachHandler != nil && len(env.Attachments) > 0 {
		if err := h.attachHandler.ProcessAttachments(ctx, created, env.Attachments); err != nil {
			logger.Warn().Str("component", "email-webhook").Err(err).Int64("messageID", created.ID).Msg("attachment processing partial failure")
		}
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(fiber.Map{"messageId": created.ID}))
}

func validateInboundHMAC(sig string, body []byte) error {
	secret := os.Getenv("INBOUND_EMAIL_SECRET")
	if secret == "" {
		return nil
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("hmac mismatch")
	}
	return nil
}
