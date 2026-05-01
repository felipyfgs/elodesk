package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type MacroHandler struct {
	svc         *service.MacroService
	auditLogger *audit.Logger
}

func NewMacroHandler(svc *service.MacroService, auditLogger *audit.Logger) *MacroHandler {
	return &MacroHandler{svc: svc, auditLogger: auditLogger}
}

func (h *MacroHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	macros, err := h.svc.List(c.Context(), accountID)
	if err != nil {
		logger.Error().Str("component", "macros").Err(err).Msg("failed to list macros")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list macros"))
	}
	return c.JSON(dto.SuccessResp(dto.MacrosToResp(macros)))
}

func (h *MacroHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	var req dto.CreateMacroReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	macro, err := h.svc.Create(c.Context(), accountID, authUser.ID, req.Name, req.Visibility, req.Conditions, req.Actions)
	if err != nil {
		return mapMacroError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.MacroToResp(macro)))
}

func (h *MacroHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid macro id"))
	}
	macro, err := h.svc.Get(c.Context(), accountID, id)
	if err != nil {
		return mapMacroError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.MacroToResp(macro)))
}

func (h *MacroHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid macro id"))
	}
	var req dto.UpdateMacroReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	macro, err := h.svc.Update(c.Context(), accountID, id, service.UpdateMacroInput{
		Name:       req.Name,
		Visibility: req.Visibility,
		Conditions: req.Conditions,
		Actions:    req.Actions,
	})
	if err != nil {
		return mapMacroError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.MacroToResp(macro)))
}

func (h *MacroHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid macro id"))
	}
	if err := h.svc.Delete(c.Context(), accountID, id); err != nil {
		return mapMacroError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *MacroHandler) Apply(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}
	convID, err := strconv.ParseInt(c.Params("convId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}
	macroID, err := strconv.ParseInt(c.Params("macroId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid macro id"))
	}

	result, err := h.svc.Apply(c.Context(), accountID, convID, macroID, authUser.ID)
	if err != nil {
		if errors.Is(err, service.ErrMacroInvalidAction) || errors.Is(err, service.ErrMacroActionFailed) {
			failedIdx := -1
			if result != nil {
				failedIdx = result.FailedIndex
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":               "macro_execution_failed",
				"message":             err.Error(),
				"failed_action_index": failedIdx,
			})
		}
		return mapMacroError(c, err)
	}

	macroIDPtr := macroID
	if h.auditLogger != nil {
		h.auditLogger.LogFromCtx(c, "macro.executed", "macro", &macroIDPtr, fiber.Map{
			"conversation_id":  convID,
			"executed_actions": result.ExecutedActions,
		})
	}
	return c.JSON(dto.SuccessResp(fiber.Map{"executedActions": result.ExecutedActions}))
}

func mapMacroError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "macro_not_found"))
	case errors.Is(err, service.ErrMacroInvalidAction):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_action",
			"message": err.Error(),
			"allowed": []string{"assign_agent", "assign_team", "add_label", "remove_label", "change_status", "snooze_until", "send_message", "add_note"},
		})
	default:
		logger.Error().Str("component", "macros").Err(err).Msg("macro service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
