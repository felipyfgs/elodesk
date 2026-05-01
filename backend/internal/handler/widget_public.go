package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"backend/internal/channel/webwidget"
	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"

	"github.com/gofiber/fiber/v2"
)

type WidgetPublicHandler struct {
	sessionSvc       *webwidget.SessionService
	identifySvc      *webwidget.IdentifyService
	widgetRepo       *repo.ChannelWebWidgetRepo
	conversationRepo *repo.ConversationRepo
	messageRepo      *repo.MessageRepo
	jwtSvc           *webwidget.VisitorJWTService
	sseHandler       *webwidget.SSEHandler
	cfg              *config.Config
}

func NewWidgetPublicHandler(
	sessionSvc *webwidget.SessionService,
	identifySvc *webwidget.IdentifyService,
	widgetRepo *repo.ChannelWebWidgetRepo,
	conversationRepo *repo.ConversationRepo,
	messageRepo *repo.MessageRepo,
	jwtSvc *webwidget.VisitorJWTService,
	sseHandler *webwidget.SSEHandler,
	cfg *config.Config,
) *WidgetPublicHandler {
	return &WidgetPublicHandler{
		sessionSvc:       sessionSvc,
		identifySvc:      identifySvc,
		widgetRepo:       widgetRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		jwtSvc:           jwtSvc,
		sseHandler:       sseHandler,
		cfg:              cfg,
	}
}

// EmbedScript serves the widget JS bundle bootstrap.
// @Summary Get widget embed script
// @Description Returns JavaScript that bootstraps the chat widget
// @Tags Public/Widget
// @Param websiteToken path string true "Widget token"
// @Success 200 {string} string "JavaScript bundle"
// @Failure 404 {object} dto.APIError
// @Router /widget/{websiteToken} [get]
func (h *WidgetPublicHandler) EmbedScript(c *fiber.Ctx) error {
	websiteToken := c.Params("websiteToken")
	widget, err := h.widgetRepo.FindByWebsiteToken(c.Context(), websiteToken)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("")
	}

	config := webwidget.GetWidgetConfig(widget)
	configJSON, _ := json.Marshal(config)

	body := fmt.Sprintf(`(function(){var c=%s;var s=document.createElement('script');s.src='%s/widget.js';s.setAttribute('data-website-token','%s');s.defer=true;document.head.appendChild(s);})();`,
		string(configJSON),
		h.cfg.WidgetPublicBaseURL,
		websiteToken,
	)

	c.Set("Content-Type", "application/javascript")
	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("ETag", fmt.Sprintf(`"%s"`, websiteToken))
	return c.SendString(body)
}

// CreateSession creates or resumes a visitor session.
// @Summary Create visitor session
// @Description Creates anonymous contact or resumes via JWT cookie
// @Tags Public/Widget
// @Accept json
// @Produce json
// @Param body body object true "Session request"
// @Success 201 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /api/v1/widget/sessions [post]
func (h *WidgetPublicHandler) CreateSession(c *fiber.Ctx) error {
	var req struct {
		WebsiteToken string `json:"websiteToken" validate:"required"`
	}
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	var existingClaims *webwidget.VisitorClaims
	cookie := c.Cookies("elodesk_widget_session_" + req.WebsiteToken)
	if cookie != "" {
		claims, err := h.jwtSvc.Parse(cookie)
		if err == nil {
			existingClaims = claims
		}
	}

	ip := c.IP()
	result, err := h.sessionSvc.CreateOrResumeSession(c.Context(), req.WebsiteToken, existingClaims, ip)
	if err != nil {
		logger.Warn().Str("component", "webwidget").Err(err).Msg("failed to create session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create session"))
	}

	isDev := h.cfg.Environment == "development"
	sameSite := "Lax"
	if !isDev {
		sameSite = "None"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "elodesk_widget_session_" + req.WebsiteToken,
		Value:    result.JWT,
		HTTPOnly: true,
		Secure:   !isDev,
		SameSite: sameSite,
		Path:     "/",
	})

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(result))
}

// SendMessage sends a message from the visitor.
// @Summary Send visitor message
// @Description Creates an incoming message in the visitor's conversation
// @Tags Public/Widget
// @Accept json
// @Produce json
// @Param body body object true "Message request"
// @Success 201 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 401 {object} dto.APIError
// @Router /api/v1/widget/messages [post]
func (h *WidgetPublicHandler) SendMessage(c *fiber.Ctx) error {
	claims, err := h.getVisitorClaims(c)
	if err != nil {
		return nil
	}

	widget, err := h.widgetRepo.FindByWebsiteToken(c.Context(), claims.WebsiteToken)
	if err != nil {
		logger.Warn().Str("component", "webwidget").Err(err).Msg("widget not found for session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "invalid session"))
	}

	var req struct {
		Content       string  `json:"content" validate:"required"`
		AttachmentIDs []int64 `json:"attachmentIds,omitempty"`
	}
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	contentType := model.MessageContentType(0)
	if req.Content == "" && len(req.AttachmentIDs) > 0 {
		contentType = model.MessageContentType(9)
	}

	senderType := "Contact"
	msg := &model.Message{
		AccountID:      widget.AccountID,
		InboxID:        widget.InboxID,
		ConversationID: claims.ConversationID,
		MessageType:    model.MessageIncoming,
		ContentType:    contentType,
		Content:        &req.Content,
		SenderType:     &senderType,
		SenderID:       &claims.ContactID,
	}

	if len(req.AttachmentIDs) > 0 {
		idsJSON, _ := json.Marshal(req.AttachmentIDs)
		s := string(idsJSON)
		msg.ContentAttrs = &s
	}

	created, err := h.messageRepo.Create(c.Context(), msg)
	if err != nil {
		logger.Warn().Str("component", "webwidget").Err(err).Msg("failed to create message")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to send message"))
	}

	_ = h.conversationRepo.UpdateLastSeen(c.Context(), claims.ConversationID)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(created))
}

// Identify verifies and upgrades an anonymous contact via HMAC.
// @Summary Identify visitor
// @Description Verifies HMAC and upgrades anonymous contact to identified
// @Tags Public/Widget
// @Accept json
// @Produce json
// @Param body body object true "Identify request"
// @Success 200 {object} dto.APIResponse
// @Failure 401 {object} dto.APIError
// @Router /api/v1/widget/identify [post]
func (h *WidgetPublicHandler) Identify(c *fiber.Ctx) error {
	claims, err := h.getVisitorClaims(c)
	if err != nil {
		return nil
	}

	var req webwidget.IdentifyRequest
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.identifySvc.Identify(c.Context(), claims, &req)
	if err != nil {
		if err.Error() == "invalid_identifier_hash" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid_identifier_hash"))
		}
		logger.Warn().Str("component", "webwidget.identify").Err(err).Msg("identify failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "identify failed"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(result))
}

// PollMessages returns new messages since last seen.
// @Summary Poll for new messages
// @Description Returns messages newer than the given message ID
// @Tags Public/Widget
// @Produce json
// @Param after query int false "Last seen message ID"
// @Param limit query int false "Max messages (default 20)"
// @Success 200 {object} dto.APIResponse
// @Failure 401 {object} dto.APIError
// @Router /api/v1/widget/messages [get]
func (h *WidgetPublicHandler) PollMessages(c *fiber.Ctx) error {
	claims, err := h.getVisitorClaims(c)
	if err != nil {
		return nil
	}

	after := c.QueryInt("after", 0)
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	messages, _, err := h.messageRepo.ListByConversation(c.Context(), repo.MessageListFilter{
		ConversationID: claims.ConversationID,
		AccountID:      0,
		Page:           1,
		PerPage:        limit,
	})
	if err != nil {
		logger.Warn().Str("component", "webwidget").Err(err).Msg("failed to list messages")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to fetch messages"))
	}

	if after > 0 {
		var filtered []model.Message
		for _, m := range messages {
			if m.ID > int64(after) {
				filtered = append(filtered, m)
			}
		}
		messages = filtered
	}

	return c.JSON(dto.SuccessResp(messages))
}

// GetAttachmentPresigned returns a presigned upload URL.
// @Summary Get attachment upload URL
// @Description Returns a presigned URL for uploading an attachment
// @Tags Public/Widget
// @Accept json
// @Produce json
// @Param body body object true "Attachment request"
// @Success 200 {object} dto.APIResponse
// @Failure 401 {object} dto.APIError
// @Failure 413 {object} dto.APIError
// @Router /api/v1/widget/attachments [post]
func (h *WidgetPublicHandler) GetAttachmentPresigned(c *fiber.Ctx) error {
	_, err := h.getVisitorClaims(c)
	if err != nil {
		return nil
	}

	var req struct {
		FileName    string `json:"fileName" validate:"required"`
		ContentType string `json:"contentType" validate:"required"`
		Size        int64  `json:"size" validate:"required"`
	}
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	maxSize := int64(10_000_000)
	if req.Size > maxSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(dto.ErrorResp("Payload Too Large", "attachment_too_large"))
	}

	return c.Status(fiber.StatusNotImplemented).JSON(dto.ErrorResp("Not Implemented", "presigned uploads not yet configured"))
}

func (h *WidgetPublicHandler) SSE(c *fiber.Ctx) error {
	return h.sseHandler.HandleSSE(c)
}

func (h *WidgetPublicHandler) getVisitorClaims(c *fiber.Ctx) (*webwidget.VisitorClaims, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		c.Request().Header.VisitAllCookie(func(key, value []byte) {
			if authHeader != "" {
				return
			}
			name := string(key)
			if strings.HasPrefix(name, "elodesk_widget_session_") {
				cookie := c.Cookies(name)
				if cookie != "" {
					authHeader = "Bearer " + cookie
				}
			}
		})
	}
	if authHeader == "" {
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid_visitor_token"))
		return nil, fmt.Errorf("missing auth")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid_visitor_token"))
		return nil, fmt.Errorf("invalid auth format")
	}

	claims, err := h.jwtSvc.Parse(parts[1])
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "invalid_visitor_token"))
		return nil, fmt.Errorf("invalid jwt: %w", err)
	}

	return claims, nil
}
