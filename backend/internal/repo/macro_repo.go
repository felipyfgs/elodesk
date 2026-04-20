package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMacroNotFound = errors.New("macro not found")

const macroSelectColumns = "id, account_id, name, visibility, conditions, actions, created_by, created_at, updated_at"

type macroScanner interface {
	Scan(dest ...any) error
}

func scanMacro(scanner macroScanner, m *model.Macro) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Name, &m.Visibility, &m.Conditions, &m.Actions, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
}

type MacroRepo struct {
	pool *pgxpool.Pool
}

func NewMacroRepo(pool *pgxpool.Pool) *MacroRepo {
	return &MacroRepo{pool: pool}
}

func (r *MacroRepo) Create(ctx context.Context, m *model.Macro) error {
	query := `INSERT INTO macros (account_id, name, visibility, conditions, actions, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Visibility, m.Conditions, m.Actions, m.CreatedBy).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create macro: %w", err)
	}
	return nil
}

func (r *MacroRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Macro, error) {
	query := `SELECT ` + macroSelectColumns + ` FROM macros WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Macro
	if err := scanMacro(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrMacroNotFound, err)
		}
		return nil, fmt.Errorf("failed to find macro: %w", err)
	}
	return &m, nil
}

func (r *MacroRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.Macro, error) {
	query := `SELECT ` + macroSelectColumns + ` FROM macros WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list macros: %w", err)
	}
	defer rows.Close()

	var result []model.Macro
	for rows.Next() {
		var m model.Macro
		if err := scanMacro(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan macro: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *MacroRepo) Update(ctx context.Context, m *model.Macro) error {
	query := `UPDATE macros SET name = $1, visibility = $2, conditions = $3, actions = $4, updated_at = NOW() WHERE id = $5 AND account_id = $6`
	_, err := r.pool.Exec(ctx, query, m.Name, m.Visibility, m.Conditions, m.Actions, m.ID, m.AccountID)
	if err != nil {
		return fmt.Errorf("failed to update macro: %w", err)
	}
	return nil
}

func (r *MacroRepo) Delete(ctx context.Context, id, accountID int64) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM macros WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete macro: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrMacroNotFound, pgx.ErrNoRows)
	}
	return nil
}
