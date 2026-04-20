package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
)

type AuditLogHandler struct {
	repo *repo.AuditLogRepo
}

func NewAuditLogHandler(r *repo.AuditLogRepo) *AuditLogHandler {
	return &AuditLogHandler{repo: r}
}

func (h *AuditLogHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	from := c.Query("from")
	to := c.Query("to")
	action := c.Query("action")
	entityType := c.Query("entity_type")
	var userIDPtr *int64
	if s := c.Query("user_id"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			userIDPtr = &v
		}
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "50"))

	entries, total, err := h.repo.List(c.Context(), accountID, from, to, action, entityType, userIDPtr, page, pageSize)
	if err != nil {
		logger.Error().Str("component", "audit_logs").Err(err).Msg("failed to list audit logs")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list audit logs"))
	}

	resp := make([]dto.AuditLogResp, 0, len(entries))
	for i := range entries {
		e := entries[i]
		var entityType string
		if e.EntityType != nil {
			entityType = *e.EntityType
		}
		var ip string
		if e.IPAddress != nil {
			ip = e.IPAddress.String()
		}
		var ua string
		if e.UserAgent != nil {
			ua = *e.UserAgent
		}
		var metadata any
		if e.Metadata != nil && *e.Metadata != "" {
			_ = json.Unmarshal([]byte(*e.Metadata), &metadata)
		}
		resp = append(resp, dto.AuditLogResp{
			ID:         e.ID,
			AccountID:  e.AccountID,
			UserID:     e.UserID,
			Action:     e.Action,
			EntityType: entityType,
			EntityID:   e.EntityID,
			Metadata:   metadata,
			IPAddress:  ip,
			UserAgent:  ua,
			CreatedAt:  e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"payload": resp,
		"meta": fiber.Map{
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	}))
}
