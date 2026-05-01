package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

const attachmentSelectColumns = "id, message_id, account_id, file_type, external_url, file_key, file_name, extension, meta, created_at, updated_at"

type attachmentScanner interface {
	Scan(dest ...any) error
}

func scanAttachment(scanner attachmentScanner, m *model.Attachment) error {
	return scanner.Scan(&m.ID, &m.MessageID, &m.AccountID, &m.FileType, &m.ExternalURL, &m.FileKey, &m.FileName, &m.Extension, &m.Meta, &m.CreatedAt, &m.UpdatedAt)
}

type AttachmentRepo struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepo(pool *pgxpool.Pool) *AttachmentRepo {
	return &AttachmentRepo{pool: pool}
}

func (r *AttachmentRepo) Create(ctx context.Context, m *model.Attachment) error {
	query := `INSERT INTO attachments (message_id, account_id, file_type, external_url, file_key, file_name, extension, meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.MessageID, m.AccountID, m.FileType, m.ExternalURL, m.FileKey, m.FileName, m.Extension, m.Meta).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	return nil
}

func (r *AttachmentRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Attachment, error) {
	query := `SELECT ` + attachmentSelectColumns + ` FROM attachments WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Attachment
	if err := scanAttachment(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrAttachmentNotFound, err)
		}
		return nil, fmt.Errorf("failed to find attachment: %w", err)
	}
	return &m, nil
}

func (r *AttachmentRepo) FindByMessageID(ctx context.Context, messageID int64) ([]model.Attachment, error) {
	query := `SELECT ` + attachmentSelectColumns + ` FROM attachments WHERE message_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to find attachments by message: %w", err)
	}
	defer rows.Close()

	var attachments []model.Attachment
	for rows.Next() {
		var m model.Attachment
		if err := scanAttachment(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, m)
	}
	return attachments, rows.Err()
}

func (r *AttachmentRepo) FindByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]model.Attachment, error) {
	if len(messageIDs) == 0 {
		return map[int64][]model.Attachment{}, nil
	}
	query := `SELECT ` + attachmentSelectColumns + ` FROM attachments WHERE message_id = ANY($1) ORDER BY id`
	rows, err := r.pool.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to find attachments by message ids: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]model.Attachment)
	for rows.Next() {
		var m model.Attachment
		if err := scanAttachment(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		result[m.MessageID] = append(result[m.MessageID], m)
	}
	return result, rows.Err()
}

func (r *AttachmentRepo) UpdateFileKey(ctx context.Context, id int64, fileKey string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attachments SET file_key = $1, updated_at = NOW() WHERE id = $2`,
		fileKey, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update attachment file_key: %w", err)
	}
	return nil
}
