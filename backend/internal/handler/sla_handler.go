package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type SLAHandler struct {
	svc *service.SLAService
}

func NewSLAHandler(svc *service.SLAService) *SLAHandler {
	return &SLAHandler{svc: svc}
}

func (h *SLAHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	slas, bindings, err := h.svc.List(c.Context(), accountID)
	if err != nil {
		logger.Error().Str("component", "sla").Err(err).Msg("failed to list sla policies")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list sla policies"))
	}
	return c.JSON(dto.SuccessResp(dto.SLAsToResp(slas, bindings)))
}

func (h *SLAHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	var req dto.CreateSLAReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	bhOnly := false
	if req.BusinessHoursOnly != nil {
		bhOnly = *req.BusinessHoursOnly
	}
	m, bindings, err := h.svc.Create(c.Context(), accountID, service.UpsertSLAInput{
		Name:                 req.Name,
		FirstResponseMinutes: req.FirstResponseMinutes,
		ResolutionMinutes:    req.ResolutionMinutes,
		BusinessHoursOnly:    bhOnly,
		InboxIDs:             req.InboxIDs,
		LabelIDs:             req.LabelIDs,
	})
	if err != nil {
		logger.Error().Str("component", "sla").Err(err).Msg("failed to create sla")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to create sla"))
	}
	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.SLAToResp(m, bindings)))
}

func (h *SLAHandler) Get(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid sla id"))
	}
	m, bindings, err := h.svc.Get(c.Context(), accountID, id)
	if err != nil {
		if repo.IsErrNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "sla_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to get sla"))
	}
	return c.JSON(dto.SuccessResp(dto.SLAToResp(m, bindings)))
}

func (h *SLAHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid sla id"))
	}
	var req dto.UpdateSLAReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}
	in := service.UpdateSLAInput{
		Name:                 req.Name,
		FirstResponseMinutes: req.FirstResponseMinutes,
		ResolutionMinutes:    req.ResolutionMinutes,
		BusinessHoursOnly:    req.BusinessHoursOnly,
	}
	if req.InboxIDs != nil {
		in.InboxIDs = &req.InboxIDs
	}
	if req.LabelIDs != nil {
		in.LabelIDs = &req.LabelIDs
	}
	m, bindings, err := h.svc.Update(c.Context(), accountID, id, in)
	if err != nil {
		if repo.IsErrNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "sla_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to update sla"))
	}
	return c.JSON(dto.SuccessResp(dto.SLAToResp(m, bindings)))
}

func (h *SLAHandler) Report(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	from := c.Query("from")
	to := c.Query("to")
	rep, err := h.svc.Report(c.Context(), accountID, from, to)
	if err != nil {
		logger.Error().Str("component", "sla").Err(err).Msg("failed to build sla report")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to build sla report"))
	}
	return c.JSON(dto.SuccessResp(rep))
}

func (h *SLAHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid sla id"))
	}
	if err := h.svc.Delete(c.Context(), accountID, id); err != nil {
		if repo.IsErrNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "sla_not_found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to delete sla"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}
