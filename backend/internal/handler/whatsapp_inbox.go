package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/channel"
	wa "backend/internal/channel/whatsapp"
	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

type WhatsAppInboxHandler struct {
	inboxRepo           *repo.InboxRepo
	channelWhatsappRepo *repo.ChannelWhatsAppRepo
	cipher              *crypto.Cipher
	waSvc               *wa.Service
}

func NewWhatsAppInboxHandler(
	inboxRepo *repo.InboxRepo,
	channelWhatsappRepo *repo.ChannelWhatsAppRepo,
	cipher *crypto.Cipher,
	waSvc *wa.Service,
) *WhatsAppInboxHandler {
	return &WhatsAppInboxHandler{
		inboxRepo:           inboxRepo,
		channelWhatsappRepo: channelWhatsappRepo,
		cipher:              cipher,
		waSvc:               waSvc,
	}
}

func (h *WhatsAppInboxHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateWhatsAppInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	if err := validateCreateWhatsAppInboxReq(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", err.Error()))
	}

	secretToEncrypt := req.ApiKey

	apiKeyCiphertext, err := h.cipher.Encrypt(secretToEncrypt)
	if err != nil {
		logger.Error().Str("component", "whatsapp-inbox").Err(err).Msg("failed to encrypt api key")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt api key"))
	}

	ch := &model.ChannelWhatsApp{
		AccountID:        accountID,
		Provider:         req.Provider,
		PhoneNumber:      req.PhoneNumber,
		ApiKeyCiphertext: apiKeyCiphertext,
	}

	switch req.Provider {
	case "whatsapp_cloud":
		verifyToken := generateWebhookVerifyToken()
		vtCiphertext, err := h.cipher.Encrypt(verifyToken)
		if err != nil {
			logger.Error().Str("component", "whatsapp-inbox").Err(err).Msg("failed to encrypt verify token")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt verify token"))
		}
		ch.WebhookVerifyTokenCiphertext = &vtCiphertext
		ch.PhoneNumberID = &req.PhoneNumberID
		ch.BusinessAccountID = &req.BusinessAccountID
	}

	if err := h.channelWhatsappRepo.Create(c.Context(), ch); err != nil {
		logger.Error().Str("component", "whatsapp-inbox").Err(err).Msg("failed to create whatsapp channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create whatsapp channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   ch.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindWhatsapp),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "whatsapp-inbox").Err(err).Msg("failed to create inbox for whatsapp")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	resp := dto.CreateWhatsAppInboxResp{
		InboxID:           inbox.ID,
		AccountID:         accountID,
		ChannelID:         ch.ID,
		Name:              req.Name,
		ChannelType:       string(channel.KindWhatsapp),
		Provider:          ch.Provider,
		PhoneNumber:       ch.PhoneNumber,
		PhoneNumberID:     req.PhoneNumberID,
		BusinessAccountID: req.BusinessAccountID,
		CreatedAt:         inbox.CreatedAt,
	}

	resp.ApiKey = req.ApiKey

	if ch.WebhookVerifyTokenCiphertext != nil {
		vt, _ := h.cipher.Decrypt(*ch.WebhookVerifyTokenCiphertext)
		resp.WebhookVerifyToken = vt
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(resp))
}

func (h *WhatsAppInboxHandler) SyncTemplates(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	ch, err := h.channelWhatsappRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	templates, err := h.waSvc.SyncTemplatesForChannel(c.Context(), ch)
	if err != nil {
		logger.Error().Str("component", "whatsapp-inbox").Err(err).Msg("failed to sync templates")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to sync templates"))
	}

	now := ch.MessageTemplatesSyncedAt
	if now == nil {
		t := ch.UpdatedAt
		now = &t
	}

	templateResps := make([]dto.TemplateResp, len(templates))
	for i, t := range templates {
		templateResps[i] = dto.TemplateResp{
			Name:     t.Name,
			Language: t.Language,
			Status:   t.Status,
		}
	}

	return c.JSON(dto.SuccessResp(dto.SyncTemplatesResp{
		Templates: templateResps,
		SyncedAt:  *now,
	}))
}

func (h *WhatsAppInboxHandler) GetByID(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if inbox.ChannelType != string(channel.KindWhatsapp) {
		return handleNotFound(c, err)
	}

	ch, err := h.channelWhatsappRepo.FindByID(c.Context(), inbox.ChannelID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	resp := dto.WhatsAppInboxResp{
		ID:                       inbox.ID,
		AccountID:                accountID,
		ChannelID:                inbox.ChannelID,
		Name:                     inbox.Name,
		ChannelType:              inbox.ChannelType,
		Provider:                 ch.Provider,
		PhoneNumber:              ch.PhoneNumber,
		PhoneNumberID:            ch.PhoneNumberID,
		BusinessAccountID:        ch.BusinessAccountID,
		MessageTemplatesSyncedAt: ch.MessageTemplatesSyncedAt,
		CreatedAt:                inbox.CreatedAt,
	}

	return c.JSON(dto.SuccessResp(resp))
}

func validateCreateWhatsAppInboxReq(req dto.CreateWhatsAppInboxReq) error {
	if req.PhoneNumber == "" {
		return errors.New("phoneNumber is required")
	}
	if req.ApiKey == "" {
		return errors.New("apiKey is required")
	}
	return nil
}

func generateWebhookVerifyToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
