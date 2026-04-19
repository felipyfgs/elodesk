package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoteNotFound = errors.New("note not found")

const noteSelectColumns = "id, account_id, contact_id, user_id, content, created_at, updated_at"

func scanNote(scanner interface{ Scan(dest ...any) error }, m *model.Note) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.ContactID, &m.UserID, &m.Content, &m.CreatedAt, &m.UpdatedAt)
}

type NoteRepo struct {
	pool *pgxpool.Pool
}

func NewNoteRepo(pool *pgxpool.Pool) *NoteRepo {
	return &NoteRepo{pool: pool}
}

func (r *NoteRepo) Create(ctx context.Context, m *model.Note) error {
	query := `INSERT INTO notes (account_id, contact_id, user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.ContactID, m.UserID, m.Content).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}
	return nil
}

func (r *NoteRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Note, error) {
	query := `SELECT ` + noteSelectColumns + ` FROM notes WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Note
	if err := scanNote(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrNoteNotFound, err)
		}
		return nil, fmt.Errorf("failed to find note: %w", err)
	}
	return &m, nil
}

func (r *NoteRepo) ListByContact(ctx context.Context, contactID, accountID int64, page, perPage int) ([]model.Note, int, error) {
	countQuery := `SELECT COUNT(*) FROM notes WHERE contact_id = $1 AND account_id = $2`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, contactID, accountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notes: %w", err)
	}
	if total == 0 {
		return []model.Note{}, 0, nil
	}

	offset := (page - 1) * perPage
	query := `SELECT ` + noteSelectColumns + ` FROM notes WHERE contact_id = $1 AND account_id = $2
		ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, query, contactID, accountID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var m model.Note
		if err := scanNote(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, m)
	}
	return notes, total, rows.Err()
}

func (r *NoteRepo) Update(ctx context.Context, m *model.Note) error {
	query := `UPDATE notes SET content = $3, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + noteSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.Content)
	if err := scanNote(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrNoteNotFound, err)
		}
		return fmt.Errorf("failed to update note: %w", err)
	}
	return nil
}

func (r *NoteRepo) Delete(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrNoteNotFound, pgx.ErrNoRows)
	}
	return nil
}
