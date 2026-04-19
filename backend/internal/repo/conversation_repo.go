package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

const conversationSelectColumns = "id, account_id, inbox_id, status, assignee_id, contact_id, contact_inbox_id, display_id, uuid, last_activity_at, additional_attributes, created_at, updated_at"

type conversationScanner interface {
	Scan(dest ...any) error
}

func scanConversation(scanner conversationScanner, m *model.Conversation) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.Status, &m.AssigneeID, &m.ContactID, &m.ContactInboxID, &m.DisplayID, &m.UUID, &m.LastActivityAt, &m.AdditionalAttrs, &m.CreatedAt, &m.UpdatedAt)
}

type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

func (r *ConversationRepo) Create(ctx context.Context, m *model.Conversation) error {
	query := `INSERT INTO conversations (account_id, inbox_id, status, assignee_id, contact_id, contact_inbox_id, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, display_id, uuid, last_activity_at, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.Status, m.AssigneeID, m.ContactID, m.ContactInboxID, m.AdditionalAttrs).
		Scan(&m.ID, &m.DisplayID, &m.UUID, &m.LastActivityAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Conversation, error) {
	query := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}
	return &m, nil
}

func (r *ConversationRepo) ToggleStatus(ctx context.Context, id, accountID int64, newStatus model.ConversationStatus) (*model.Conversation, error) {
	query := `UPDATE conversations SET status = $1, updated_at = NOW() WHERE id = $2 AND account_id = $3
		RETURNING ` + conversationSelectColumns
	row := r.pool.QueryRow(ctx, query, newStatus, id, accountID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to toggle conversation status: %w", err)
	}
	return &m, nil
}

type ConversationFilter struct {
	AccountID  int64
	InboxID    *int64
	Status     *model.ConversationStatus
	AssigneeID *int64
	ContactID  *int64
	Page       int
	PerPage    int
}

func (r *ConversationRepo) ListByAccount(ctx context.Context, f ConversationFilter) ([]model.Conversation, int, error) {
	countQuery := `SELECT COUNT(*) FROM conversations WHERE account_id = $1`
	var args []any
	args = append(args, f.AccountID)
	argN := 2

	if f.InboxID != nil {
		countQuery += fmt.Sprintf(" AND inbox_id = $%d", argN)
		args = append(args, *f.InboxID)
		argN++
	}
	if f.Status != nil {
		countQuery += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *f.Status)
		argN++
	}
	if f.AssigneeID != nil {
		countQuery += fmt.Sprintf(" AND assignee_id = $%d", argN)
		args = append(args, *f.AssigneeID)
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	dataQuery := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE account_id = $1`
	if f.InboxID != nil {
		dataQuery += fmt.Sprintf(" AND inbox_id = $%d", argN)
		args = append(args, *f.InboxID)
		argN++
	}
	if f.Status != nil {
		dataQuery += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *f.Status)
		argN++
	}
	if f.AssigneeID != nil {
		dataQuery += fmt.Sprintf(" AND assignee_id = $%d", argN)
		args = append(args, *f.AssigneeID)
		argN++
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery += fmt.Sprintf(" ORDER BY last_activity_at DESC LIMIT %d OFFSET %d", f.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var convos []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convos = append(convos, m)
	}
	return convos, total, rows.Err()
}

func (r *ConversationRepo) UpdateLastSeen(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET last_activity_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}
	return nil
}
