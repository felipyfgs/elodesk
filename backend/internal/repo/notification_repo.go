package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

const notificationSelectColumns = "id, account_id, user_id, type, payload, read_at, created_at"

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	query := `INSERT INTO notifications (account_id, user_id, type, payload)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id, created_at`
	payload := n.Payload
	if payload == "" {
		payload = "{}"
	}
	err := r.pool.QueryRow(ctx, query, n.AccountID, n.UserID, n.Type, payload).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

type NotificationListFilter struct {
	AccountID  int64
	UserID     int64
	UnreadOnly bool
	Limit      int
	Cursor     int64 // created_at-based cursor encoded as id for pagination
}

func (r *NotificationRepo) List(ctx context.Context, f NotificationListFilter) ([]model.Notification, error) {
	if f.Limit <= 0 {
		f.Limit = 25
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	where := "WHERE account_id = $1 AND user_id = $2"
	args := []any{f.AccountID, f.UserID}
	idx := 3
	if f.UnreadOnly {
		where += " AND read_at IS NULL"
	}
	if f.Cursor > 0 {
		where += fmt.Sprintf(" AND id < $%d", idx)
		args = append(args, f.Cursor)
	}
	query := fmt.Sprintf(`SELECT %s FROM notifications %s ORDER BY id DESC LIMIT %d`, notificationSelectColumns, where, f.Limit)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var result []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.AccountID, &n.UserID, &n.Type, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, accountID, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE account_id = $1 AND user_id = $2 AND read_at IS NULL`,
		accountID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, accountID, userID int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND account_id = $2 AND user_id = $3 AND read_at IS NULL`,
		id, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND account_id = $2 AND user_id = $3)`, id, accountID, userID).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check notification: %w", err)
		}
		if !exists {
			return fmt.Errorf("%w: %w", ErrNotificationNotFound, pgx.ErrNoRows)
		}
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, accountID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE account_id = $1 AND user_id = $2 AND read_at IS NULL`,
		accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications read: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetUserPreferences(ctx context.Context, userID int64) (string, error) {
	var prefs string
	err := r.pool.QueryRow(ctx, `SELECT notification_preferences FROM users WHERE id = $1`, userID).Scan(&prefs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "{}", nil
		}
		return "", fmt.Errorf("failed to get notification preferences: %w", err)
	}
	if prefs == "" {
		return "{}", nil
	}
	return prefs, nil
}

func (r *NotificationRepo) SetUserPreferences(ctx context.Context, userID int64, prefs string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET notification_preferences = $1::jsonb, updated_at = NOW() WHERE id = $2`, prefs, userID)
	if err != nil {
		return fmt.Errorf("failed to set notification preferences: %w", err)
	}
	return nil
}
