package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInboxAgentNotFound = errors.New("inbox agent not found")

const inboxAgentSelectColumns = "id, inbox_id, user_id, created_at"

type inboxAgentScanner interface {
	Scan(dest ...any) error
}

func scanInboxAgent(scanner inboxAgentScanner, m *model.InboxAgent) error {
	return scanner.Scan(&m.ID, &m.InboxID, &m.UserID, &m.CreatedAt)
}

type InboxAgentRepo struct {
	pool *pgxpool.Pool
}

func NewInboxAgentRepo(pool *pgxpool.Pool) *InboxAgentRepo {
	return &InboxAgentRepo{pool: pool}
}

func (r *InboxAgentRepo) ListByInbox(ctx context.Context, inboxID, accountID int64) ([]model.InboxAgent, error) {
	query := `SELECT ia.` + inboxAgentSelectColumns + ` FROM inbox_agents ia
		JOIN inboxes i ON i.id = ia.inbox_id
		WHERE ia.inbox_id = $1 AND i.account_id = $2
		ORDER BY ia.created_at ASC`
	rows, err := r.pool.Query(ctx, query, inboxID, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list inbox agents: %w", err)
	}
	defer rows.Close()

	var agents []model.InboxAgent
	for rows.Next() {
		var m model.InboxAgent
		if err := scanInboxAgent(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan inbox agent: %w", err)
		}
		agents = append(agents, m)
	}
	return agents, rows.Err()
}

func (r *InboxAgentRepo) SetByInbox(ctx context.Context, inboxID, accountID int64, userIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DELETE FROM inbox_agents WHERE inbox_id = $1`, inboxID)
	if err != nil {
		return fmt.Errorf("failed to clear inbox agents: %w", err)
	}

	for _, uid := range userIDs {
		query := `INSERT INTO inbox_agents (inbox_id, user_id)
			SELECT $1, $2 FROM inboxes WHERE id = $1 AND account_id = $3`
		_, err := tx.Exec(ctx, query, inboxID, uid, accountID)
		if err != nil {
			return fmt.Errorf("failed to add inbox agent: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *InboxAgentRepo) ExistsByInboxAndUser(ctx context.Context, inboxID, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM inbox_agents WHERE inbox_id = $1 AND user_id = $2)`,
		inboxID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check inbox agent: %w", err)
	}
	return exists, nil
}
