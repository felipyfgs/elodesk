package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/repo"
	"backend/internal/service"
)

type ForwardHandler struct {
	forwardSvc *service.ForwardService
}

func NewForwardHandler(forwardSvc *service.ForwardService) *ForwardHandler {
	return &ForwardHandler{forwardSvc: forwardSvc}
}

func (h *ForwardHandler) Forward(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "user not found"))
	}

	var req dto.ForwardMessagesReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	targets := make([]service.ForwardTarget, 0, len(req.Targets))
	for _, t := range req.Targets {
		if t.ConversationID != nil && *t.ConversationID > 0 {
			targets = append(targets, service.ForwardTarget{
				ConversationID: *t.ConversationID,
			})
		} else if t.ContactID != nil && t.InboxID != nil && *t.ContactID > 0 && *t.InboxID > 0 {
			targets = append(targets, service.ForwardTarget{
				ContactID: *t.ContactID,
				InboxID:   *t.InboxID,
			})
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request",
				"each target must specify conversation_id or (contact_id + inbox_id)"))
		}
	}

	results, err := h.forwardSvc.ForwardMessages(c.Context(), accountID, user.ID, req.SourceMessageIDs, targets)
	if err != nil {
		if errors.Is(err, service.ErrForwardLimitExceeded) ||
			errors.Is(err, service.ErrForwardTargetsLimit) ||
			errors.Is(err, service.ErrForwardEmptySource) ||
			errors.Is(err, service.ErrForwardNoTargets) ||
			errors.Is(err, service.ErrForwardInvalidTarget) ||
			errors.Is(err, service.ErrForwardIncompatibleTarget) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation", err.Error()))
		}
		if errors.Is(err, repo.ErrMessageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "one or more source messages not found"))
		}
		return handleNotFound(c, err)
	}

	respResults := make([]dto.ForwardResultResp, 0, len(results))
	for _, r := range results {
		resp := dto.ForwardResultResp{
			Target: dto.ForwardTargetReq{
				ConversationID: &r.Target.ConversationID,
				ContactID:      &r.Target.ContactID,
				InboxID:        &r.Target.InboxID,
			},
			Status:              r.Status,
			CreatedMessageIDs:   r.CreatedMessageIDs,
			CreatedConversation: r.CreatedConversation,
		}
		if r.ConversationID > 0 {
			resp.ConversationID = &r.ConversationID
		}
		if r.Err != nil {
			errStr := r.Err.Error()
			resp.Error = &errStr
		}
		respResults = append(respResults, resp)
	}

	return c.JSON(dto.SuccessResp(dto.ForwardMessagesResp{Results: respResults}))
}
