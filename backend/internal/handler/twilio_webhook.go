package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/channel"
	twiliochan "backend/internal/channel/twilio"
	"backend/internal/config"
	appcrypto "backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type TwilioWebhookHandler struct {
	channelRepo      *repo.ChannelTwilioRepo
	inboxRepo        *repo.InboxRepo
	contactRepo      *repo.ContactRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	cipher           *appcrypto.Cipher
	dedup            *channel.DedupLock
	client           *twiliochan.Client
	twilioChan       *twiliochan.Channel
	cfg              *config.Config
	baseURL          string
}

func NewTwilioWebhookHandler(
	channelRepo *repo.ChannelTwilioRepo,
	inboxRepo *repo.InboxRepo,
	contactRepo *repo.ContactRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	cipher *appcrypto.Cipher,
	dedup *channel.DedupLock,
	client *twiliochan.Client,
	twilioChan *twiliochan.Channel,
	cfg *config.Config,
) *TwilioWebhookHandler {
	return &TwilioWebhookHandler{
		channelRepo:      channelRepo,
		inboxRepo:        inboxRepo,
		contactRepo:      contactRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		cipher:           cipher,
		dedup:            dedup,
		client:           client,
		twilioChan:       twilioChan,
		cfg:              cfg,
		baseURL:          cfg.APIURL,
	}
}

// Provision handles POST /api/v1/accounts/:aid/inboxes/twilio.
//
//	@Summary		Provision Twilio inbox (SMS or WhatsApp)
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			aid		path		int							true	"Account ID"
//	@Param			body	body		dto.CreateTwilioInboxReq	true	"Provisioning request"
//	@Success		201		{object}	dto.APIResponse{data=dto.TwilioInboxResp}
//	@Failure		400		{object}	dto.APIError
//	@Failure		403		{object}	dto.APIError
//	@Router			/api/v1/accounts/{aid}/inboxes/twilio [post]
func (h *TwilioWebhookHandler) Provision(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateTwilioInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	medium := model.TwilioMedium(req.Medium)
	if !h.mediumEnabled(medium) {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("feature_disabled", "twilio medium disabled"))
	}

	if req.PhoneNumber == "" && req.MessagingServiceSID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("missing_identifier", "phoneNumber or messagingServiceSid is required"))
	}
	if req.PhoneNumber != "" && req.MessagingServiceSID != "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("conflicting_identifiers", "phoneNumber and messagingServiceSid are mutually exclusive"))
	}

	if err := h.client.ValidateAccount(c.Context(), req.AccountSID, req.APIKeySID, req.AuthToken); err != nil {
		if twiliochan.IsAuthError(err) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_credentials", "twilio credentials rejected"))
		}
		logger.Warn().Str("component", "channel.twilio").Err(err).Msg("twilio credential validation failed")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("invalid_credentials", err.Error()))
	}

	authTokenCipher, err := h.cipher.Encrypt(req.AuthToken)
	if err != nil {
		logger.Error().Str("component", "channel.twilio").Err(err).Msg("encrypt auth token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt auth token"))
	}

	identifier, err := generateTwilioIdentifier()
	if err != nil {
		logger.Error().Str("component", "channel.twilio").Err(err).Msg("generate webhook identifier")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to generate webhook identifier"))
	}

	ch := &model.ChannelTwilio{
		AccountID:           accountID,
		Medium:              medium,
		AccountSID:          req.AccountSID,
		AuthTokenCiphertext: authTokenCipher,
		WebhookIdentifier:   identifier,
	}
	if req.APIKeySID != "" {
		v := req.APIKeySID
		ch.APIKeySID = &v
	}
	if req.PhoneNumber != "" {
		v := req.PhoneNumber
		ch.PhoneNumber = &v
	}
	if req.MessagingServiceSID != "" {
		v := req.MessagingServiceSID
		ch.MessagingServiceSID = &v
	}
	if err := h.channelRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "channel.twilio").Err(err).Msg("create channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindTwilio),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "channel.twilio").Err(err).Msg("create inbox")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	webhookBase := fmt.Sprintf("%s/webhooks/twilio/%s", h.baseURL, identifier)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.TwilioInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TwilioChannelResp{
			ID:                          ch.ID,
			Medium:                      string(ch.Medium),
			AccountSID:                  ch.AccountSID,
			APIKeySID:                   ch.APIKeySID,
			PhoneNumber:                 ch.PhoneNumber,
			MessagingServiceSID:         ch.MessagingServiceSID,
			WebhookIdentifier:           ch.WebhookIdentifier,
			ContentTemplatesLastUpdated: ch.ContentTemplatesLastUpdated,
			RequiresReauth:              ch.RequiresReauth,
			CreatedAt:                   ch.CreatedAt,
			UpdatedAt:                   ch.UpdatedAt,
		},
		WebhookURLs: &dto.TwilioWebhookURLs{
			Primary: webhookBase,
			Status:  webhookBase + "/status",
		},
	}))
}

// Receive handles POST /webhooks/twilio/:identifier.
//
//	@Summary		Twilio webhook delivery
//	@Tags			webhooks
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			identifier	path		string	true	"Webhook identifier"
//	@Success		200			{string}	string	"OK"
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/twilio/{identifier} [post]
func (h *TwilioWebhookHandler) Receive(c *fiber.Ctx) error {
	ch, ok := h.verifySignatureAndLoad(c, "twilio_webhook")
	if !ok {
		return nil
	}
	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), ch.ID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	form := formToValues(c)
	params := twiliochan.ParseInbound(form)

	ingester := twiliochan.NewIngester(h.channelRepo, h.inboxRepo, h.contactRepo, h.contactInboxRepo, h.conversationRepo, h.messageRepo, h.dedup)
	if err := ingester.Ingest(c.Context(), ch, inbox, params); err != nil {
		logger.Warn().Str("component", "channel.twilio").Err(err).Msg("ingest failed")
	}
	return c.SendStatus(fiber.StatusOK)
}

// Status handles POST /webhooks/twilio/:identifier/status.
//
//	@Summary		Twilio status callback
//	@Tags			webhooks
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			identifier	path		string	true	"Webhook identifier"
//	@Success		200			{string}	string	"OK"
//	@Failure		401			{object}	dto.APIError
//	@Router			/webhooks/twilio/{identifier}/status [post]
func (h *TwilioWebhookHandler) Status(c *fiber.Ctx) error {
	ch, ok := h.verifySignatureAndLoad(c, "twilio_status")
	if !ok {
		return nil
	}

	form := formToValues(c)
	sid := form.Get("MessageSid")
	status := form.Get("MessageStatus")
	errCode := form.Get("ErrorCode")
	if sid == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	msg, err := h.messageRepo.FindBySourceID(c.Context(), sid, ch.AccountID)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	var extErr *string
	if errCode != "" {
		extErr = &errCode
	}
	if _, err := h.messageRepo.UpdateStatus(c.Context(), msg.ID, ch.AccountID, status, extErr); err != nil {
		logger.Warn().Str("component", "channel.twilio").Err(err).Msg("update status")
	}
	return c.SendStatus(fiber.StatusOK)
}

// SyncTemplates handles POST /api/v1/accounts/:aid/inboxes/:id/twilio_templates.
//
//	@Summary		Sync Twilio WhatsApp content templates
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Param			aid	path		int	true	"Account ID"
//	@Param			id	path		int	true	"Inbox ID"
//	@Success		200	{object}	dto.APIResponse{data=dto.SyncTwilioTemplatesResp}
//	@Router			/api/v1/accounts/{aid}/inboxes/{id}/twilio_templates [post]
func (h *TwilioWebhookHandler) SyncTemplates(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindTwilio) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twilio channel"))
	}

	ch, err := h.channelRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	templates, err := h.twilioChan.SyncTemplatesForChannel(c.Context(), ch)
	if err != nil {
		logger.Warn().Str("component", "channel.twilio").Err(err).Msg("sync templates")
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResp("sync_failed", err.Error()))
	}

	refreshed, err := h.channelRepo.FindByID(c.Context(), ch.ID, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "reload channel"))
	}
	syncedAt := refreshed.UpdatedAt
	if refreshed.ContentTemplatesLastUpdated != nil {
		syncedAt = *refreshed.ContentTemplatesLastUpdated
	}
	return c.JSON(dto.SuccessResp(dto.SyncTwilioTemplatesResp{
		Count:    len(templates),
		SyncedAt: syncedAt,
	}))
}

// Delete handles DELETE /api/v1/accounts/:aid/inboxes/:id/twilio.
//
//	@Summary		Delete Twilio inbox
//	@Tags			inboxes
//	@Security		BearerAuth
//	@Param			aid	path	int	true	"Account ID"
//	@Param			id	path	int	true	"Inbox ID"
//	@Success		200	{object}	dto.APIResponse
//	@Router			/api/v1/accounts/{aid}/inboxes/{id}/twilio [delete]
// GetByInboxID handles GET /api/v1/accounts/:aid/inboxes/:id/twilio.
func (h *TwilioWebhookHandler) GetByInboxID(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindTwilio) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twilio channel"))
	}

	ch, err := h.channelRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.TwilioInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.TwilioChannelResp{
			ID:                          ch.ID,
			Medium:                      string(ch.Medium),
			AccountSID:                  ch.AccountSID,
			APIKeySID:                   ch.APIKeySID,
			PhoneNumber:                 ch.PhoneNumber,
			MessagingServiceSID:         ch.MessagingServiceSID,
			WebhookIdentifier:           ch.WebhookIdentifier,
			ContentTemplatesLastUpdated: ch.ContentTemplatesLastUpdated,
			RequiresReauth:            ch.RequiresReauth,
			CreatedAt:                   ch.CreatedAt,
			UpdatedAt:                   ch.UpdatedAt,
		},
	}))
}

// Update handles PUT /api/v1/accounts/:aid/inboxes/:id/twilio.
func (h *TwilioWebhookHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	inboxID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateTwilioInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), inboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if inbox.ChannelType != string(channel.KindTwilio) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twilio channel"))
	}

	if req.Name != "" {
		if err := h.inboxRepo.UpdateName(c.Context(), inboxID, accountID, req.Name); err != nil {
			return handleNotFound(c, err)
		}
	}

	if req.AuthToken != "" {
		authCipher, err := h.cipher.Encrypt(req.AuthToken)
		if err != nil {
			logger.Error().Str("component", "channel.twilio").Err(err).Msg("encrypt auth token")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt auth token"))
		}
		if err := h.channelRepo.UpdateAuthToken(c.Context(), inbox.ChannelID, authCipher); err != nil {
			return handleNotFound(c, err)
		}
	}

	return h.GetByInboxID(c)
}

func (h *TwilioWebhookHandler) Delete(c *fiber.Ctx) error {
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
	if inbox.ChannelType != string(channel.KindTwilio) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "inbox is not a twilio channel"))
	}
	if err := h.channelRepo.Delete(c.Context(), inbox.ChannelID); err != nil {
		logger.Error().Str("component", "channel.twilio").Err(err).Msg("delete channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete channel"))
	}
	return c.JSON(dto.SuccessResp(nil))
}

func (h *TwilioWebhookHandler) verifySignatureAndLoad(c *fiber.Ctx, component string) (*model.ChannelTwilio, bool) {
	identifier := c.Params("identifier")
	ch, err := h.channelRepo.FindByWebhookIdentifier(c.Context(), identifier)
	if err != nil {
		_ = c.SendStatus(fiber.StatusOK)
		return nil, false
	}

	authToken, err := h.cipher.Decrypt(ch.AuthTokenCiphertext)
	if err != nil {
		logger.Error().Str("component", component).Err(err).Msg("decrypt auth token")
		_ = c.SendStatus(fiber.StatusOK)
		return nil, false
	}

	form := formToValues(c)
	proto := c.Protocol()
	if fp := c.Get("X-Forwarded-Proto"); fp != "" {
		proto = fp
	}
	host := c.Hostname()
	if fh := c.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	fullURL := fmt.Sprintf("%s://%s%s", proto, host, c.OriginalURL())
	signature := c.Get(twiliochan.HeaderSignature)
	if !twiliochan.VerifySignature(authToken, fullURL, form, signature) {
		logger.Warn().Str("component", component).Str("identifier", identifier).Msg("invalid signature")
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid signature"))
		return nil, false
	}
	return ch, true
}

func (h *TwilioWebhookHandler) mediumEnabled(m model.TwilioMedium) bool {
	switch m {
	case model.TwilioMediumWhatsApp:
		return h.cfg.FeatureChannelTwilioWhatsapp
	case model.TwilioMediumSMS:
		return h.cfg.FeatureTwilioSmsMedium
	}
	return false
}

func formToValues(c *fiber.Ctx) url.Values {
	form := make(url.Values)
	c.Request().PostArgs().VisitAll(func(k, v []byte) {
		form.Add(string(k), string(v))
	})
	return form
}

func generateTwilioIdentifier() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate twilio identifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

