package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type CustomAttributeHandler struct {
	svc *service.CustomAttributeService
}

func NewCustomAttributeHandler(svc *service.CustomAttributeService) *CustomAttributeHandler {
	return &CustomAttributeHandler{svc: svc}
}

func (h *CustomAttributeHandler) ListDefinitions(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	attrModel := c.Query("attribute_model", "")

	defs, err := h.svc.ListDefinitions(c.Context(), accountID, attrModel)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CustomAttrDefsToResp(defs)))
}

func (h *CustomAttributeHandler) CreateDefinition(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateCustomAttributeDefinitionReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	m := &model.CustomAttributeDefinition{
		AccountID:            accountID,
		AttributeKey:         req.AttributeKey,
		AttributeDisplayName: req.AttributeDisplayName,
		AttributeDisplayType: req.AttributeDisplayType,
		AttributeModel:       req.AttributeModel,
		AttributeDescription: req.AttributeDescription,
		RegexPattern:         req.RegexPattern,
		DefaultValue:         req.DefaultValue,
	}
	if len(req.AttributeValues) > 0 {
		s := string(req.AttributeValues)
		m.AttributeValues = &s
	}

	def, err := h.svc.CreateDefinition(c.Context(), accountID, m)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CustomAttrDefToResp(def)))
}

func (h *CustomAttributeHandler) UpdateDefinition(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid id"))
	}

	var req dto.UpdateCustomAttributeDefinitionReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	m := &model.CustomAttributeDefinition{}
	if req.AttributeKey != nil {
		m.AttributeKey = *req.AttributeKey
	}
	if req.AttributeDisplayName != nil {
		m.AttributeDisplayName = *req.AttributeDisplayName
	}
	if req.AttributeDisplayType != nil {
		m.AttributeDisplayType = *req.AttributeDisplayType
	}
	if req.AttributeModel != nil {
		m.AttributeModel = *req.AttributeModel
	}
	if len(req.AttributeValues) > 0 {
		s := string(req.AttributeValues)
		m.AttributeValues = &s
	}
	m.AttributeDescription = req.AttributeDescription
	m.RegexPattern = req.RegexPattern
	m.DefaultValue = req.DefaultValue

	def, err := h.svc.UpdateDefinition(c.Context(), id, accountID, m)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CustomAttrDefToResp(def)))
}

func (h *CustomAttributeHandler) DeleteDefinition(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid id"))
	}

	if err := h.svc.DeleteDefinition(c.Context(), id, accountID); err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *CustomAttributeHandler) SetContactAttributes(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	var values dto.SetCustomAttributesReq
	if err := c.BodyParser(&values); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	result, err := h.svc.SetContactAttributes(c.Context(), contactID, accountID, values)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(result))
}

func (h *CustomAttributeHandler) RemoveContactAttributes(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	var req dto.RemoveCustomAttributesReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.svc.RemoveContactAttributes(c.Context(), contactID, accountID, req.Keys)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(result))
}

func (h *CustomAttributeHandler) SetConversationAttributes(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var values dto.SetCustomAttributesReq
	if err := c.BodyParser(&values); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	result, err := h.svc.SetConversationAttributes(c.Context(), conversationID, accountID, values)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(result))
}

func (h *CustomAttributeHandler) RemoveConversationAttributes(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req dto.RemoveCustomAttributesReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.svc.RemoveConversationAttributes(c.Context(), conversationID, accountID, req.Keys)
	if err != nil {
		return handleCustomAttrError(c, err)
	}

	return c.JSON(dto.SuccessResp(result))
}

func handleCustomAttrError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "resource not found"))
	case err == service.ErrAttributeKeyReserved:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "attribute_key_reserved"))
	case err == service.ErrListValuesRequired:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "list_values_required"))
	case err == service.ErrUnknownAttributeKey:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "unknown_attribute_key"))
	case err == service.ErrValueNotInList:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "value_not_in_list"))
	case err == service.ErrInvalidAttributeValue:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid_attribute_value"))
	default:
		logger.Error().Str("component", "custom_attributes").Err(err).Msg("custom attributes service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
