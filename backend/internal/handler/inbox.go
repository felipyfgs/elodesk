package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

func handleNotFound(c *fiber.Ctx, err error) error {
	if repo.IsErrNotFound(err) {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "resource not found"))
	}
	logger.Error().Str("component", "handler").Err(err).Msg("handler error")
	return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
}

// handleError maps domain errors to standard HTTP status codes. Callers
// SHOULD pass the error from service/repo layers; the function checks known
// sentinel errors via errors.Is and maps to 404, 409, 422, or 500.
func handleError(c *fiber.Ctx, err error) error {
	if repo.IsErrNotFound(err) {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "resource not found"))
	}
	if errors.Is(err, repo.ErrConflict) {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "resource already exists"))
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "duplicate resource"))
	}
	logger.Error().Str("component", "handler").Err(err).Msg("handler error")
	return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
}

type InboxHandler struct {
	svc         *service.InboxService
	auditLogger *audit.Logger
}

func NewInboxHandler(svc *service.InboxService, auditLogger *audit.Logger) *InboxHandler {
	return &InboxHandler{svc: svc, auditLogger: auditLogger}
}

func (h *InboxHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	creds, err := h.svc.ProvisionAPI(c.Context(), accountID, service.ProvisionAPIInput{
		Name:                 req.Name,
		WebhookURL:           req.WebhookURL,
		HmacMandatory:        req.HmacMandatory,
		AdditionalAttributes: req.AdditionalAttributes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidAgentReplyTimeWindow) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_agent_reply_time_window"))
		}
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil {
		inboxID := creds.Inbox.ID
		h.auditLogger.LogFromCtx(c, "inbox.created", "inbox", &inboxID, fiber.Map{
			"name":         creds.Inbox.Name,
			"channel_type": creds.Inbox.ChannelType,
		})
	}

	resp := dto.CreateInboxResp{
		InboxResp: dto.InboxResp{
			ID:          creds.Inbox.ID,
			AccountID:   creds.Inbox.AccountID,
			ChannelID:   creds.Inbox.ChannelID,
			Name:        creds.Inbox.Name,
			ChannelType: creds.Inbox.ChannelType,
			CreatedAt:   creds.Inbox.CreatedAt,
		},
		Identifier: creds.ChannelAPI.Identifier,
		ApiToken:   creds.ApiToken,
		HmacToken:  creds.HmacToken,
		Secret:     creds.Secret,
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(resp))
}

// UpdateChannelAPI handles PUT /inboxes/:id when the inbox is Channel::Api.
// Accepts the whitelist in UpdateChannelAPIReq. Other channel kinds have
// their own handlers — this one rejects non-Api inboxes to avoid stepping on
// per-kind update logic.
func (h *InboxHandler) UpdateChannelAPI(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateChannelAPIReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if req.Name != "" {
		if err := h.svc.UpdateName(c.Context(), int64(id), accountID, req.Name); err != nil {
			return handleNotFound(c, err)
		}
	}

	ch, err := h.svc.UpdateChannelAPIEditable(c.Context(), int64(id), accountID, service.UpdateAPIInput{
		WebhookURL:           req.WebhookURL,
		HmacMandatory:        req.HmacMandatory,
		AdditionalAttributes: req.AdditionalAttributes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidAgentReplyTimeWindow) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_agent_reply_time_window"))
		}
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil {
		inboxID := int64(id)
		h.auditLogger.LogFromCtx(c, "inbox.updated", "inbox", &inboxID, fiber.Map{
			"channel_type": "Channel::Api",
		})
	}

	return c.JSON(dto.SuccessResp(channelAPIModelToResp(ch)))
}

// GetChannelAPI returns the editable, non-secret Channel::Api metadata for an
// inbox. Plaintext API/HMAC tokens are intentionally never returned here.
func (h *InboxHandler) GetChannelAPI(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	ch, err := h.svc.GetChannelAPIEditable(c.Context(), int64(id), accountID)
	if err != nil {
		if errors.Is(err, repo.ErrChannelAPINotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "channel_api_not_found"))
		}
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(channelAPIModelToResp(ch)))
}

// RotateAPIToken issues a new identifier + api_token for the inbox. The
// previous credentials are invalidated on success. RBAC: caller must be
// Owner/Admin on the account (enforced by the RequireAdmin middleware in
// router.go).
func (h *InboxHandler) RotateAPIToken(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	ch, apiToken, secret, err := h.svc.RotateAPIToken(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil {
		inboxID := int64(id)
		h.auditLogger.LogFromCtx(c, "inbox.api_token_rotated", "inbox", &inboxID, nil)
	}

	return c.JSON(dto.SuccessResp(dto.RotateAPITokenResp{
		Identifier: ch.Identifier,
		ApiToken:   apiToken,
		Secret:     secret,
	}))
}

func channelAPIModelToResp(ch *model.ChannelAPI) dto.ChannelAPIResp {
	return dto.ChannelAPIResp{
		ID:                   ch.ID,
		Identifier:           ch.Identifier,
		WebhookURL:           ch.WebhookURL,
		HmacMandatory:        ch.HmacMandatory,
		AdditionalAttributes: ch.AdditionalAttributes,
		CreatedAt:            ch.CreatedAt,
		UpdatedAt:            ch.UpdatedAt,
	}
}

func (h *InboxHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	inboxes, err := h.svc.ListByAccount(c.Context(), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	payload := make([]dto.InboxResp, len(inboxes))
	for i := range inboxes {
		payload[i] = inboxModelToResp(&inboxes[i])
	}

	return c.JSON(dto.SuccessResp(payload))
}

func (h *InboxHandler) GetByID(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	inbox, err := h.svc.GetByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(inboxModelToResp(inbox)))
}

func inboxModelToResp(i *model.Inbox) dto.InboxResp {
	return dto.InboxResp{
		ID:          i.ID,
		AccountID:   i.AccountID,
		ChannelID:   i.ChannelID,
		Name:        i.Name,
		ChannelType: i.ChannelType,
		CreatedAt:   i.CreatedAt,
	}
}

func businessHoursModelToResp(m *model.InboxBusinessHours) dto.InboxBusinessHoursResp {
	var createdAt *time.Time
	var updatedAt *time.Time
	if !m.CreatedAt.IsZero() {
		createdAt = &m.CreatedAt
	}
	if !m.UpdatedAt.IsZero() {
		updatedAt = &m.UpdatedAt
	}
	return dto.InboxBusinessHoursResp{
		InboxID:   m.InboxID,
		Timezone:  m.Timezone,
		Schedule:  businessHoursScheduleToDTO(m.Schedule),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func businessHoursScheduleToDTO(schedule map[string]model.BusinessHoursSlot) map[string]dto.BusinessHoursSlot {
	out := make(map[string]dto.BusinessHoursSlot, len(schedule))
	for day, slot := range schedule {
		out[day] = dto.BusinessHoursSlot{
			Enabled:     slot.Enabled,
			OpenHour:    slot.OpenHour,
			OpenMinute:  slot.OpenMinute,
			CloseHour:   slot.CloseHour,
			CloseMinute: slot.CloseMinute,
		}
	}
	return out
}

func businessHoursScheduleToModel(schedule map[string]dto.BusinessHoursSlot) map[string]model.BusinessHoursSlot {
	out := make(map[string]model.BusinessHoursSlot, len(schedule))
	for day, slot := range schedule {
		out[day] = model.BusinessHoursSlot{
			Enabled:     slot.Enabled,
			OpenHour:    slot.OpenHour,
			OpenMinute:  slot.OpenMinute,
			CloseHour:   slot.CloseHour,
			CloseMinute: slot.CloseMinute,
		}
	}
	return out
}

func (h *InboxHandler) GetBusinessHours(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	hours, err := h.svc.GetBusinessHours(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(businessHoursModelToResp(hours)))
}

func (h *InboxHandler) UpdateBusinessHours(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateInboxBusinessHoursReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	hours, err := h.svc.UpdateBusinessHours(c.Context(), int64(id), accountID, req.Timezone, businessHoursScheduleToModel(req.Schedule))
	if err != nil {
		if errors.Is(err, service.ErrInvalidBusinessHours) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_business_hours"))
		}
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil {
		inboxID := int64(id)
		h.auditLogger.LogFromCtx(c, "inbox.business_hours_updated", "inbox", &inboxID, fiber.Map{
			"timezone": hours.Timezone,
		})
	}

	return c.JSON(dto.SuccessResp(businessHoursModelToResp(hours)))
}

func (h *InboxHandler) ListAgents(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	agents, err := h.svc.ListInboxAgents(c.Context(), int64(id), accountID)
	if err != nil {
		logger.Error().Str("component", "handler").Err(err).Msg("list inbox agents error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	payload := make([]dto.InboxAgentResp, len(agents))
	for i := range agents {
		payload[i] = dto.InboxAgentResp{
			ID:        agents[i].ID,
			InboxID:   agents[i].InboxID,
			UserID:    agents[i].UserID,
			CreatedAt: agents[i].CreatedAt,
		}
	}

	return c.JSON(dto.SuccessResp(payload))
}

func (h *InboxHandler) SetAgents(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.SetInboxAgentsReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.SetInboxAgents(c.Context(), int64(id), accountID, req.UserIDs); err != nil {
		logger.Error().Str("component", "handler").Err(err).Msg("set inbox agents error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

func (h *InboxHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	var req dto.UpdateInboxReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.UpdateName(c.Context(), int64(id), accountID, req.Name); err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(nil))
}

func (h *InboxHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid inbox id"))
	}

	if err := h.svc.DeleteInbox(c.Context(), int64(id), accountID); err != nil {
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil {
		inboxID := int64(id)
		h.auditLogger.LogFromCtx(c, "inbox.deleted", "inbox", &inboxID, nil)
	}

	return c.JSON(dto.SuccessResp(nil))
}
