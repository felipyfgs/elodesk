package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"backend/internal/crypto"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/middleware"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type ContactHandler struct {
	svc              *service.ContactService
	inboxRepo        *repo.InboxRepo
	contactInboxRepo *repo.ContactInboxRepo
	cipher           *crypto.Cipher
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

func (h *ContactHandler) SetCipher(c *crypto.Cipher) {
	h.cipher = c
}

func currentUserID(c *fiber.Ctx) *int64 {
	if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
		id := u.ID
		return &id
	}
	return nil
}

func (h *ContactHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	if err := h.svc.Delete(c.Context(), accountID, int64(id), currentUserID(c)); err != nil {
		return handleNotFound(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ContactHandler) Merge(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	childID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	var req dto.ContactMergeReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if req.PrimaryContactID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "primary_contact_id is required"))
	}
	primary, err := h.svc.Merge(c.Context(), accountID, int64(childID), req.PrimaryContactID, currentUserID(c))
	if err != nil {
		if errors.Is(err, service.ErrSameContactMerge) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable", "same_contact_merge"))
		}
		return handleNotFound(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.ContactToResp(primary)))
}

func (h *ContactHandler) Block(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	var req dto.ContactBlockReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.svc.SetBlocked(c.Context(), accountID, int64(id), currentUserID(c), req.Blocked); err != nil {
		return handleNotFound(c, err)
	}
	contact, err := h.svc.FindByID(c.Context(), int64(id), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
}

func (h *ContactHandler) SetAvatar(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	var req dto.ContactAvatarReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if req.ObjectKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "object_key is required"))
	}
	contact, err := h.svc.SetAvatar(c.Context(), accountID, int64(id), currentUserID(c), req.ObjectKey)
	if err != nil {
		if errors.Is(err, service.ErrInvalidAvatarObject) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable", "invalid_object_key"))
		}
		return handleNotFound(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
}

func (h *ContactHandler) DeleteAvatar(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	if err := h.svc.DeleteAvatar(c.Context(), accountID, int64(id), currentUserID(c)); err != nil {
		return handleNotFound(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ContactHandler) Events(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "25"))
	events, total, err := h.svc.ListEvents(c.Context(), accountID, int64(id), page, pageSize)
	if err != nil {
		return handleNotFound(c, err)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	return c.JSON(dto.SuccessResp(dto.AuditEventListResp{
		Meta:    dto.NewMetaResp(total, page, pageSize),
		Payload: events,
	}))
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

	inbox, err := h.inboxRepo.FindByChannelID(c.Context(), channelApi.ID)
	if err != nil {
		return handleNotFound(c, err)
	}

	hmacVerified := false
	if req.IdentifierHash != nil && *req.IdentifierHash != "" {
		if h.cipher != nil {
			identifier := ""
			if req.Identifier != nil {
				identifier = *req.Identifier
			}
			if middleware.ValidIdentifierHash(channelApi, h.cipher, identifier, *req.IdentifierHash) {
				hmacVerified = true
			} else if channelApi.HmacMandatory {
				return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "HMAC failed: Invalid Identifier Hash Provided"))
			}
		}
	} else if channelApi.HmacMandatory {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "HMAC failed: Invalid Identifier Hash Provided"))
	}

	attrs := service.ContactCreateAttrs{
		Name:        req.Name,
		Email:       req.Email,
		PhoneNumber: req.Phone,
		Identifier:  req.Identifier,
		AvatarURL:   req.AvatarURL,
	}
	if len(req.AdditionalAttributes) > 0 && string(req.AdditionalAttributes) != "null" {
		if !json.Valid(req.AdditionalAttributes) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "additional_attributes must be valid JSON"))
		}
		s := string(req.AdditionalAttributes)
		attrs.AdditionalAttrs = &s
	}

	ci, err := h.svc.CreateOrReuseContactInbox(c.Context(), inbox, attrs, req.SourceID, hmacVerified)
	if err != nil {
		return handleNotFound(c, err)
	}

	contact, err := h.svc.FindByID(c.Context(), ci.ContactID, accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.ContactToResp(contact)))
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

	identifierHash := c.Get("X-Contact-Identifier-Hash")
	if identifierHash != "" && h.cipher != nil {
		identifier := ""
		if contact.Identifier != nil {
			identifier = *contact.Identifier
		}
		if middleware.ValidIdentifierHash(channelApi, h.cipher, identifier, identifierHash) {
			ci, err := h.contactInboxRepo.FindBySourceID(c.Context(), sourceID, inbox.ID)
			if err == nil {
				_ = h.contactInboxRepo.UpdateHmacVerified(c.Context(), ci.ID, true)
			}
		}
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

	identifierHash := c.Get("X-Contact-Identifier-Hash")
	if identifierHash == "" {
		var bodyReq struct {
			IdentifierHash *string `json:"identifier_hash"`
		}
		if err := c.BodyParser(&bodyReq); err == nil && bodyReq.IdentifierHash != nil {
			identifierHash = *bodyReq.IdentifierHash
		}
	}

	if identifierHash != "" && h.cipher != nil {
		identifier := ""
		if contact.Identifier != nil {
			identifier = *contact.Identifier
		}
		if middleware.ValidIdentifierHash(channelApi, h.cipher, identifier, identifierHash) {
			ci, err := h.contactInboxRepo.FindBySourceID(c.Context(), sourceID, inbox.ID)
			if err == nil {
				_ = h.contactInboxRepo.UpdateHmacVerified(c.Context(), ci.ID, true)
			}
		} else if channelApi.HmacMandatory {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "HMAC failed: Invalid Identifier Hash Provided"))
		}
	}

	params := service.ContactIdentifyParams{
		Email:       req.Email,
		PhoneNumber: req.Phone,
		AvatarURL:   req.AvatarURL,
	}

	updated, err := h.svc.Identify(c.Context(), accountID, contact, params)
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

func (h *ContactHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req struct {
		Name                 string          `json:"name"`
		Email                *string         `json:"email,omitempty"`
		PhoneNumber          *string         `json:"phone_number,omitempty"`
		Identifier           *string         `json:"identifier,omitempty"`
		AdditionalAttributes json.RawMessage `json:"additional_attributes,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "name is required"))
	}

	var additionalAttrs *string
	if len(req.AdditionalAttributes) > 0 && string(req.AdditionalAttributes) != "null" {
		if !json.Valid(req.AdditionalAttributes) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "additional_attributes must be valid JSON"))
		}
		s := string(req.AdditionalAttributes)
		additionalAttrs = &s
	}

	contact := &model.Contact{
		AccountID:       accountID,
		Name:            req.Name,
		Email:           req.Email,
		PhoneNumber:     req.PhoneNumber,
		Identifier:      req.Identifier,
		AdditionalAttrs: additionalAttrs,
	}

	created, err := h.svc.Create(c.Context(), accountID, contact)
	if err != nil {
		logger.Error().Str("component", "contacts").Err(err).Msg("failed to create contact")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create contact"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.ContactToResp(created)))
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
		Name                 *string         `json:"name,omitempty"`
		Email                *string         `json:"email,omitempty"`
		PhoneNumber          *string         `json:"phone_number,omitempty"`
		Identifier           *string         `json:"identifier,omitempty"`
		AdditionalAttributes json.RawMessage `json:"additional_attributes,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	var additionalAttrs map[string]any
	if len(req.AdditionalAttributes) > 0 && string(req.AdditionalAttributes) != "null" {
		if err := json.Unmarshal(req.AdditionalAttributes, &additionalAttrs); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "additional_attributes must be a JSON object"))
		}
	}

	updated, err := h.svc.UpdateDetails(c.Context(), int64(id), accountID, req.Name, req.Email, req.PhoneNumber, additionalAttrs)
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

	return c.JSON(dto.SuccessResp(convos))
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
