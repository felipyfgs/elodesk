package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCannedResponseNotFound = errors.New("canned response not found")
var ErrCannedShortCodeTaken = errors.New("canned short_code already taken")

const cannedResponseSelectColumns = "id, account_id, short_code, content, created_at, updated_at"

func scanCannedResponse(scanner interface{ Scan(dest ...any) error }, m *model.CannedResponse) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.ShortCode, &m.Content, &m.CreatedAt, &m.UpdatedAt)
}

type CannedResponseRepo struct {
	pool *pgxpool.Pool
}

func NewCannedResponseRepo(pool *pgxpool.Pool) *CannedResponseRepo {
	return &CannedResponseRepo{pool: pool}
}

func (r *CannedResponseRepo) Create(ctx context.Context, m *model.CannedResponse) error {
	query := `INSERT INTO canned_responses (account_id, short_code, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.ShortCode, m.Content).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrCannedShortCodeTaken, err)
		}
		return fmt.Errorf("failed to create canned response: %w", err)
	}
	return nil
}

func (r *CannedResponseRepo) FindByID(ctx context.Context, id, accountID int64) (*model.CannedResponse, error) {
	query := `SELECT ` + cannedResponseSelectColumns + ` FROM canned_responses WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.CannedResponse
	if err := scanCannedResponse(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrCannedResponseNotFound, err)
		}
		return nil, fmt.Errorf("failed to find canned response: %w", err)
	}
	return &m, nil
}

func (r *CannedResponseRepo) ListByAccount(ctx context.Context, accountID int64, search string, limit int) ([]model.CannedResponse, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}

	var args []any
	args = append(args, accountID)

	var conditions []string
	conditions = append(conditions, "account_id = $1")

	if search != "" {
		conditions = append(conditions, "(short_code ILIKE $2 OR content ILIKE $2)")
		args = append(args, "%"+search+"%")
	}

	query := `SELECT ` + cannedResponseSelectColumns + ` FROM canned_responses WHERE ` + strings.Join(conditions, " AND ")

	if search != "" {
		query += ` ORDER BY
			CASE WHEN short_code ILIKE $2 THEN 0
				WHEN short_code ILIKE '%' || $3 || '%' THEN 1
				ELSE 2 END,
			short_code ASC`
		args = append(args, search)
	} else {
		query += ` ORDER BY short_code ASC`
	}

	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list canned responses: %w", err)
	}
	defer rows.Close()

	var items []model.CannedResponse
	for rows.Next() {
		var m model.CannedResponse
		if err := scanCannedResponse(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan canned response: %w", err)
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *CannedResponseRepo) Update(ctx context.Context, m *model.CannedResponse) error {
	query := `UPDATE canned_responses SET short_code = $3, content = $4, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + cannedResponseSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.ShortCode, m.Content)
	if err := scanCannedResponse(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrCannedResponseNotFound, err)
		}
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrCannedShortCodeTaken, err)
		}
		return fmt.Errorf("failed to update canned response: %w", err)
	}
	return nil
}

func (r *CannedResponseRepo) Delete(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM canned_responses WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete canned response: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrCannedResponseNotFound, pgx.ErrNoRows)
	}
	return nil
}
