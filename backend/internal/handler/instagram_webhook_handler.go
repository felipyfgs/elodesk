package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"backend/internal/channel"
	igchan "backend/internal/channel/instagram"
	"backend/internal/channel/meta"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// InstagramWebhookHandler handles Instagram webhook verification, webhook
// delivery, and inbox provisioning.
type InstagramWebhookHandler struct {
	igRepo           *repo.ChannelInstagramRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	appSecret        string
	verifyToken      string
	dedup            *channel.DedupLock
	asynqClient      *asynq.Client
}

func NewInstagramWebhookHandler(
	igRepo *repo.ChannelInstagramRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	appSecret, verifyToken string,
) *InstagramWebhookHandler {
	return &InstagramWebhookHandler{
		igRepo:           igRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		appSecret:        appSecret,
		verifyToken:      verifyToken,
		dedup:            dedup,
		asynqClient:      asynqClient,
	}
}

// Verify handles GET /webhooks/instagram/:identifier (Meta hub.challenge handshake).
//
//	@Summary		Instagram webhook verification
//	@Tags			webhooks
//	@Produce		plain
//	@Param			identifier	path		string	true	"Instagram ID"
//	@Success		200			{string}	string	"hub.challenge"
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/instagram/{identifier} [get]
func (h *InstagramWebhookHandler) Verify(c *fiber.Ctx) error {
	return meta.HandleVerifyChallenge(c, h.verifyToken)
}

// Receive handles POST /webhooks/instagram/:identifier (inbound messages).
//
//	@Summary		Instagram webhook delivery
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			identifier	path		string	true	"Instagram ID"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/instagram/{identifier} [post]
func (h *InstagramWebhookHandler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	if !meta.VerifySignature(body, c.Get("X-Hub-Signature-256"), h.appSecret) {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	instagramID := c.Params("identifier")
	ch, err := h.igRepo.FindByInstagramID(c.Context(), instagramID)
	if err != nil {
		// Return 200 to Meta even when the channel isn't found to avoid retry storms
		return c.SendStatus(fiber.StatusOK)
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	if err := igchan.ProcessWebhook(
		c.Context(), body, inbox, ch.AccountID,
		h.dedup, h.asynqClient,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo,
	); err != nil {
		// Log but always ack to Meta
		_ = err
	}

	return c.SendStatus(fiber.StatusOK)
}

// Provision handles POST /api/v1/accounts/:aid/inboxes/instagram.
//
//	@Summary		Provision Instagram inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			aid		path		int							true	"Account ID"
//	@Param			body	body		dto.CreateInstagramInboxReq	true	"Provisioning request"
//	@Success		201		{object}	dto.APIResponse{data=dto.InstagramInboxResp}
//	@Failure		400		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/instagram [post]
func (h *InstagramWebhookHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateInstagramInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	tokenCiphertext, err := h.cipher.Encrypt(req.AccessToken)
	if err != nil {
		logger.Error().Str("component", "instagram-inbox").Err(err).Msg("failed to encrypt instagram token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt token"))
	}

	ch := &model.ChannelInstagram{
		AccountID:             accountID,
		InstagramID:           req.InstagramID,
		AccessTokenCiphertext: tokenCiphertext,
		ExpiresAt:             time.Now().Add(60 * 24 * time.Hour),
	}
	if err := h.igRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "instagram-inbox").Err(err).Msg("failed to create instagram channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create instagram channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindInstagram),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "instagram-inbox").Err(err).Msg("failed to create inbox for instagram")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.InstagramInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.InstagramChannelResp{
			ID:             ch.ID,
			InstagramID:    ch.InstagramID,
			ExpiresAt:      ch.ExpiresAt,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}
