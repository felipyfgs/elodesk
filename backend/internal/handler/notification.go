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
	svc              *service.NotificationService
	conversationRepo *repo.ConversationRepo
}

func NewNotificationHandler(svc *service.NotificationService, conversationRepo *repo.ConversationRepo) *NotificationHandler {
	return &NotificationHandler{svc: svc, conversationRepo: conversationRepo}
}

type notificationConvSummary struct {
	ID          int64                       `json:"id"`
	DisplayID   int64                       `json:"display_id"`
	Status      int                         `json:"status"`
	Inbox       *notificationInboxSummary   `json:"inbox,omitempty"`
	Contact     *notificationContactSummary `json:"contact,omitempty"`
	Assignee    *notificationUserSummary    `json:"assignee,omitempty"`
	LastMessage *notificationMessageSummary `json:"last_message,omitempty"`
}

type notificationInboxSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
}

type notificationContactSummary struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type notificationUserSummary struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type notificationMessageSummary struct {
	Content     *string `json:"content,omitempty"`
	ContentType int     `json:"content_type"`
	MessageType int     `json:"message_type"`
	Private     bool    `json:"private"`
	CreatedAt   int64   `json:"created_at"`
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
	sortOrder := c.Query("sort_order", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	items, err := h.svc.List(c.Context(), repo.NotificationListFilter{
		AccountID:  accountID,
		UserID:     authUser.ID,
		UnreadOnly: status == "unread",
		Limit:      limit,
		Cursor:     cursor,
		SortOrder:  sortOrder,
	})
	if err != nil {
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to list notifications")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to list notifications"))
	}

	unread, err := h.svc.UnreadCount(c.Context(), accountID, authUser.ID)
	if err != nil {
		logger.Warn().Str("component", "notifications").Err(err).Msg("failed to count unread")
	}

	parsedPayloads := make([]map[string]any, len(items))
	convIDSet := make(map[int64]struct{}, len(items))
	for i, n := range items {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(n.Payload), &parsed); err != nil || parsed == nil {
			parsed = map[string]any{}
		}
		parsedPayloads[i] = parsed
		if cid, ok := payloadConversationID(parsed); ok {
			convIDSet[cid] = struct{}{}
		}
	}

	var convs map[int64]*repo.ConversationHydrated
	if h.conversationRepo != nil && len(convIDSet) > 0 {
		ids := make([]int64, 0, len(convIDSet))
		for id := range convIDSet {
			ids = append(ids, id)
		}
		convs, err = h.conversationRepo.ListHydratedByIDs(c.Context(), accountID, ids)
		if err != nil {
			logger.Warn().Str("component", "notifications").Err(err).Msg("failed to hydrate conversations for notifications")
			convs = nil
		}
	}

	payload := make([]fiber.Map, 0, len(items))
	var nextCursor int64
	for i, n := range items {
		entry := fiber.Map{
			"id":        n.ID,
			"accountId": n.AccountID,
			"userId":    n.UserID,
			"type":      n.Type,
			"payload":   parsedPayloads[i],
			"readAt":    n.ReadAt,
			"createdAt": n.CreatedAt,
		}
		if cid, ok := payloadConversationID(parsedPayloads[i]); ok {
			if conv, found := convs[cid]; found {
				entry["conversation"] = buildNotificationConvSummary(conv)
			}
		}
		payload = append(payload, entry)
		nextCursor = n.ID
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"items":       payload,
		"unreadCount": unread,
		"nextCursor":  nextCursor,
	}))
}

func payloadConversationID(p map[string]any) (int64, bool) {
	var n int64
	switch v := p["conversation_id"].(type) {
	case float64:
		n = int64(v)
	case int64:
		n = v
	case int:
		n = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		n = parsed
	default:
		return 0, false
	}
	if n <= 0 {
		return 0, false
	}
	return n, true
}

func buildNotificationConvSummary(h *repo.ConversationHydrated) notificationConvSummary {
	c := h.Conversation
	out := notificationConvSummary{
		ID:        c.ID,
		DisplayID: c.DisplayID,
		Status:    int(c.Status),
		Inbox: &notificationInboxSummary{
			ID:          h.Inbox.ID,
			Name:        h.Inbox.Name,
			ChannelType: h.Inbox.ChannelType,
		},
		Contact: &notificationContactSummary{
			ID:        h.Contact.ID,
			Name:      h.Contact.Name,
			AvatarURL: h.Contact.AvatarURL,
		},
	}
	if h.Assignee != nil {
		out.Assignee = &notificationUserSummary{
			ID:        h.Assignee.ID,
			Name:      h.Assignee.Name,
			AvatarURL: h.Assignee.AvatarURL,
		}
	}
	if h.LastNonActivityMessage != nil {
		m := h.LastNonActivityMessage
		out.LastMessage = &notificationMessageSummary{
			Content:     m.Content,
			ContentType: int(m.ContentType),
			MessageType: int(m.MessageType),
			Private:     m.Private,
			CreatedAt:   m.CreatedAt.Unix(),
		}
	}
	return out
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

// MarkUnread reverts a notification's read state. Mirrors Chatwoot's
// POST /api/v1/notifications/:id/unread.
func (h *NotificationHandler) MarkUnread(c *fiber.Ctx) error {
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

	if err := h.svc.MarkUnread(c.Context(), id, accountID, authUser.ID); err != nil {
		if repo.IsErrNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "notification not found"))
		}
		logger.Error().Str("component", "notifications").Err(err).Msg("failed to mark notification unread")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "failed to mark unread"))
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
	prefs, err := h.svc.FindUserPreferences(c.Context(), userID)
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
