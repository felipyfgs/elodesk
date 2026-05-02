package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// rebalanceQueries holds the SQL templates needed to renumber position-ordered
// rows for either pipeline_stages (parent = pipeline_id) or pipeline_cards
// (parent = stage_id). Centralised to keep dupl-free repos.
type rebalanceQueries struct {
	selectIDs string
	updatePos string
}

var (
	rebalanceStageQueries = rebalanceQueries{
		selectIDs: `SELECT id FROM pipeline_stages WHERE pipeline_id = $1 ORDER BY position ASC FOR UPDATE`,
		updatePos: `UPDATE pipeline_stages SET position = $1, updated_at = NOW() WHERE id = $2`,
	}
	rebalanceCardQueries = rebalanceQueries{
		selectIDs: `SELECT id FROM pipeline_cards WHERE stage_id = $1 ORDER BY position ASC FOR UPDATE`,
		updatePos: `UPDATE pipeline_cards SET position = $1, updated_at = NOW() WHERE id = $2`,
	}
)

// rebalancePositions renumbers position-ordered children of a parent in a
// transaction so that gaps become 100, 200, 300, ...  Returns when the
// transaction commits successfully; the caller refreshes its view of the data.
func rebalancePositions(ctx context.Context, pool *pgxpool.Pool, q rebalanceQueries, parentID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, q.selectIDs, parentID)
	if err != nil {
		return fmt.Errorf("failed to load rows for rebalance: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()

	for i, id := range ids {
		newPos := float64((i + 1) * 100)
		if _, err := tx.Exec(ctx, q.updatePos, newPos, id); err != nil {
			return fmt.Errorf("failed to update position: %w", err)
		}
	}
	return tx.Commit(ctx)
}
