package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type ConversationHandler struct {
	svc              *service.ConversationService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
	auditLogger      *audit.Logger
}

func NewConversationHandler(
	svc *service.ConversationService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	auditLogger *audit.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		svc:              svc,
		inboxRepo:        inboxRepo,
		contactInboxRepo: contactInboxRepo,
		auditLogger:      auditLogger,
	}
}

func (h *ConversationHandler) Create(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	sourceID := c.Params("sourceId")

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	ci, err := h.contactInboxRepo.FindBySourceID(c.Context(), sourceID, inbox.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.Create(c.Context(), accountID, inbox.ID, ci.ContactID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}

func (h *ConversationHandler) UpdateLastSeen(c *fiber.Ctx) error {
	// Ownership check: conversation must belong to the inbox that owns the
	// authenticated channel API token. Prevents enumeration across tenants.
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	cid, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.GetByID(c.Context(), cid, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	if convo.InboxID != inbox.ID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "conversation not found"))
	}

	if err := h.svc.UpdateLastSeen(c.Context(), cid); err != nil {
		logger.Error().Str("component", "conversations").Err(err).Msg("failed to update last seen")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update last seen"))
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *ConversationHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	filter := repo.ConversationFilter{
		AccountID: accountID,
		Page:      page,
		PerPage:   perPage,
		SortBy:    parseConversationSort(c.Query("sort_by")),
	}

	if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
		uid := u.ID
		filter.CurrentUser = &uid
	}

	if inboxIDStr := c.Query("inbox_id"); inboxIDStr != "" {
		inboxID, err := strconv.ParseInt(inboxIDStr, 10, 64)
		if err == nil {
			filter.InboxID = &inboxID
		}
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.Atoi(statusStr)
		if err == nil {
			s := model.ConversationStatus(status)
			filter.Status = &s
		}
	}

	filter.AssigneeType = parseAssigneeType(c.Query("assignee_type"))

	// assignee_type takes precedence; assignee_id is only honored when no
	// assignee_type filter is active to avoid double assignee constraints.
	if filter.AssigneeType == "" {
		if assigneeIDStr := c.Query("assignee_id"); assigneeIDStr != "" {
			assigneeID, err := strconv.ParseInt(assigneeIDStr, 10, 64)
			if err == nil {
				filter.AssigneeID = &assigneeID
			}
		}
	}

	convos, total, err := h.svc.ListByAccount(c.Context(), filter)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.ConversationsToResp(convos),
	}))
}

func (h *ConversationHandler) Meta(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	u, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "user not found"))
	}
	userID := u.ID

	var inboxID *int64
	if inboxIDStr := c.Query("inbox_id"); inboxIDStr != "" {
		if id, err := strconv.ParseInt(inboxIDStr, 10, 64); err == nil {
			inboxID = &id
		}
	}

	counts, err := h.svc.CountMeta(c.Context(), accountID, userID, inboxID)
	if err != nil {
		logger.Error().Str("component", "conversations").Err(err).Msg("failed to load conversation meta")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to load conversation meta"))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"payload": counts}))
}

func parseConversationSort(raw string) repo.ConversationSortKey {
	switch repo.ConversationSortKey(raw) {
	case repo.ConversationSortLastActivityAsc,
		repo.ConversationSortLastActivityDesc,
		repo.ConversationSortCreatedAsc,
		repo.ConversationSortCreatedDesc:
		return repo.ConversationSortKey(raw)
	}
	return repo.ConversationSortLastActivityDesc
}

func parseAssigneeType(raw string) repo.ConversationAssigneeType {
	switch repo.ConversationAssigneeType(raw) {
	case repo.ConversationAssigneeTypeMine,
		repo.ConversationAssigneeTypeUnassigned,
		repo.ConversationAssigneeTypeAll:
		return repo.ConversationAssigneeType(raw)
	}
	return ""
}

func (h *ConversationHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	convo, err := h.svc.GetByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}

func (h *ConversationHandler) Assign(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req struct {
		AssigneeID *int64 `json:"assignee_id"`
		TeamID     *int64 `json:"team_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	convo, err := h.svc.Assign(c.Context(), int64(id), accountID, req.AssigneeID, req.TeamID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}

func (h *ConversationHandler) ToggleStatus(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req struct {
		Status int `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	status := model.ConversationStatus(req.Status)
	switch status {
	case model.ConversationOpen, model.ConversationResolved, model.ConversationPending, model.ConversationSnoozed:
	default:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid status value"))
	}

	convo, err := h.svc.ToggleStatus(c.Context(), int64(id), accountID, status)
	if err != nil {
		return handleNotFound(c, err)
	}

	if h.auditLogger != nil && status == model.ConversationResolved {
		convID := convo.ID
		h.auditLogger.LogFromCtx(c, "conversation.resolved", "conversation", &convID, fiber.Map{
			"inbox_id":   convo.InboxID,
			"contact_id": convo.ContactID,
		})
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}
