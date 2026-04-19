package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type ContactHandler struct {
	svc              *service.ContactService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
}

func NewContactHandler(
	svc *service.ContactService,
	inboxRepo *repo.InboxRepo,
	contactInboxRepo *repo.ContactInboxRepo,
) *ContactHandler {
	return &ContactHandler{
		svc:              svc,
		inboxRepo:        inboxRepo,
		contactInboxRepo: contactInboxRepo,
	}
}

func (h *ContactHandler) CreateContact(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelApi)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateContactReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	var additionalAttrs *string
	if len(req.CustomAttributes) > 0 {
		s := string(req.CustomAttributes)
		additionalAttrs = &s
	}

	contact := &model.Contact{
		AccountID:       accountID,
		Name:            req.Name,
		Email:           req.Email,
		PhoneNumber:     req.Phone,
		Identifier:      req.Identifier,
		AdditionalAttrs: additionalAttrs,
	}

	created, err := h.svc.Create(c.Context(), accountID, contact)
	if err != nil {
		return handleNotFound(c, err)
	}

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	if err := h.svc.EnsureContactInbox(c.Context(), created.ID, inbox.ID, req.SourceID); err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(created)))
}

func (h *ContactHandler) GetContact(c *fiber.Ctx) error {
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

	contact, err := h.svc.FindBySourceID(c.Context(), sourceID, inbox.ID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
}

func (h *ContactHandler) UpdateContact(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelApi)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "channel api not found"))
	}

	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.UpdateContactReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	sourceID := c.Params("sourceId")

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	contact, err := h.svc.FindBySourceID(c.Context(), sourceID, inbox.ID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	updated, err := h.svc.Update(c.Context(), contact.ID, accountID, req.Name, req.Email, req.Phone)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(updated)))
}

func (h *ContactHandler) Search(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	query := c.Query("q", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	contacts, total, err := h.svc.Search(c.Context(), accountID, query, page, perPage)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.ContactsToResp(contacts),
	}))
}

func (h *ContactHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	contact, err := h.svc.FindByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
}
