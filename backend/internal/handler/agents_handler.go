package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/audit"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type AgentHandler struct {
	svc            *service.AgentService
	invitationRepo *repo.AgentInvitationRepo
	audit          *audit.Logger
}

func NewAgentHandler(svc *service.AgentService, invitationRepo *repo.AgentInvitationRepo, auditLogger *audit.Logger) *AgentHandler {
	return &AgentHandler{svc: svc, invitationRepo: invitationRepo, audit: auditLogger}
}

func (h *AgentHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	members, err := h.svc.List(c.Context(), accountID)
	if err != nil {
		logger.Error().Str("component", "agents").Err(err).Msg("failed to list agents")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list agents"))
	}

	resp := make([]dto.AgentResp, 0, len(members))
	for _, m := range members {
		resp = append(resp, dto.AgentResp{
			ID:         m.ID,
			UserID:     m.UserID,
			Name:       m.Name,
			Email:      m.Email,
			Role:       m.Role,
			Status:     "active",
			LastActive: m.LastActiveAt,
			CreatedAt:  m.CreatedAt,
		})
	}

	invitations, err := h.invitationRepo.ListByAccount(c.Context(), accountID)
	if err != nil {
		logger.Error().Str("component", "agents").Err(err).Msg("failed to list invitations")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list invitations"))
	}

	now := time.Now().UTC()
	for i := range invitations {
		inv := &invitations[i]
		if inv.ConsumedAt != nil || now.After(inv.ExpiresAt) {
			continue
		}
		name := inv.Email
		if inv.Name != nil && *inv.Name != "" {
			name = *inv.Name
		}
		resp = append(resp, dto.AgentResp{
			ID:        inv.ID,
			UserID:    0,
			Name:      name,
			Email:     inv.Email,
			Role:      int(inv.Role),
			Status:    "invited",
			CreatedAt: inv.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(dto.SuccessResp(resp))
}

func (h *AgentHandler) Invite(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	var req dto.InviteAgentReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.svc.Invite(c.Context(), accountID, req.Email, req.Role, req.Name, authUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationAlreadyPending):
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "invitation_already_pending"))
		default:
			logger.Error().Str("component", "agents").Err(err).Msg("failed to invite agent")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to invite agent"))
		}
	}

	h.audit.LogFromCtx(c, "user.invited", "agent_invitation", &result.InvitationID, fiber.Map{"email": req.Email, "role": req.Role})

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(fiber.Map{
		"invitation_id": result.InvitationID,
		"status":        result.Status,
	}))
}

func (h *AgentHandler) AcceptInvitation(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "missing token"))
	}

	var req dto.AcceptInvitationReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.svc.AcceptInvitation(c.Context(), token, req.Password, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationNotFound):
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "invitation_not_found"))
		case errors.Is(err, service.ErrInvitationExpired):
			return c.Status(fiber.StatusGone).JSON(dto.ErrorResp("Gone", "invitation_expired"))
		case errors.Is(err, service.ErrInvitationAlreadyUsed):
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "invitation_already_used"))
		default:
			logger.Error().Str("component", "agents").Err(err).Msg("failed to accept invitation")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to accept invitation"))
		}
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"user":         userToResp(result.User),
		"account":      accountToResp(result.Account),
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
	}))
}

func (h *AgentHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}

	var req dto.UpdateAgentReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	if err := h.svc.UpdateAgent(c.Context(), accountID, userID, req.Role); err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDemoteLastOwner):
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "cannot_demote_last_owner"))
		default:
			logger.Error().Str("component", "agents").Err(err).Msg("failed to update agent")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update agent"))
		}
	}

	if req.Role != nil {
		h.audit.LogFromCtx(c, "user.role_changed", "user", &userID, fiber.Map{"role": *req.Role})
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"result": "success"}))
}

func (h *AgentHandler) Remove(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}

	if err := h.svc.RemoveAgent(c.Context(), accountID, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDemoteLastOwner):
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "cannot_remove_last_owner"))
		default:
			logger.Error().Str("component", "agents").Err(err).Msg("failed to remove agent")
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to remove agent"))
		}
	}

	h.audit.LogFromCtx(c, "agent.removed", "user", &userID, nil)

	return c.SendStatus(fiber.StatusNoContent)
}
