package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"backend/internal/channel"
	tgchan "backend/internal/channel/telegram"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type TelegramWebhookHandler struct {
	tgRepo           *repo.ChannelTelegramRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
	api              *tgchan.APIClient
}

func NewTelegramWebhookHandler(
	tgRepo *repo.ChannelTelegramRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	api *tgchan.APIClient,
) *TelegramWebhookHandler {
	return &TelegramWebhookHandler{
		tgRepo:           tgRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            dedup,
		asynqClient:      asynqClient,
		api:              api,
	}
}

// Receive handles POST /webhooks/telegram/:identifier (inbound messages).
//
//	@Summary		Telegram webhook delivery
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			identifier	path		string	true	"Webhook identifier"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/telegram/{identifier} [post]
func (h *TelegramWebhookHandler) Receive(c *fiber.Ctx) error {
	identifier := c.Params("identifier")

	ch, err := h.tgRepo.FindByWebhookIdentifier(c.Context(), identifier)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	secretToken, err := h.cipher.Decrypt(ch.SecretTokenCiphertext)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	providedSecret := c.Get("X-Telegram-Bot-Api-Secret-Token")
	if providedSecret == "" || providedSecret != secretToken {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid or missing secret token"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	if err := tgchan.ProcessWebhook(
		c.Context(), c.Body(), inbox, ch.AccountID,
		h.dedup, h.asynqClient,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo,
	); err != nil {
		_ = err
	}

	return c.SendStatus(fiber.StatusOK)
}

// Provision handles POST /api/v1/accounts/:aid/inboxes/telegram.
//
//	@Summary		Provision Telegram inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			aid		path		int							true	"Account ID"
//	@Param			body	body		dto.CreateTelegramInboxReq	true	"Provisioning request"
//	@Success		201		{object}	dto.APIResponse{data=dto.TelegramInboxResp}
//	@Failure		400		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/telegram [post]
func (h *TelegramWebhookHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateTelegramInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	ctx := c.Context()

	me, err := h.api.GetMe(ctx, req.BotToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_bot_token", "failed to validate bot token with Telegram"))
	}

	secretToken, err := generateOpaqueToken(32)
	if err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to generate secret token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate secret token"))
	}

	webhookIdentifier, err := generateOpaqueToken(16)
	if err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to generate webhook identifier")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate webhook identifier"))
	}

	webhookURL := buildWebhookURL(c, webhookIdentifier)
	if err := h.api.SetWebhook(ctx, req.BotToken, webhookURL, secretToken); err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to register telegram webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", fmt.Sprintf("failed to register webhook: %s", err.Error())))
	}

	tokenCiphertext, err := h.cipher.Encrypt(req.BotToken)
	if err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to encrypt bot token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt bot token"))
	}

	secretCiphertext, err := h.cipher.Encrypt(secretToken)
	if err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to encrypt secret token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt secret token"))
	}

	botName := me.Username
	if botName == "" {
		botName = me.FirstName
	}

	ch := &model.ChannelTelegram{
		AccountID:             accountID,
		BotTokenCiphertext:    tokenCiphertext,
		BotName:               &botName,
		WebhookIdentifier:     webhookIdentifier,
		SecretTokenCiphertext: secretCiphertext,
	}
	if err := h.tgRepo.Create(ctx, ch); err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to create telegram channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create telegram channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindTelegram),
	}
	if err := h.inboxRepo.Create(ctx, inbox); err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to create inbox for telegram")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.TelegramInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TelegramChannelResp{
			ID:                ch.ID,
			BotName:           ch.BotName,
			WebhookIdentifier: ch.WebhookIdentifier,
			RequiresReauth:    ch.RequiresReauth,
			CreatedAt:         ch.CreatedAt,
			UpdatedAt:         ch.UpdatedAt,
		},
	}))
}

// Delete handles DELETE /api/v1/accounts/:aid/inboxes/:id/telegram.
//
//	@Summary		Delete Telegram inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Param			aid		path		int		true	"Account ID"
//	@Param			id		path		int		true	"Inbox ID"
//	@Success		200		{object}	dto.APIResponse
//	@Failure		404		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/{id}/telegram [delete]
func (h *TelegramWebhookHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if inbox.ChannelType != string(channel.KindTelegram) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a telegram channel"))
	}

	ch, err := h.tgRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	botToken, err := h.cipher.Decrypt(ch.BotTokenCiphertext)
	if err == nil {
		if delErr := h.api.DeleteWebhook(c.Context(), botToken); delErr != nil {
			_ = delErr
		}
	}

	if err := h.tgRepo.Delete(c.Context(), ch.ID); err != nil {
		logger.Error().Str("component", "telegram-inbox").Err(err).Msg("failed to delete telegram channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete telegram channel"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

func generateOpaqueToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func buildWebhookURL(c *fiber.Ctx, identifier string) string {
	scheme := "https"
	if proto := c.Get("X-Forwarded-Proto"); proto == "http" {
		scheme = "http"
	}
	host := c.Hostname()
	return fmt.Sprintf("%s://%s/webhooks/telegram/%s", scheme, host, identifier)
}
