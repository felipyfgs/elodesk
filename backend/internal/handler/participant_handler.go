package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/repo"
	"backend/internal/service"
)

// ParticipantHandler exposes the conversation participants endpoint
// (GET /api/v1/accounts/:aid/conversations/:id/participants). Used by the
// frontend to render the group members list and by integrations (Wzap) to
// sync the roster when a WhatsApp group changes.
type ParticipantHandler struct {
	svc *service.ParticipantService
}

func NewParticipantHandler(svc *service.ParticipantService) *ParticipantHandler {
	return &ParticipantHandler{svc: svc}
}

// List handles GET /api/v1/accounts/:aid/conversations/:id/participants.
// Returns an empty array (not 404) for 1:1 conversations with no participants.
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

// Sync handles POST /api/v1/accounts/:aid/conversations/:id/participants/sync.
// Accepts a list of members to reconcile against the conversation's current
// participants. Used by Wzap when a WhatsApp group's roster changes.
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
