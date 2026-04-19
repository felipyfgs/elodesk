package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type LabelsHandler struct {
	svc *service.LabelsService
}

func NewLabelsHandler(svc *service.LabelsService) *LabelsHandler {
	return &LabelsHandler{svc: svc}
}

func (h *LabelsHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	labels, err := h.svc.List(c.Context(), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.LabelsToResp(labels)))
}

func (h *LabelsHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	color := "#1f93ff"
	if req.Color != "" {
		color = req.Color
	}
	showOnSidebar := false
	if req.ShowOnSidebar != nil {
		showOnSidebar = *req.ShowOnSidebar
	}

	label, err := h.svc.Create(c.Context(), accountID, req.Title, color, req.Description, showOnSidebar)
	if err != nil {
		return handleLabelError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.LabelToResp(label)))
}

func (h *LabelsHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid label id"))
	}

	var req dto.UpdateLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	label, err := h.svc.Update(c.Context(), id, accountID, req.Title, req.Color, req.Description, req.ShowOnSidebar)
	if err != nil {
		return handleLabelError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.LabelToResp(label)))
}

func (h *LabelsHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid label id"))
	}

	if err := h.svc.Delete(c.Context(), id, accountID); err != nil {
		return handleLabelError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *LabelsHandler) ApplyToConversation(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req dto.ApplyLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.ApplyLabel(c.Context(), accountID, req.LabelID, "conversation", conversationID); err != nil {
		return handleLabelError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *LabelsHandler) RemoveFromConversation(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	labelID, err := strconv.ParseInt(c.Params("labelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid label id"))
	}

	if err := h.svc.RemoveLabel(c.Context(), accountID, labelID, "conversation", conversationID); err != nil {
		return handleLabelError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *LabelsHandler) ListConversationLabels(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	conversationID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	labels, err := h.svc.ListByTaggable(c.Context(), accountID, "conversation", conversationID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.LabelsToResp(labels)))
}

func (h *LabelsHandler) ApplyToContact(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	var req dto.ApplyLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.ApplyLabel(c.Context(), accountID, req.LabelID, "contact", contactID); err != nil {
		return handleLabelError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *LabelsHandler) RemoveFromContact(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	labelID, err := strconv.ParseInt(c.Params("labelId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid label id"))
	}

	if err := h.svc.RemoveLabel(c.Context(), accountID, labelID, "contact", contactID); err != nil {
		return handleLabelError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *LabelsHandler) ListContactLabels(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	labels, err := h.svc.ListByTaggable(c.Context(), accountID, "contact", contactID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.LabelsToResp(labels)))
}

func handleLabelError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "label not found"))
	case err == service.ErrLabelTitleTaken:
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "label_title_taken"))
	case err == service.ErrInvalidLabelColor:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid hex color"))
	default:
		logger.Error().Str("component", "labels").Err(err).Msg("labels service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
