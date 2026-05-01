package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type CannedResponseHandler struct {
	svc *service.CannedResponseService
}

func NewCannedResponseHandler(svc *service.CannedResponseService) *CannedResponseHandler {
	return &CannedResponseHandler{svc: svc}
}

func (h *CannedResponseHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	search := c.Query("search", "")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	items, err := h.svc.List(c.Context(), accountID, search, limit)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CannedResponsesToResp(items)))
}

func (h *CannedResponseHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateCannedResponseReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	item, err := h.svc.Create(c.Context(), accountID, req.ShortCode, req.Content)
	if err != nil {
		return handleCannedError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CannedResponseToResp(item)))
}

func (h *CannedResponseHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid canned response id"))
	}

	var req dto.UpdateCannedResponseReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	item, err := h.svc.Update(c.Context(), id, accountID, req.ShortCode, req.Content)
	if err != nil {
		return handleCannedError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.CannedResponseToResp(item)))
}

func (h *CannedResponseHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid canned response id"))
	}

	if err := h.svc.Delete(c.Context(), id, accountID); err != nil {
		return handleCannedError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func handleCannedError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "canned response not found"))
	case err == service.ErrCannedShortCodeTaken:
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "canned_short_code_taken"))
	default:
		logger.Error().Str("component", "canned_responses").Err(err).Msg("canned responses service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
