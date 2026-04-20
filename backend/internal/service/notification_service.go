package service

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
)

// NotificationService persists per-user notifications and broadcasts them on
// the user's realtime room so UI slideovers/badges update instantly.
type NotificationService struct {
	repo *repo.NotificationRepo
	hub  *realtime.Hub
}

func NewNotificationService(repo *repo.NotificationRepo, hub *realtime.Hub) *NotificationService {
	return &NotificationService{repo: repo, hub: hub}
}

// Create persists a notification and publishes a realtime `notification.new`
// event on the user's private room. Payload is marshalled to JSON — callers
// pass typed structs or map[string]any.
func (s *NotificationService) Create(ctx context.Context, accountID, userID int64, ntype string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notification.create marshal: %w", err)
	}
	n := &model.Notification{
		AccountID: accountID,
		UserID:    userID,
		Type:      ntype,
		Payload:   string(data),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return err
	}
	if s.hub != nil {
		event := map[string]any{
			"type": "notification.new",
			"payload": map[string]any{
				"id":        n.ID,
				"accountId": n.AccountID,
				"userId":    n.UserID,
				"type":      n.Type,
				"payload":   json.RawMessage(data),
				"createdAt": n.CreatedAt,
			},
		}
		encoded, err := json.Marshal(event)
		if err == nil {
			s.hub.Broadcast(realtime.UserRoom(accountID, userID), encoded)
		} else {
			logger.Warn().Str("component", "notifications").Err(err).Msg("failed to marshal notification event")
		}
	}
	return nil
}

func (s *NotificationService) List(ctx context.Context, f repo.NotificationListFilter) ([]model.Notification, error) {
	return s.repo.List(ctx, f)
}

func (s *NotificationService) UnreadCount(ctx context.Context, accountID, userID int64) (int, error) {
	return s.repo.UnreadCount(ctx, accountID, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, accountID, userID int64) error {
	if err := s.repo.MarkRead(ctx, id, accountID, userID); err != nil {
		return err
	}
	if s.hub != nil {
		event := map[string]any{
			"type":    "notification.read",
			"payload": map[string]any{"id": id},
		}
		if data, err := json.Marshal(event); err == nil {
			s.hub.Broadcast(realtime.UserRoom(accountID, userID), data)
		}
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, accountID, userID int64) error {
	if err := s.repo.MarkAllRead(ctx, accountID, userID); err != nil {
		return err
	}
	if s.hub != nil {
		event := map[string]any{
			"type":    "notification.read_all",
			"payload": map[string]any{},
		}
		if data, err := json.Marshal(event); err == nil {
			s.hub.Broadcast(realtime.UserRoom(accountID, userID), data)
		}
	}
	return nil
}

func (s *NotificationService) GetUserPreferences(ctx context.Context, userID int64) (string, error) {
	return s.repo.GetUserPreferences(ctx, userID)
}

func (s *NotificationService) SetUserPreferences(ctx context.Context, userID int64, prefs string) error {
	return s.repo.SetUserPreferences(ctx, userID, prefs)
}
