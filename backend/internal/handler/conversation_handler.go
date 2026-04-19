package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type ConversationHandler struct {
	svc              *service.ConversationService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
}

func NewConversationHandler(
	svc *service.ConversationService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
) *ConversationHandler {
	return &ConversationHandler{
		svc:              svc,
		inboxRepo:        inboxRepo,
		contactInboxRepo: contactInboxRepo,
	}
}

func (h *ConversationHandler) Create(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelApi)
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
	cid, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	if err := h.svc.UpdateLastSeen(c.Context(), cid); err != nil {
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

	convos, total, err := h.svc.ListByAccount(c.Context(), filter)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.ConversationsToResp(convos),
	}))
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
