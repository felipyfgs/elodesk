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

var ErrPipelineNotFound = errors.New("pipeline not found")

const pipelineSelectColumns = "id, account_id, name, description, template_key, card_kind, icon, color, archived_at, created_by, created_at, updated_at"

type pipelineScanner interface {
	Scan(dest ...any) error
}

func scanPipeline(scanner pipelineScanner, m *model.Pipeline) error {
	return scanner.Scan(
		&m.ID, &m.AccountID, &m.Name, &m.Description, &m.TemplateKey,
		&m.CardKind, &m.Icon, &m.Color, &m.ArchivedAt, &m.CreatedBy,
		&m.CreatedAt, &m.UpdatedAt,
	)
}

type PipelineRepo struct {
	pool *pgxpool.Pool
}

func NewPipelineRepo(pool *pgxpool.Pool) *PipelineRepo {
	return &PipelineRepo{pool: pool}
}

func (r *PipelineRepo) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *PipelineRepo) Insert(ctx context.Context, tx pgx.Tx, m *model.Pipeline) error {
	query := `INSERT INTO pipelines (account_id, name, description, template_key, card_kind, icon, color, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	var q pipelineScanner
	if tx != nil {
		q = tx.QueryRow(ctx, query, m.AccountID, m.Name, m.Description, m.TemplateKey, m.CardKind, m.Icon, m.Color, m.CreatedBy)
	} else {
		q = r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Description, m.TemplateKey, m.CardKind, m.Icon, m.Color, m.CreatedBy)
	}
	if err := q.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return fmt.Errorf("failed to insert pipeline: %w", err)
	}
	return nil
}

func (r *PipelineRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Pipeline, error) {
	query := `SELECT ` + pipelineSelectColumns + ` FROM pipelines WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Pipeline
	if err := scanPipeline(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrPipelineNotFound, err)
		}
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}
	return &m, nil
}

func (r *PipelineRepo) ListByAccount(ctx context.Context, accountID int64, includeArchived bool) ([]model.Pipeline, error) {
	var query string
	if includeArchived {
		query = `SELECT ` + pipelineSelectColumns + ` FROM pipelines WHERE account_id = $1 ORDER BY archived_at NULLS FIRST, name ASC`
	} else {
		query = `SELECT ` + pipelineSelectColumns + ` FROM pipelines WHERE account_id = $1 AND archived_at IS NULL ORDER BY name ASC`
	}
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()
	var out []model.Pipeline
	for rows.Next() {
		var m model.Pipeline
		if err := scanPipeline(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PipelineRepo) Update(ctx context.Context, m *model.Pipeline) error {
	query := `UPDATE pipelines
		SET name = $3, description = $4, icon = $5, color = $6, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + pipelineSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.Name, m.Description, m.Icon, m.Color)
	if err := scanPipeline(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrPipelineNotFound, err)
		}
		return fmt.Errorf("failed to update pipeline: %w", err)
	}
	return nil
}

func (r *PipelineRepo) Archive(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE pipelines SET archived_at = NOW(), updated_at = NOW() WHERE id = $1 AND account_id = $2 AND archived_at IS NULL`,
		id, accountID)
	if err != nil {
		return fmt.Errorf("failed to archive pipeline: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrPipelineNotFound, pgx.ErrNoRows)
	}
	return nil
}

// CountActiveByAccount returns the count of non-archived pipelines for an account.
func (r *PipelineRepo) CountActiveByAccount(ctx context.Context, accountID int64) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pipelines WHERE account_id = $1 AND archived_at IS NULL`, accountID).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count pipelines: %w", err)
	}
	return n, nil
}

// LastActivityByAccount returns the max(updated_at) of cards per pipeline for the account.
// Returns map[pipelineID]time.Time. Pipelines with no cards omitted.
func (r *PipelineRepo) LastActivityByAccount(ctx context.Context, accountID int64) (map[int64]time.Time, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.pipeline_id, MAX(c.updated_at) FROM pipeline_cards c
		 JOIN pipelines p ON p.id = c.pipeline_id
		 WHERE p.account_id = $1
		 GROUP BY c.pipeline_id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load last activity: %w", err)
	}
	defer rows.Close()
	out := map[int64]time.Time{}
	for rows.Next() {
		var pid int64
		var ts time.Time
		if err := rows.Scan(&pid, &ts); err != nil {
			return nil, err
		}
		out[pid] = ts
	}
	return out, rows.Err()
}
