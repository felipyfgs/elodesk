package service

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
)

type NotificationService struct {
	repo *repo.NotificationRepo
	rt   *RealtimeService
}

func NewNotificationService(repo *repo.NotificationRepo, rt *RealtimeService) *NotificationService {
	return &NotificationService{repo: repo, rt: rt}
}

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

// MarkUnread reverts a previously read notification back to unread and emits
// `notification.unread` so the bell badge increments live across tabs.
func (s *NotificationService) MarkUnread(ctx context.Context, id, accountID, userID int64) error {
	if err := s.repo.MarkUnread(ctx, id, accountID, userID); err != nil {
		return err
	}
	if s.rt != nil {
		s.rt.BroadcastUserEvent(accountID, userID, "notification.unread", map[string]any{"id": id})
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

func (s *NotificationService) FindUserPreferences(ctx context.Context, userID int64) (string, error) {
	return s.repo.FindUserPreferences(ctx, userID)
}

func (s *NotificationService) SetUserPreferences(ctx context.Context, userID int64, prefs string) error {
	return s.repo.SetUserPreferences(ctx, userID, prefs)
}
