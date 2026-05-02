package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type PipelineHandler struct {
	svc *service.PipelineService
}

func NewPipelineHandler(svc *service.PipelineService) *PipelineHandler {
	return &PipelineHandler{svc: svc}
}

// ===== Templates =====

func (h *PipelineHandler) ListTemplates(c *fiber.Ctx) error {
	return c.JSON(dto.SuccessResp(service.ListTemplates()))
}

// ===== Pipelines =====

func (h *PipelineHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	includeArchived := c.Query("archived") == "true"
	items, err := h.svc.List(c.Context(), accountID, includeArchived)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.PipelinesToResp(items)))
}

func (h *PipelineHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid pipeline id"))
	}
	p, stages, err := h.svc.Get(c.Context(), id, accountID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.PipelineToResp(p, stages)))
}

func (h *PipelineHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	var req dto.CreatePipelineReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	p, stages, err := h.svc.Create(c.Context(), accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.PipelineToResp(p, stages)))
}

func (h *PipelineHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid pipeline id"))
	}
	var req dto.UpdatePipelineReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	p, err := h.svc.Update(c.Context(), id, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.PipelineToResp(p, nil)))
}

func (h *PipelineHandler) Archive(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid pipeline id"))
	}
	if err := h.svc.Archive(c.Context(), id, accountID, userID); err != nil {
		return handlePipelineError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ===== Stages =====

func (h *PipelineHandler) CreateStage(c *fiber.Ctx) error {
	accountID, pipelineID, err := parsePipelineParam(c)
	if err != nil {
		return nil
	}
	userID := userIDFromCtx(c)
	var req dto.CreateStageReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	stage, err := h.svc.CreateStage(c.Context(), pipelineID, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.StageToResp(stage)))
}

func (h *PipelineHandler) UpdateStage(c *fiber.Ctx) error {
	accountID, pipelineID, err := parsePipelineParam(c)
	if err != nil {
		return nil
	}
	userID := userIDFromCtx(c)
	stageID, err := strconv.ParseInt(c.Params("sid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid stage id"))
	}
	var req dto.UpdateStageReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	stage, _, err := h.svc.UpdateStage(c.Context(), pipelineID, stageID, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.StageToResp(stage)))
}

func (h *PipelineHandler) DeleteStage(c *fiber.Ctx) error {
	accountID, pipelineID, err := parsePipelineParam(c)
	if err != nil {
		return nil
	}
	userID := userIDFromCtx(c)
	stageID, err := strconv.ParseInt(c.Params("sid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid stage id"))
	}
	if err := h.svc.DeleteStage(c.Context(), pipelineID, stageID, accountID, userID); err != nil {
		return handlePipelineError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ===== Cards =====

func (h *PipelineHandler) ListCards(c *fiber.Ctx) error {
	accountID, pipelineID, err := parsePipelineParam(c)
	if err != nil {
		return nil
	}
	rels, err := h.svc.ListCards(c.Context(), pipelineID, accountID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	out := make([]dto.CardResp, len(rels))
	for i := range rels {
		out[i] = dto.CardToResp(&rels[i].Card, rels[i].AssigneeIDs, rels[i].LabelIDs)
	}
	return c.JSON(dto.SuccessResp(out))
}

func (h *PipelineHandler) CreateCard(c *fiber.Ctx) error {
	accountID, pipelineID, err := parsePipelineParam(c)
	if err != nil {
		return nil
	}
	userID := userIDFromCtx(c)
	var req dto.CreateCardReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	rel, err := h.svc.CreateCard(c.Context(), pipelineID, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) GetCard(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	rel, err := h.svc.GetCard(c.Context(), cardID, accountID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) UpdateCard(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	var req dto.UpdateCardReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	rel, err := h.svc.UpdateCard(c.Context(), cardID, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) MoveCard(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	var req dto.MoveCardReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	rel, err := h.svc.MoveCard(c.Context(), cardID, accountID, userID, req)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) DeleteCard(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	if err := h.svc.DeleteCard(c.Context(), cardID, accountID, userID); err != nil {
		return handlePipelineError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *PipelineHandler) AddAssignee(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	var req dto.AssignAgentReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	rel, err := h.svc.AssignAgent(c.Context(), cardID, accountID, userID, req.UserID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) RemoveAssignee(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	targetUserID, err := strconv.ParseInt(c.Params("uid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}
	rel, err := h.svc.UnassignAgent(c.Context(), cardID, accountID, userID, targetUserID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) ApplyLabel(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	var req dto.ApplyCardLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	rel, err := h.svc.ApplyLabel(c.Context(), cardID, accountID, userID, req.LabelID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

func (h *PipelineHandler) RemoveLabel(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	userID := userIDFromCtx(c)
	cardID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid card id"))
	}
	labelID, err := strconv.ParseInt(c.Params("lid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid label id"))
	}
	rel, err := h.svc.RemoveLabel(c.Context(), cardID, accountID, userID, labelID)
	if err != nil {
		return handlePipelineError(c, err)
	}
	return c.JSON(dto.SuccessResp(dto.CardToResp(&rel.Card, rel.AssigneeIDs, rel.LabelIDs)))
}

// ===== helpers =====

func parsePipelineParam(c *fiber.Ctx) (accountID, pipelineID int64, err error) {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		_ = c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
		return 0, 0, errResponseSent
	}
	pipelineID, err = strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid pipeline id"))
		return 0, 0, errResponseSent
	}
	return accountID, pipelineID, nil
}

func userIDFromCtx(c *fiber.Ctx) *int64 {
	if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
		id := u.ID
		return &id
	}
	return nil
}

func handlePipelineError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "resource not found"))
	case errors.Is(err, service.ErrStageHasCards):
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "cannot delete stage with existing cards"))
	case errors.Is(err, service.ErrUnknownTemplate):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "unknown template_key"))
	case errors.Is(err, service.ErrTemplateOrStagesRequired):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "either template_key or stages must be provided"))
	case errors.Is(err, service.ErrStageBelongsOtherPipeline):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "stage belongs to a different pipeline"))
	case errors.Is(err, service.ErrCardKindLinkMismatch):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "linked_entity_type does not match pipeline card_kind"))
	case errors.Is(err, service.ErrLinkedEntityRequired):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "linked_entity_id is required for this card kind"))
	case errors.Is(err, service.ErrLinkedEntityForbidden):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "card_kind does not accept linked_entity"))
	case errors.Is(err, service.ErrPipelineUserNotInAccount):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "user does not belong to this account"))
	case errors.Is(err, service.ErrLabelNotInAccount):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResp("Unprocessable Entity", "label does not belong to this account"))
	default:
		logger.Error().Str("component", "pipelines").Err(err).Msg("pipeline service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
