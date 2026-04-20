package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/model"
)

var ErrCustomFilterNotFound = errors.New("custom filter not found")

const customFilterSelectColumns = "id, account_id, user_id, name, filter_type, query, created_at, updated_at"

func scanCustomFilter(scanner interface{ Scan(dest ...any) error }, m *model.CustomFilter) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.UserID, &m.Name, &m.FilterType, &m.Query, &m.CreatedAt, &m.UpdatedAt)
}

type CustomFilterRepo struct {
	pool *pgxpool.Pool
}

func NewCustomFilterRepo(pool *pgxpool.Pool) *CustomFilterRepo {
	return &CustomFilterRepo{pool: pool}
}

func (r *CustomFilterRepo) Create(ctx context.Context, m *model.CustomFilter) error {
	query := `INSERT INTO custom_filters (account_id, user_id, name, filter_type, query)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.UserID, m.Name, m.FilterType, m.Query).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create custom filter: %w", err)
	}
	return nil
}

func (r *CustomFilterRepo) FindByID(ctx context.Context, id, accountID, userID int64) (*model.CustomFilter, error) {
	query := `SELECT ` + customFilterSelectColumns + ` FROM custom_filters WHERE id = $1 AND account_id = $2 AND user_id = $3`
	row := r.pool.QueryRow(ctx, query, id, accountID, userID)
	var m model.CustomFilter
	if err := scanCustomFilter(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrCustomFilterNotFound, err)
		}
		return nil, fmt.Errorf("failed to find custom filter: %w", err)
	}
	return &m, nil
}

func (r *CustomFilterRepo) ListByUser(ctx context.Context, accountID, userID int64, filterType string) ([]model.CustomFilter, error) {
	query := `SELECT ` + customFilterSelectColumns + ` FROM custom_filters WHERE account_id = $1 AND user_id = $2`
	var args []any
	args = append(args, accountID, userID)
	if filterType != "" {
		query += ` AND filter_type = $3`
		args = append(args, filterType)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom filters: %w", err)
	}
	defer rows.Close()

	var filters []model.CustomFilter
	for rows.Next() {
		var m model.CustomFilter
		if err := scanCustomFilter(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan custom filter: %w", err)
		}
		filters = append(filters, m)
	}
	return filters, rows.Err()
}

func (r *CustomFilterRepo) Update(ctx context.Context, m *model.CustomFilter) error {
	query := `UPDATE custom_filters SET name = $3, filter_type = $4, query = $5, updated_at = NOW()
		WHERE id = $1 AND account_id = $2 AND user_id = $6
		RETURNING ` + customFilterSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.Name, m.FilterType, m.Query, m.UserID)
	if err := scanCustomFilter(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrCustomFilterNotFound, err)
		}
		return fmt.Errorf("failed to update custom filter: %w", err)
	}
	return nil
}

func (r *CustomFilterRepo) Delete(ctx context.Context, id, accountID, userID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM custom_filters WHERE id = $1 AND account_id = $2 AND user_id = $3`,
		id, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete custom filter: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrCustomFilterNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *CustomFilterRepo) CountByUser(ctx context.Context, accountID, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM custom_filters WHERE account_id = $1 AND user_id = $2`,
		accountID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count custom filters: %w", err)
	}
	return count, nil
}
