package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"backend/internal/channel"
	fbchan "backend/internal/channel/facebook"
	"backend/internal/channel/meta"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// FacebookWebhookHandler handles Facebook Messenger webhook verification,
// webhook delivery, and inbox provisioning.
type FacebookWebhookHandler struct {
	fbRepo           *repo.ChannelFacebookRepo
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

func NewFacebookWebhookHandler(
	fbRepo *repo.ChannelFacebookRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	asynqClient *asynq.Client,
	appSecret, verifyToken string,
) *FacebookWebhookHandler {
	return &FacebookWebhookHandler{
		fbRepo:           fbRepo,
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

// Verify handles GET /webhooks/facebook/:identifier (Meta hub.challenge handshake).
//
//	@Summary		Facebook webhook verification
//	@Tags			webhooks
//	@Produce		plain
//	@Param			identifier	path		string	true	"Facebook Page ID"
//	@Success		200			{string}	string	"hub.challenge"
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/facebook/{identifier} [get]
func (h *FacebookWebhookHandler) Verify(c *fiber.Ctx) error {
	return meta.HandleVerifyChallenge(c, h.verifyToken)
}

// Receive handles POST /webhooks/facebook/:identifier (inbound messages).
//
//	@Summary		Facebook webhook delivery
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			identifier	path		string	true	"Facebook Page ID"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/facebook/{identifier} [post]
func (h *FacebookWebhookHandler) Receive(c *fiber.Ctx) error {
	body := c.Body()

	if !meta.VerifySignature(body, c.Get("X-Hub-Signature-256"), h.appSecret) {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
	}

	pageID := c.Params("identifier")
	ch, err := h.fbRepo.FindByPageID(c.Context(), pageID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	if err := fbchan.ProcessWebhook(
		c.Context(), body, inbox, ch.AccountID,
		h.dedup, h.asynqClient,
		h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo,
	); err != nil {
		_ = err
	}

	return c.SendStatus(fiber.StatusOK)
}

// Provision handles POST /api/v1/accounts/:aid/inboxes/facebook_page.
//
//	@Summary		Provision Facebook Page inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			aid		path		int							true	"Account ID"
//	@Param			body	body		dto.CreateFacebookInboxReq	true	"Provisioning request"
//	@Success		201		{object}	dto.APIResponse{data=dto.FacebookInboxResp}
//	@Failure		400		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/facebook_page [post]
func (h *FacebookWebhookHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateFacebookInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	pageTokenCiphertext, err := h.cipher.Encrypt(req.PageAccessToken)
	if err != nil {
		logger.Error().Str("component", "facebook-inbox").Err(err).Msg("failed to encrypt page token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt page token"))
	}

	var userTokenCiphertext *string
	if req.UserAccessToken != nil && *req.UserAccessToken != "" {
		ct, encErr := h.cipher.Encrypt(*req.UserAccessToken)
		if encErr != nil {
			logger.Error().Str("component", "facebook-inbox").Err(encErr).Msg("failed to encrypt user token")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt user token"))
		}
		userTokenCiphertext = &ct
	}

	ch := &model.ChannelFacebookPage{
		AccountID:                 accountID,
		PageID:                    req.PageID,
		PageAccessTokenCiphertext: pageTokenCiphertext,
		UserAccessTokenCiphertext: userTokenCiphertext,
		InstagramID:               req.InstagramID,
	}
	if err := h.fbRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "facebook-inbox").Err(err).Msg("failed to create facebook channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create facebook channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindFacebookPage),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "facebook-inbox").Err(err).Msg("failed to create inbox for facebook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.FacebookInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.FacebookChannelResp{
			ID:             ch.ID,
			PageID:         ch.PageID,
			InstagramID:    ch.InstagramID,
			RequiresReauth: ch.RequiresReauth,
			CreatedAt:      ch.CreatedAt,
			UpdatedAt:      ch.UpdatedAt,
		},
	}))
}
