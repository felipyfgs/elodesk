package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/repo"
	"backend/internal/service"
)

type ParticipantHandler struct {
	svc *service.ParticipantService
}

func NewParticipantHandler(svc *service.ParticipantService) *ParticipantHandler {
	return &ParticipantHandler{svc: svc}
}

func (h *ParticipantHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	convIDStr := c.Params("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	rows, err := h.svc.List(c.Context(), accountID, convID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list participants"))
	}

	out := make([]dto.ParticipantResp, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.ParticipantResp{
			ID:      r.ID,
			Role:    r.Role,
			Contact: dto.ContactToResp(&r.Contact),
		})
	}
	return c.JSON(dto.ParticipantListResp{Data: out})
}

func (h *ParticipantHandler) Sync(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	convIDStr := c.Params("id")
	convID, err := strconv.ParseInt(convIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid conversation id"))
	}

	var req struct {
		Members []repo.Member `json:"members"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	if err := h.svc.SyncMembers(c.Context(), accountID, convID, req.Members); err != nil {
		return handleError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}
