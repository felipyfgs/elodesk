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

var ErrInvitationNotFound = errors.New("invitation not found")
var ErrInvitationAlreadyPending = errors.New("invitation already pending")

const agentInvitationSelectColumns = "id, account_id, email, role, name, token_hash, expires_at, consumed_at, created_by, created_at, updated_at"

type agentInvitationScanner interface {
	Scan(dest ...any) error
}

func scanAgentInvitation(scanner agentInvitationScanner, m *model.AgentInvitation) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Email, &m.Role, &m.Name, &m.TokenHash, &m.ExpiresAt, &m.ConsumedAt, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
}

type AgentInvitationRepo struct {
	pool *pgxpool.Pool
}

func NewAgentInvitationRepo(pool *pgxpool.Pool) *AgentInvitationRepo {
	return &AgentInvitationRepo{pool: pool}
}

func (r *AgentInvitationRepo) Create(ctx context.Context, m *model.AgentInvitation) error {
	query := `INSERT INTO agent_invitations (account_id, email, role, name, token_hash, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Email, m.Role, m.Name, m.TokenHash, m.ExpiresAt, m.CreatedBy).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrInvitationAlreadyPending, err)
		}
		return fmt.Errorf("failed to create agent invitation: %w", err)
	}
	return nil
}

func (r *AgentInvitationRepo) FindByID(ctx context.Context, id int64) (*model.AgentInvitation, error) {
	query := `SELECT ` + agentInvitationSelectColumns + ` FROM agent_invitations WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.AgentInvitation
	if err := scanAgentInvitation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInvitationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find invitation by id: %w", err)
	}
	return &m, nil
}

func (r *AgentInvitationRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.AgentInvitation, error) {
	query := `SELECT ` + agentInvitationSelectColumns + ` FROM agent_invitations WHERE token_hash = $1`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var m model.AgentInvitation
	if err := scanAgentInvitation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInvitationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find invitation by token: %w", err)
	}
	return &m, nil
}

func (r *AgentInvitationRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.AgentInvitation, error) {
	query := `SELECT ` + agentInvitationSelectColumns + ` FROM agent_invitations WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}
	defer rows.Close()

	var result []model.AgentInvitation
	for rows.Next() {
		var m model.AgentInvitation
		if err := scanAgentInvitation(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *AgentInvitationRepo) MarkConsumedTx(ctx context.Context, tx pgx.Tx, id int64) error {
	now := time.Now()
	_, err := tx.Exec(ctx, `UPDATE agent_invitations SET consumed_at = $1, updated_at = $1 WHERE id = $2`, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark invitation consumed: %w", err)
	}
	return nil
}

func (r *AgentInvitationRepo) Delete(ctx context.Context, accountID, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM agent_invitations WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}
	return nil
}
