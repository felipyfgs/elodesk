package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
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
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
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

	contact, err := h.svc.FindBySourceID(c.Context(), sourceID, inbox.ID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
}

func (h *ContactHandler) UpdateContact(c *fiber.Ctx) error {
	channelApi, ok := c.Locals("channelApi").(*model.ChannelAPI)
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

	query := c.Query("search", c.Query("q", ""))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("pageSize", c.Query("per_page", "25")))
	if perPage > 100 {
		perPage = 100
	}
	if perPage < 1 {
		perPage = 25
	}

	labels := c.Query("labels")
	var labelList []string
	if labels != "" {
		labelList = strings.Split(labels, ",")
	}

	contacts, total, err := h.svc.SearchWithLabels(c.Context(), accountID, query, labelList, page, perPage)
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

func (h *ContactHandler) UpdateContactByID(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Email       *string `json:"email,omitempty"`
		PhoneNumber *string `json:"phone_number,omitempty"`
		Identifier  *string `json:"identifier,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	updated, err := h.svc.Update(c.Context(), int64(id), accountID, req.Name, req.Email, req.PhoneNumber)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(updated)))
}

func (h *ContactHandler) ListContactConversations(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	convos, err := h.svc.FindConversations(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ConversationsToResp(convos)))
}

func (h *ContactHandler) Import(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	body := c.Body()
	if len(body) > 10*1024*1024 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(dto.ErrorResp("Too Large", "file must be under 10 MB"))
	}

	parsed, err := ParseContactCSV(string(body))
	if err != nil {
		if errors.Is(err, ErrMissingRequiredColumn) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "CSV must have at least 'name' or 'email' column"))
		}
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "failed to parse CSV"))
	}

	const batchSize = 500
	var resp dto.ContactImportResp
	importErrors := parsed.Errors
	processed := 0

	for i := 0; i < len(parsed.Contacts); i += batchSize {
		end := i + batchSize
		if end > len(parsed.Contacts) {
			end = len(parsed.Contacts)
		}
		batch := parsed.Contacts[i:end]

		result, err := h.svc.ImportBatch(c.Context(), accountID, batch)
		if err != nil {
			logger.Error().Str("component", "handler").Err(err).Msg("contact import batch error")
			for j := range batch {
				importErrors = append(importErrors, dto.ImportError{Row: i + j + 2, Reason: "batch insert failed"})
			}
		} else {
			resp.Inserted += result.Inserted
			resp.Updated += result.Updated
		}
		processed += len(batch)
	}

	resp.TotalRows = processed
	if len(importErrors) > 0 {
		resp.Errors = importErrors
	}

	return c.JSON(dto.SuccessResp(resp))
}
