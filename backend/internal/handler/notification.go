package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}

	status := c.Query("status", "unread")
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	cursor, _ := strconv.ParseInt(c.Query("cursor", "0"), 10, 64)

	items, err := h.svc.List(c.Context(), repo.NotificationListFilter{
		AccountID:  accountID,
		UserID:     authUser.ID,
		UnreadOnly: status == "unread",
		Limit:      limit,
		Cursor:     cursor,
	})
	if err != nil {
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to list notifications")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list notifications"))
	}

	unread, err := h.svc.UnreadCount(c.Context(), accountID, authUser.ID)
	if err != nil {
		logger.Warn().Str("component", "notifications").Err(err).Msg("failed to count unread")
	}

	payload := make([]fiber.Map, 0, len(items))
	var nextCursor int64
	for _, n := range items {
		var parsed any
		if err := json.Unmarshal([]byte(n.Payload), &parsed); err != nil {
			parsed = map[string]any{}
		}
		payload = append(payload, fiber.Map{
			"id":        n.ID,
			"accountId": n.AccountID,
			"userId":    n.UserID,
			"type":      n.Type,
			"payload":   parsed,
			"readAt":    n.ReadAt,
			"createdAt": n.CreatedAt,
		})
		nextCursor = n.ID
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"items":       payload,
		"unreadCount": unread,
		"nextCursor":  nextCursor,
	}))
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid id"))
	}

	if err := h.svc.MarkRead(c.Context(), id, accountID, authUser.ID); err != nil {
		if repo.IsErrNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "notification not found"))
		}
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to mark notification read")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to mark read"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}
	if err := h.svc.MarkAllRead(c.Context(), accountID, authUser.ID); err != nil {
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to mark all read")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to mark all read"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) GetPreferences(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}
	if authUser.ID != userID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "cannot read other user preferences"))
	}
	prefs, err := h.svc.GetUserPreferences(c.Context(), userID)
	if err != nil {
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to get prefs")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed"))
	}
	var parsed any
	if err := json.Unmarshal([]byte(prefs), &parsed); err != nil {
		parsed = map[string]any{}
	}
	return c.JSON(dto.SuccessResp(parsed))
}

func (h *NotificationHandler) SetPreferences(c *fiber.Ctx) error {
	authUser, ok := c.Locals("user").(*repo.AuthUser)
	if !ok || authUser == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "not authenticated"))
	}
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid user id"))
	}
	if authUser.ID != userID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "cannot update other user preferences"))
	}
	var body map[string]any
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.svc.SetUserPreferences(c.Context(), userID, string(data)); err != nil {
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to set prefs")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed"))
	}
	return c.JSON(dto.SuccessResp(body))
}
