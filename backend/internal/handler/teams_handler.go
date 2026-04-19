package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type TeamsHandler struct {
	svc *service.TeamsService
}

func NewTeamsHandler(svc *service.TeamsService) *TeamsHandler {
	return &TeamsHandler{svc: svc}
}

func (h *TeamsHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	teams, err := h.svc.List(c.Context(), accountID)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.TeamsToResp(teams)))
}

func (h *TeamsHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	var req dto.CreateTeamReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	allowAutoAssign := false
	if req.AllowAutoAssign != nil {
		allowAutoAssign = *req.AllowAutoAssign
	}

	team, err := h.svc.Create(c.Context(), accountID, req.Name, req.Description, allowAutoAssign)
	if err != nil {
		return handleTeamError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.TeamToResp(team)))
}

func (h *TeamsHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid team id"))
	}

	var req dto.UpdateTeamReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	team, err := h.svc.Update(c.Context(), id, accountID, req.Name, req.Description, req.AllowAutoAssign)
	if err != nil {
		return handleTeamError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.TeamToResp(team)))
}

func (h *TeamsHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid team id"))
	}

	if err := h.svc.Delete(c.Context(), id, accountID); err != nil {
		return handleTeamError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func (h *TeamsHandler) ListMembers(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid team id"))
	}

	members, err := h.svc.ListMembers(c.Context(), accountID, id)
	if err != nil {
		return handleTeamError(c, err)
	}

	return c.JSON(dto.SuccessResp(members))
}

func (h *TeamsHandler) AddMembers(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid team id"))
	}

	var req dto.AddTeamMembersReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	members, err := h.svc.AddMembers(c.Context(), accountID, id, req.UserIDs)
	if err != nil {
		return handleTeamError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(members))
}

func (h *TeamsHandler) RemoveMembers(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid team id"))
	}

	var req dto.RemoveTeamMembersReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.RemoveMembers(c.Context(), accountID, id, req.UserIDs); err != nil {
		return handleTeamError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func handleTeamError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "team not found"))
	case err == service.ErrTeamNameTaken:
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "team_name_taken"))
	case err == service.ErrUserNotInAccount:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "user_not_in_account"))
	default:
		logger.Error().Str("component", "teams").Err(err).Msg("teams service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
