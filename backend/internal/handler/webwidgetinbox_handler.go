package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"backend/internal/channel"
	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"

	appcrypto "backend/internal/crypto"

	"github.com/gofiber/fiber/v2"
)

type WebWidgetInboxHandler struct {
	widgetRepo *repo.ChannelWebWidgetRepo
	inboxRepo  *repo.InboxRepo
	cipher     *appcrypto.Cipher
	cfg        *config.Config
}

func NewWebWidgetInboxHandler(
	widgetRepo *repo.ChannelWebWidgetRepo,
	inboxRepo *repo.InboxRepo,
	cipher *appcrypto.Cipher,
	cfg *config.Config,
) *WebWidgetInboxHandler {
	return &WebWidgetInboxHandler{
		widgetRepo: widgetRepo,
		inboxRepo:  inboxRepo,
		cipher:     cipher,
		cfg:        cfg,
	}
}

func generateToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Create provisions a new Web Widget channel and inbox.
// @Summary Create web widget inbox
// @Description Creates a new Web Widget channel with encrypted HMAC token and returns embed script
// @Tags Admin/Inboxes
// @Accept json
// @Produce json
// @Param body body dto.CreateWebWidgetInboxReq true "Widget inbox config"
// @Success 201 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /api/v1/accounts/{aid}/inboxes/web_widget [post]
func (h *WebWidgetInboxHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateWebWidgetInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	websiteToken := generateToken(32)
	hmacToken := generateToken(32)
	hmacCiphertext, err := h.cipher.Encrypt(hmacToken)
	if err != nil {
		logger.Error().Str("component", "webwidget").Err(err).Msg("failed to encrypt hmac token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt hmac token"))
	}

	widgetColor := "#0084FF"
	if req.WidgetColor != nil {
		widgetColor = *req.WidgetColor
	}
	welcomeTitle := ""
	if req.WelcomeTitle != nil {
		welcomeTitle = *req.WelcomeTitle
	}
	welcomeTagline := ""
	if req.WelcomeTagline != nil {
		welcomeTagline = *req.WelcomeTagline
	}
	replyTime := "in_a_few_minutes"
	if req.ReplyTime != nil {
		replyTime = *req.ReplyTime
	}
	featureFlags := `{"attachments":true,"emoji_picker":true,"end_conversation":false}`
	if req.FeatureFlags != nil {
		featureFlags = *req.FeatureFlags
	}

	widget := &model.ChannelWebWidget{
		AccountID:           accountID,
		WebsiteToken:        websiteToken,
		HmacTokenCiphertext: hmacCiphertext,
		WebsiteURL:          req.WebsiteURL,
		WidgetColor:         widgetColor,
		WelcomeTitle:        welcomeTitle,
		WelcomeTagline:      welcomeTagline,
		ReplyTime:           replyTime,
		FeatureFlags:        featureFlags,
	}

	if err := h.widgetRepo.Create(c.Context(), widget); err != nil {
		logger.Warn().Str("component", "webwidget.provision").Err(err).Msg("failed to create widget channel")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create widget channel"))
	}

	inbox := &model.Inbox{
		AccountID:   accountID,
		ChannelID:   widget.ID,
		Name:        req.Name,
		ChannelType: string(channel.KindWebWidget),
	}
	if err := h.inboxRepo.Create(c.Context(), inbox); err != nil {
		logger.Error().Str("component", "webwidget").Err(err).Msg("failed to create inbox for widget")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create inbox"))
	}

	widget.InboxID = inbox.ID
	_ = h.widgetRepo.UpdateInboxID(c.Context(), widget.ID, inbox.ID)

	embedScript := h.generateEmbedScript(websiteToken)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.WebWidgetInboxResp{
		InboxResp: inboxModelToResp(inbox),
		Channel: dto.WebWidgetChannelResp{
			ID:             widget.ID,
			WebsiteToken:   widget.WebsiteToken,
			WebsiteURL:     widget.WebsiteURL,
			WidgetColor:    widget.WidgetColor,
			WelcomeTitle:   widget.WelcomeTitle,
			WelcomeTagline: widget.WelcomeTagline,
			ReplyTime:      widget.ReplyTime,
			FeatureFlags:   widget.FeatureFlags,
			EmbedScript:    embedScript,
			CreatedAt:      widget.CreatedAt,
			UpdatedAt:      widget.UpdatedAt,
		},
	}))
}

// RotateHmac generates a new HMAC token for the widget channel.
// @Summary Rotate HMAC token
// @Description Generates a new HMAC token and returns it once
// @Tags Admin/Inboxes
// @Produce json
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /api/v1/accounts/{aid}/inboxes/{id}/rotate_hmac [post]
func (h *WebWidgetInboxHandler) RotateHmac(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	widget, err := h.widgetRepo.FindByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	newHmacToken := generateToken(32)
	newCiphertext, err := h.cipher.Encrypt(newHmacToken)
	if err != nil {
		logger.Error().Str("component", "webwidget").Err(err).Msg("failed to encrypt new hmac token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to encrypt new hmac token"))
	}

	if err := h.widgetRepo.UpdateHmacToken(c.Context(), widget.ID, newCiphertext); err != nil {
		logger.Error().Str("component", "webwidget").Err(err).Msg("failed to rotate hmac token")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to rotate hmac token"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(dto.RotateHmacResp{
		HmacToken: newHmacToken,
	}))
}

func (h *WebWidgetInboxHandler) generateEmbedScript(websiteToken string) string {
	return fmt.Sprintf(`<script src="%s/widget/%s" data-website-token="%s" defer></script>`,
		h.cfg.WidgetPublicBaseURL,
		websiteToken,
		websiteToken,
	)
}
