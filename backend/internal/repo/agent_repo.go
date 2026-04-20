package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentMember struct {
	ID           int64
	UserID       int64
	Name         string
	Email        string
	Role         int
	InvitationID *int64
	InvitedEmail *string
	LastActiveAt *string
	CreatedAt    string
}

type AgentRepo struct {
	pool *pgxpool.Pool
}

func NewAgentRepo(pool *pgxpool.Pool) *AgentRepo {
	return &AgentRepo{pool: pool}
}

func (r *AgentRepo) ListByAccount(ctx context.Context, accountID int64) ([]AgentMember, error) {
	query := `SELECT
		au.id,
		au.user_id,
		COALESCE(u.name, COALESCE(inv.name, inv.email)) as name,
		COALESCE(u.email, inv.email) as email,
		au.role,
		inv.id as invitation_id,
		CASE WHEN u.id IS NULL THEN inv.email ELSE NULL END as invited_email,
		NULL::text as last_active_at,
		au.created_at
	FROM account_users au
	JOIN users u ON u.id = au.user_id
	LEFT JOIN agent_invitations inv ON inv.account_id = au.account_id AND lower(inv.email) = lower(u.email) AND inv.consumed_at IS NOT NULL
	WHERE au.account_id = $1
	ORDER BY u.name ASC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var result []AgentMember
	for rows.Next() {
		var m AgentMember
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.Email, &m.Role, &m.InvitationID, &m.InvitedEmail, &m.LastActiveAt, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *AgentRepo) CountOwners(ctx context.Context, accountID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM account_users WHERE account_id = $1 AND role = 2`, accountID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count owners: %w", err)
	}
	return count, nil
}

func (r *AgentRepo) UpdateRole(ctx context.Context, accountID, userID int64, role int) error {
	_, err := r.pool.Exec(ctx, `UPDATE account_users SET role = $1 WHERE account_id = $2 AND user_id = $3`, role, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to update agent role: %w", err)
	}
	return nil
}

func (r *AgentRepo) Remove(ctx context.Context, accountID, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM account_users WHERE account_id = $1 AND user_id = $2`, accountID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove agent: %w", err)
	}
	return nil
}

func (r *AgentRepo) GetRole(ctx context.Context, accountID, userID int64) (int, error) {
	var role int
	err := r.pool.QueryRow(ctx, `SELECT role FROM account_users WHERE account_id = $1 AND user_id = $2`, accountID, userID).Scan(&role)
	if err != nil {
		return 0, fmt.Errorf("failed to get agent role: %w", err)
	}
	return role, nil
}

func (r *AgentRepo) IsMember(ctx context.Context, accountID, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM account_users WHERE account_id = $1 AND user_id = $2)`, accountID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check agent membership: %w", err)
	}
	return exists, nil
}
