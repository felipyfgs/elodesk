package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMessageNotFound = errors.New("message not found")

const messageSelectColumns = "id, account_id, inbox_id, conversation_id, message_type, content_type, content, source_id, private, status, content_attributes, sender_type, sender_id, external_source_ids, created_at, updated_at, deleted_at"

type messageScanner interface {
	Scan(dest ...any) error
}

func scanMessage(scanner messageScanner, m *model.Message) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.ConversationID, &m.MessageType, &m.ContentType, &m.Content, &m.SourceID, &m.Private, &m.Status, &m.ContentAttrs, &m.SenderType, &m.SenderID, &m.ExternalSourceIDs, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
}

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

// Create inserts a message. When source_id is non-nil the operation is
// idempotent via the partial unique index idx_messages_inbox_source: a
// subsequent call from the same provider with the same (inbox_id, source_id)
// returns the existing row with content refreshed (providers may re-deliver
// edited content). When source_id is nil each call inserts a new row.
func (r *MessageRepo) Create(ctx context.Context, m *model.Message) (*model.Message, error) {
	if m.SourceID != nil {
		query := `INSERT INTO messages (account_id, inbox_id, conversation_id, message_type, content_type, content, source_id, private, status, content_attributes, sender_type, sender_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (inbox_id, source_id) WHERE source_id IS NOT NULL DO UPDATE SET
				content = COALESCE(EXCLUDED.content, messages.content),
				updated_at = NOW()
			RETURNING ` + messageSelectColumns
		row := r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.ConversationID, m.MessageType, m.ContentType, m.Content, m.SourceID, m.Private, m.Status, m.ContentAttrs, m.SenderType, m.SenderID)
		var result model.Message
		if err := scanMessage(row, &result); err != nil {
			return nil, fmt.Errorf("failed to upsert message: %w", err)
		}
		return &result, nil
	}

	query := `INSERT INTO messages (account_id, inbox_id, conversation_id, message_type, content_type, content, source_id, private, status, content_attributes, sender_type, sender_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING ` + messageSelectColumns
	row := r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.ConversationID, m.MessageType, m.ContentType, m.Content, m.SourceID, m.Private, m.Status, m.ContentAttrs, m.SenderType, m.SenderID)
	var result model.Message
	if err := scanMessage(row, &result); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	return &result, nil
}

func (r *MessageRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Message, error) {
	query := `SELECT ` + messageSelectColumns + ` FROM messages WHERE id = $1 AND account_id = $2 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Message
	if err := scanMessage(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrMessageNotFound, err)
		}
		return nil, fmt.Errorf("failed to find message: %w", err)
	}
	return &m, nil
}

func (r *MessageRepo) SoftDelete(ctx context.Context, id, accountID int64) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND account_id = $3 AND deleted_at IS NULL`,
		now, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
	}
	return nil
}

func (r *MessageRepo) FindBySourceID(ctx context.Context, sourceID string, accountID int64) (*model.Message, error) {
	query := `SELECT ` + messageSelectColumns + ` FROM messages WHERE source_id = $1 AND account_id = $2 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query, sourceID, accountID)
	var m model.Message
	if err := scanMessage(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrMessageNotFound, err)
		}
		return nil, fmt.Errorf("failed to find message by source_id: %w", err)
	}
	return &m, nil
}

type MessageListFilter struct {
	ConversationID int64
	AccountID      int64
	Before         *time.Time
	Page           int
	PerPage        int
}

func (r *MessageRepo) ListByConversation(ctx context.Context, f MessageListFilter) ([]model.Message, int, error) {
	countQuery := `SELECT COUNT(*) FROM messages WHERE conversation_id = $1 AND account_id = $2 AND deleted_at IS NULL`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, f.ConversationID, f.AccountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}
	if total == 0 {
		return []model.Message{}, 0, nil
	}

	dataQuery := `SELECT ` + messageSelectColumns + ` FROM messages WHERE conversation_id = $1 AND account_id = $2 AND deleted_at IS NULL`
	var args []any
	args = append(args, f.ConversationID, f.AccountID)
	argN := 3

	if f.Before != nil {
		dataQuery += fmt.Sprintf(" AND created_at < $%d", argN)
		args = append(args, *f.Before)
		argN++
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", f.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var m model.Message
		if err := scanMessage(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, total, rows.Err()
}
