package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"backend/internal/channel"
	linechan "backend/internal/channel/line"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type LineWebhookHandler struct {
	lineRepo         *repo.ChannelLineRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
	api              *linechan.APIClient
}

func NewLineWebhookHandler(
	lineRepo *repo.ChannelLineRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	api *linechan.APIClient,
) *LineWebhookHandler {
	return &LineWebhookHandler{
		lineRepo:         lineRepo,
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

// Receive handles POST /webhooks/line/:line_channel_id (inbound messages).
//
//	@Summary		LINE webhook delivery
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			line_channel_id	path		string	true	"LINE channel id"
//	@Success		200				{object}	dto.APIResponse
//	@Failure		401				{object}	dto.APIError
//	@Router			/webhooks/line/{line_channel_id} [post]
func (h *LineWebhookHandler) Receive(c *fiber.Ctx) error {
	lineChannelID := c.Params("line_channel_id")

	ch, err := h.lineRepo.FindByLineChannelID(c.Context(), lineChannelID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	secret, err := h.cipher.Decrypt(ch.LineChannelSecretCiphertext)
	if err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to decrypt line channel secret")
		return c.SendStatus(fiber.StatusOK)
	}

	signature := c.Get("X-Line-Signature")
	body := c.Body()
	if !linechan.VerifySignature(secret, body, signature) {
		logger.Warn().Str("component", "channel.line").Str("lineChannelId", lineChannelID).Msg("invalid signature")
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	token, err := h.cipher.Decrypt(ch.LineChannelTokenCiphertext)
	if err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to decrypt line channel token")
		return c.SendStatus(fiber.StatusOK)
	}

	if err := linechan.ProcessWebhook(c.Context(), body, ch, inbox, h.dedup, h.api, token,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo); err != nil {
		logger.Warn().Str("component", "channel.line").Err(err).Msg("line process webhook error")
	}
	return c.SendStatus(fiber.StatusOK)
}

// Provision handles POST /api/v1/accounts/:aid/inboxes/line.
//
//	@Summary		Provision LINE inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			aid		path		int							true	"Account ID"
//	@Param			body	body		dto.CreateLineInboxReq		true	"Provisioning request"
//	@Success		201		{object}	dto.APIResponse{data=dto.LineInboxResp}
//	@Failure		400		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/line [post]
func (h *LineWebhookHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateLineInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	ctx := c.Context()

	info, err := h.api.GetBotInfo(ctx, req.LineChannelToken)
	if err != nil {
		logger.Warn().Str("component", "channel.line").Err(err).Msg("line bot info validation failed")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_line_token", "failed to validate LINE channel token"))
	}

	secretCiphertext, err := h.cipher.Encrypt(req.LineChannelSecret)
	if err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to encrypt line secret")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt line secret"))
	}
	tokenCiphertext, err := h.cipher.Encrypt(req.LineChannelToken)
	if err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to encrypt line token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt line token"))
	}

	botBasicID := info.BasicID
	botDisplayName := info.DisplayName

	ch := &model.ChannelLine{
		AccountID:                   accountID,
		LineChannelID:               req.LineChannelID,
		LineChannelSecretCiphertext: secretCiphertext,
		LineChannelTokenCiphertext:  tokenCiphertext,
		BotBasicID:                  &botBasicID,
		BotDisplayName:              &botDisplayName,
	}
	if err := h.lineRepo.Create(ctx, ch); err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to create line channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create line channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindLine),
	}
	if err := h.inboxRepo.Create(ctx, inbox); err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to create line inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.LineInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.LineChannelResp{
			ID:             ch.ID,
			LineChannelID:  ch.LineChannelID,
			BotBasicID:     ch.BotBasicID,
			BotDisplayName: ch.BotDisplayName,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}

// Delete handles DELETE /api/v1/accounts/:aid/inboxes/:id/line.
//
//	@Summary		Delete LINE inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Param			aid		path		int		true	"Account ID"
//	@Param			id		path		int		true	"Inbox ID"
//	@Success		200		{object}	dto.APIResponse
//	@Failure		404		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/{id}/line [delete]
// GetByInboxID handles GET /api/v1/accounts/:aid/inboxes/:id/line.
func (h *LineWebhookHandler) GetByInboxID(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindLine) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a line channel"))
	}

	ch, err := h.lineRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.LineInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.LineChannelResp{
			ID:             ch.ID,
			LineChannelID:  ch.LineChannelID,
			BotBasicID:     ch.BotBasicID,
			BotDisplayName: ch.BotDisplayName,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}

// Update handles PUT /api/v1/accounts/:aid/inboxes/:id/line.
func (h *LineWebhookHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateLineInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if inbox.ChannelType != string(channel.KindLine) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a line channel"))
	}

	if req.Name != "" {
		if err := h.inboxRepo.UpdateName(c.Context(), inboxID, accountID, req.Name); err != nil {
			return handleNotFound(c, err)
		}
	}

	if req.LineChannelSecret != "" || req.LineChannelToken != "" {
		ch, err := h.lineRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
		if err != nil {
			return handleNotFound(c, err)
		}

		secretCipher := ch.LineChannelSecretCiphertext
		if req.LineChannelSecret != "" {
			secretCipher, err = h.cipher.Encrypt(req.LineChannelSecret)
			if err != nil {
				logger.Error().Str("component", "channel.line").Err(err).Msg("failed to encrypt line secret")
				return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt line secret"))
			}
		}
		tokenCipher := ch.LineChannelTokenCiphertext
		if req.LineChannelToken != "" {
			tokenCipher, err = h.cipher.Encrypt(req.LineChannelToken)
			if err != nil {
				logger.Error().Str("component", "channel.line").Err(err).Msg("failed to encrypt line token")
				return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt line token"))
			}
		}
		if err := h.lineRepo.UpdateCredentials(c.Context(), ch.ID, secretCipher, tokenCipher); err != nil {
			return handleNotFound(c, err)
		}
	}

	return h.GetByInboxID(c)
}

func (h *LineWebhookHandler) Delete(c *fiber.Ctx) error {
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

	if inbox.ChannelType != string(channel.KindLine) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a line channel"))
	}

	if err := h.lineRepo.Delete(c.Context(), inbox.ChannelID); err != nil {
		logger.Error().Str("component", "channel.line").Err(err).Msg("failed to delete line channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete line channel"))
	}

	return c.JSON(dto.SuccessResp(nil))
}
