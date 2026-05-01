package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"

	appchannel "backend/internal/channel"
	wa "backend/internal/channel/whatsapp"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type WhatsAppWebhookHandler struct {
	svc       *wa.Service
	inboxRepo *repo.InboxRepo
	chWaRepo  *repo.ChannelWhatsAppRepo
}

func NewWhatsAppWebhookHandler(svc *wa.Service, inboxRepo *repo.InboxRepo, chWaRepo *repo.ChannelWhatsAppRepo) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{svc: svc, inboxRepo: inboxRepo, chWaRepo: chWaRepo}
}

func (h *WhatsAppWebhookHandler) HandleHandshake(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "identifier required"))
	}

	query := map[string]string{
		"hub.mode":         c.Query("hub.mode"),
		"hub.verify_token": c.Query("hub.verify_token"),
		"hub.challenge":    c.Query("hub.challenge"),
	}

	ch, err := h.findChannelByIdentifier(c.Context(), identifier)
	if err != nil {
		logger.Warn().Str("component", "channel.whatsapp").Str("identifier", identifier).Msg("channel not found for handshake")
		return c.Status(fiber.StatusUnauthorized).SendString("")
	}

	challenge, ok := h.svc.VerifyHandshake(c.Context(), ch, query)
	if !ok {
		logger.Warn().Str("component", "channel.whatsapp").Str("identifier", identifier).Msg("invalid handshake")
		return c.Status(fiber.StatusUnauthorized).SendString("")
	}

	c.Set("Content-Type", "text/plain")
	return c.Status(fiber.StatusOK).SendString(challenge)
}

func (h *WhatsAppWebhookHandler) HandleDelivery(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "identifier required"))
	}

	req := &appchannel.InboundRequest{
		Body: c.Body(),
		Headers: map[string]string{
			"Content-Type":        c.Get("Content-Type"),
			"X-Hub-Signature-256": c.Get("X-Hub-Signature-256"),
			"D360-API-KEY":        c.Get("D360-API-KEY"),
		},
		PathParams: map[string]string{
			"identifier": identifier,
		},
	}

	if err := h.svc.HandleInbound(c.Context(), identifier, req); err != nil {
		logger.Error().Str("component", "channel.whatsapp").Err(err).Str("identifier", identifier).Msg("handle inbound")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal error"))
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *WhatsAppWebhookHandler) findChannelByIdentifier(ctx context.Context, identifier string) (*model.ChannelWhatsApp, error) {
	inbox, err := h.inboxRepo.FindByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}
	ch, err := h.chWaRepo.FindByID(ctx, inbox.ChannelID, inbox.AccountID)
	if err != nil {
		return nil, err
	}
	return ch, nil
}
