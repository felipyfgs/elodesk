package handler

import (
	"strconv"
	"strings"

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
	messageSvc       *service.MessageService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
	conversationRepo *repo.ConversationRepo
	agentRepo        *repo.AgentRepo
	teamRepo         *repo.TeamRepo
	auditLogger      *audit.Logger
}

func NewConversationHandler(
	svc *service.ConversationService,
	messageSvc *service.MessageService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
	conversationRepo *repo.ConversationRepo,
	agentRepo *repo.AgentRepo,
	teamRepo *repo.TeamRepo,
	auditLogger *audit.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		svc:              svc,
		messageSvc:       messageSvc,
		inboxRepo:        inboxRepo,
		contactInboxRepo: contactInboxRepo,
		conversationRepo: conversationRepo,
		agentRepo:        agentRepo,
		teamRepo:         teamRepo,
		auditLogger:      auditLogger,
	}
}

// CreateAuthenticated lets an authenticated agent start a new conversation with
// an existing contact from the dashboard. Optionally sends a first outgoing
// message within the same request. Mirrors Chatwoot's
// POST /api/v1/accounts/:account_id/conversations.
func (h *ConversationHandler) CreateAuthenticated(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateAuthenticatedConversationReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if req.ContactID == 0 || req.InboxID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "contact_id and inbox_id are required"))
	}

	inbox, err := h.inboxRepo.FindByID(c.Context(), req.InboxID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	// Validate assignee is a member of the account
	if req.AssigneeID != nil && h.agentRepo != nil {
		ok, err := h.agentRepo.IsMember(c.Context(), accountID, *req.AssigneeID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to verify assignee"))
		}
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "assignee does not belong to this account"))
		}
	}

	// Validate team is scoped to the account
	if req.TeamID != nil && h.teamRepo != nil {
		if _, err := h.teamRepo.FindByID(c.Context(), *req.TeamID, accountID); err != nil {
			return handleNotFound(c, err)
		}
	}

	opts := service.ConversationCreateOpts{
		AssigneeID:           req.AssigneeID,
		TeamID:               req.TeamID,
		AdditionalAttributes: req.AdditionalAttributes,
		CustomAttributes:     req.CustomAttributes,
	}
	if req.Status != nil {
		switch strings.ToLower(*req.Status) {
		case "resolved":
			s := model.ConversationResolved
			opts.Status = &s
		case "open":
			s := model.ConversationOpen
			opts.Status = &s
		case "pending":
			s := model.ConversationPending
			opts.Status = &s
		case "snoozed":
			s := model.ConversationSnoozed
			opts.Status = &s
		}
	}

	convo, err := h.svc.CreateWithOpts(c.Context(), accountID, inbox.ID, req.ContactID, opts)
	if err != nil {
		return handleNotFound(c, err)
	}

	if req.Message != nil && strings.TrimSpace(req.Message.Content) != "" && h.messageSvc != nil {
		content := req.Message.Content
		msg := &model.Message{
			Content:     &content,
			Private:     req.Message.Private,
			MessageType: model.MessageOutgoing,
			ContentType: model.ContentTypeText,
		}
		if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
			senderType := "User"
			uid := u.ID
			msg.SenderType = &senderType
			msg.SenderID = &uid
		}
		if _, err := h.messageSvc.Create(c.Context(), accountID, inbox.ID, convo.ID, msg); err != nil {
			logger.Error().Str("component", "conversations").Err(err).Int64("conversation_id", convo.ID).Msg("failed to create initial message")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "conversation created but failed to send initial message"))
		}
	}

	if h.auditLogger != nil {
		convID := convo.ID
		h.auditLogger.LogFromCtx(c, "conversation.created", "conversation", &convID, fiber.Map{
			"inbox_id":   convo.InboxID,
			"contact_id": convo.ContactID,
		})
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
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

	var req dto.CreateConversationReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	opts := service.ConversationCreateOpts{
		CustomAttributes:     req.CustomAttributes,
		AdditionalAttributes: req.AdditionalAttributes,
		AssigneeID:           req.AssigneeID,
		TeamID:               req.TeamID,
	}

	if req.Status != nil {
		switch strings.ToLower(*req.Status) {
		case "resolved":
			s := model.ConversationResolved
			opts.Status = &s
		case "open":
			s := model.ConversationOpen
			opts.Status = &s
		case "pending":
			s := model.ConversationPending
			opts.Status = &s
		case "snoozed":
			s := model.ConversationSnoozed
			opts.Status = &s
		}
	}

	convo, err := h.svc.CreateWithOpts(c.Context(), accountID, inbox.ID, ci.ContactID, opts)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
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

	if teamIDStr := c.Query("team_id"); teamIDStr != "" {
		teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
		if err == nil {
			filter.TeamID = &teamID
		}
	}

	if q := c.Query("q"); q != "" {
		filter.Query = q
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

	payload, meta, err := h.svc.ListWithMeta(c.Context(), filter)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationListResp{
		Meta:    meta,
		Payload: payload,
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

	hydrated, err := h.svc.FindByIDFull(c.Context(), accountID, int64(id))
	if err != nil {
		return handleNotFound(c, err)
	}
	row := repo.ConversationHydratedToFullRow(hydrated)
	if row.LastNonActivityMessage != nil && h.messageSvc != nil {
		senders := h.messageSvc.HydrateMessageSenders(c.Context(), []model.Message{*row.LastNonActivityMessage}, accountID)
		row.LastNonActivitySender = senders[row.LastNonActivityMessage.ID]
	}
	return c.JSON(dto.SuccessResp(dto.ConversationToRespFull(&row)))
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

	if _, err := h.svc.Assign(c.Context(), int64(id), accountID, req.AssigneeID, req.TeamID); err != nil {
		return handleNotFound(c, err)
	}

	// Re-fetch the hydrated conversation so the response carries the same
	// `meta`, `inbox`, and `last_non_activity_message` shape the frontend
	// store relies on. Without this, `upsert` would strip those fields.
	hydrated, err := h.svc.FindByIDFull(c.Context(), accountID, int64(id))
	if err != nil {
		return handleNotFound(c, err)
	}
	row := repo.ConversationHydratedToFullRow(hydrated)
	if row.LastNonActivityMessage != nil && h.messageSvc != nil {
		senders := h.messageSvc.HydrateMessageSenders(c.Context(), []model.Message{*row.LastNonActivityMessage}, accountID)
		row.LastNonActivitySender = senders[row.LastNonActivityMessage.ID]
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToRespFull(&row)))
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

func (h *ConversationHandler) ListByContact(c *fiber.Ctx) error {
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

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	var convos []model.Conversation
	var total int

	if ci.HmacVerified {
		convos, total, err = h.conversationRepo.ListByContactID(c.Context(), ci.ContactID, accountID, page, perPage)
	} else {
		convos, total, err = h.conversationRepo.ListByContactInboxID(c.Context(), ci.ID, accountID, page, perPage)
	}
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.PaginatedResp[dto.ConversationResp]{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.ConversationsToResp(convos),
	}))
}

func (h *ConversationHandler) ShowPublic(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.GetByID(c.Context(), id, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if convo.InboxID != inbox.ID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "conversation not found"))
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}

func (h *ConversationHandler) TogglePublicStatus(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	var status model.ConversationStatus
	switch strings.ToLower(req.Status) {
	case "resolved":
		status = model.ConversationResolved
	case "open":
		status = model.ConversationOpen
	case "pending":
		status = model.ConversationPending
	case "snoozed":
		status = model.ConversationSnoozed
	default:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid status value"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.GetByID(c.Context(), id, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if convo.InboxID != inbox.ID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "conversation not found"))
	}

	if convo.Status == status {
		return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
	}

	convo, err = h.svc.ToggleStatus(c.Context(), id, accountID, status)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationToResp(convo)))
}

func (h *ConversationHandler) ToggleTyping(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req struct {
		TypingStatus string `json:"typing_status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	if req.TypingStatus != "on" && req.TypingStatus != "off" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "typing_status must be 'on' or 'off'"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.GetByID(c.Context(), id, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if convo.InboxID != inbox.ID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "conversation not found"))
	}

	eventName := "conversation.typing_on"
	if req.TypingStatus == "off" {
		eventName = "conversation.typing_off"
	}

	logger.Info().Str("component", "conversations").Str("event", eventName).Int64("conversation_id", id).Msg("typing event")

	return c.SendStatus(fiber.StatusOK)
}

func (h *ConversationHandler) UpdateLastSeenPublic(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	sourceID := c.Params("sourceId")

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	ci, err := h.contactInboxRepo.FindBySourceID(c.Context(), sourceID, inbox.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	convo, err := h.svc.GetByID(c.Context(), id, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if convo.InboxID != inbox.ID || convo.ContactInboxID == nil || *convo.ContactInboxID != ci.ID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "conversation not found"))
	}

	if err := h.svc.UpdateLastSeen(c.Context(), id); err != nil {
		logger.Error().Str("component", "conversations").Err(err).Msg("failed to update last seen")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update last seen"))
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}
