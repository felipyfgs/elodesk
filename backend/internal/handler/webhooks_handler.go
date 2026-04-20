package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/audit"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type WebhookHandler struct {
	repo   *repo.OutboundWebhookRepo
	audit  *audit.Logger
	cipher *appcrypto.Cipher
}

func NewWebhookHandler(r *repo.OutboundWebhookRepo, auditLogger *audit.Logger, cipher *appcrypto.Cipher) *WebhookHandler {
	return &WebhookHandler{repo: r, audit: auditLogger, cipher: cipher}
}

func (h *WebhookHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	webhooks, err := h.repo.ListByAccount(c.Context(), accountID)
	if err != nil {
		logger.Error().Str("component", "webhooks").Err(err).Msg("failed to list webhooks")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list webhooks"))
	}
	return c.JSON(dto.SuccessResp(dto.WebhooksToResp(webhooks)))
}

func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	var req dto.CreateWebhookReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	secret, err := generateWebhookSecret()
	if err != nil {
		logger.Error().Str("component", "webhooks").Err(err).Msg("failed to generate secret")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate secret"))
	}

	encryptedSecret, err := h.cipher.Encrypt(secret)
	if err != nil {
		logger.Error().Str("component", "webhooks").Err(err).Msg("failed to encrypt secret")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt secret"))
	}

	m := &model.OutboundWebhook{
		AccountID:     accountID,
		URL:           req.URL,
		Subscriptions: string(req.Subscriptions),
		Secret:        encryptedSecret,
		IsActive:      true,
	}
	if err := h.repo.Create(c.Context(), m); err != nil {
		logger.Error().Str("component", "webhooks").Err(err).Msg("failed to create webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create webhook"))
	}

	h.audit.LogFromCtx(c, "webhook.configured", "outbound_webhook", &m.ID, fiber.Map{"url": m.URL})

	resp := dto.WebhookToResp(m)
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(fiber.Map{
		"webhook": resp,
		"secret":  secret,
	}))
}

func (h *WebhookHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid webhook id"))
	}

	current, err := h.repo.FindByID(c.Context(), id, accountID)
	if err != nil {
		if errors.Is(err, repo.ErrWebhookNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "webhook_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to find webhook"))
	}

	var req dto.UpdateWebhookReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	if req.URL != nil {
		current.URL = *req.URL
	}
	if req.Subscriptions != nil {
		current.Subscriptions = string(req.Subscriptions)
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	if err := h.repo.Update(c.Context(), current); err != nil {
		logger.Error().Str("component", "webhooks").Err(err).Msg("failed to update webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update webhook"))
	}
	return c.JSON(dto.SuccessResp(dto.WebhookToResp(current)))
}

func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid webhook id"))
	}
	if err := h.repo.Delete(c.Context(), id, accountID); err != nil {
		if errors.Is(err, repo.ErrWebhookNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "webhook_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete webhook"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
