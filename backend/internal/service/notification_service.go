package service

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

// NotificationService persists per-user notifications and broadcasts them on
// the user's realtime room so UI slideovers/badges update instantly.
//
// Eventos emitidos (todos via RealtimeService.BroadcastUserEvent — mesma forma
// `{type, payload}` dos demais broadcasts):
//
//   - notification.new      → payload completo da notificação persistida
//   - notification.read     → {id}
//   - notification.read_all → {}
type NotificationService struct {
	repo *repo.NotificationRepo
	rt   *RealtimeService
}

func NewNotificationService(repo *repo.NotificationRepo, rt *RealtimeService) *NotificationService {
	return &NotificationService{repo: repo, rt: rt}
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
	if s.rt != nil {
		s.rt.BroadcastUserEvent(accountID, userID, "notification.new", map[string]any{
			"id":        n.ID,
			"accountId": n.AccountID,
			"userId":    n.UserID,
			"type":      n.Type,
			"payload":   json.RawMessage(data),
			"createdAt": n.CreatedAt,
		})
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
	if s.rt != nil {
		s.rt.BroadcastUserEvent(accountID, userID, "notification.read", map[string]any{"id": id})
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, accountID, userID int64) error {
	if err := s.repo.MarkAllRead(ctx, accountID, userID); err != nil {
		return err
	}
	if s.rt != nil {
		s.rt.BroadcastUserEvent(accountID, userID, "notification.read_all", map[string]any{})
	}
	return nil
}

func (s *NotificationService) GetUserPreferences(ctx context.Context, userID int64) (string, error) {
	return s.repo.GetUserPreferences(ctx, userID)
}

func (s *NotificationService) SetUserPreferences(ctx context.Context, userID int64, prefs string) error {
	return s.repo.SetUserPreferences(ctx, userID, prefs)
}
