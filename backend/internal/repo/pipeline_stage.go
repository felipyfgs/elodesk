package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPipelineStageNotFound = errors.New("pipeline stage not found")

const pipelineStageSelectColumns = "id, pipeline_id, name, position, color, is_terminal, terminal_kind, created_at, updated_at"

type stageScanner interface {
	Scan(dest ...any) error
}

func scanStage(scanner stageScanner, m *model.PipelineStage) error {
	return scanner.Scan(
		&m.ID, &m.PipelineID, &m.Name, &m.Position, &m.Color,
		&m.IsTerminal, &m.TerminalKind, &m.CreatedAt, &m.UpdatedAt,
	)
}

type PipelineStageRepo struct {
	pool *pgxpool.Pool
}

func NewPipelineStageRepo(pool *pgxpool.Pool) *PipelineStageRepo {
	return &PipelineStageRepo{pool: pool}
}

func (r *PipelineStageRepo) Insert(ctx context.Context, tx pgx.Tx, m *model.PipelineStage) error {
	query := `INSERT INTO pipeline_stages (pipeline_id, name, position, color, is_terminal, terminal_kind)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, query, m.PipelineID, m.Name, m.Position, m.Color, m.IsTerminal, m.TerminalKind)
	} else {
		row = r.pool.QueryRow(ctx, query, m.PipelineID, m.Name, m.Position, m.Color, m.IsTerminal, m.TerminalKind)
	}
	if err := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return fmt.Errorf("failed to insert stage: %w", err)
	}
	return nil
}

func (r *PipelineStageRepo) FindByID(ctx context.Context, id int64) (*model.PipelineStage, error) {
	query := `SELECT ` + pipelineStageSelectColumns + ` FROM pipeline_stages WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.PipelineStage
	if err := scanStage(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrPipelineStageNotFound, err)
		}
		return nil, fmt.Errorf("failed to find stage: %w", err)
	}
	return &m, nil
}

func (r *PipelineStageRepo) ListByPipeline(ctx context.Context, pipelineID int64) ([]model.PipelineStage, error) {
	query := `SELECT ` + pipelineStageSelectColumns + ` FROM pipeline_stages WHERE pipeline_id = $1 ORDER BY position ASC`
	rows, err := r.pool.Query(ctx, query, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages: %w", err)
	}
	defer rows.Close()
	var out []model.PipelineStage
	for rows.Next() {
		var m model.PipelineStage
		if err := scanStage(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan stage: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PipelineStageRepo) Update(ctx context.Context, m *model.PipelineStage) error {
	query := `UPDATE pipeline_stages
		SET name = $2, position = $3, color = $4, is_terminal = $5, terminal_kind = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + pipelineStageSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.Name, m.Position, m.Color, m.IsTerminal, m.TerminalKind)
	if err := scanStage(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrPipelineStageNotFound, err)
		}
		return fmt.Errorf("failed to update stage: %w", err)
	}
	return nil
}

func (r *PipelineStageRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM pipeline_stages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete stage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrPipelineStageNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *PipelineStageRepo) MaxPosition(ctx context.Context, pipelineID int64) (float64, error) {
	var max *float64
	if err := r.pool.QueryRow(ctx, `SELECT MAX(position) FROM pipeline_stages WHERE pipeline_id = $1`, pipelineID).Scan(&max); err != nil {
		return 0, fmt.Errorf("failed to compute max position: %w", err)
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

// Rebalance renumbers all stage positions for the pipeline as 100, 200, 300, ...
// Returns the updated stages in their new order.
func (r *PipelineStageRepo) Rebalance(ctx context.Context, pipelineID int64) ([]model.PipelineStage, error) {
	if err := rebalancePositions(ctx, r.pool, rebalanceStageQueries, pipelineID); err != nil {
		return nil, err
	}
	return r.ListByPipeline(ctx, pipelineID)
}
