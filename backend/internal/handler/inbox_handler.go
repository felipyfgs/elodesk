package handler

import (
	"github.com/gofiber/fiber/v2"

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
	logger.Error().Str("component", "handler").Err(err).Msg("inbox handler error")
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

	creds, err := h.svc.Provision(c.Context(), accountID, req.Name)
	if err != nil {
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
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(resp))
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
