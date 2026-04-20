package repo

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogRepo struct {
	pool *pgxpool.Pool
}

func NewAuditLogRepo(pool *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{pool: pool}
}

func (r *AuditLogRepo) Create(ctx context.Context, accountID int64, userID *int64, action, entityType string, entityID *int64, metadata string, ipAddress net.IP, userAgent string) error {
	query := `INSERT INTO audit_logs (account_id, user_id, action, entity_type, entity_id, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, accountID, userID, action, entityType, entityID, metadata, ipAddress, userAgent)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

type AuditLogEntry struct {
	ID         int64
	AccountID  int64
	UserID     *int64
	Action     string
	EntityType *string
	EntityID   *int64
	Metadata   *string
	IPAddress  *net.IP
	UserAgent  *string
	CreatedAt  time.Time
}

func (r *AuditLogRepo) List(ctx context.Context, accountID int64, from, to, action, entityType string, userID *int64, page, pageSize int) ([]AuditLogEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	baseWhere := "WHERE account_id = $1"
	args := []any{accountID}
	argIdx := 2

	if from != "" {
		baseWhere += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, from)
		argIdx++
	}
	if to != "" {
		baseWhere += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, to)
		argIdx++
	}
	if action != "" {
		baseWhere += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	if entityType != "" {
		baseWhere += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}
	if userID != nil {
		baseWhere += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}

	// Count
	var total int
	countQuery := "SELECT COUNT(*) FROM audit_logs " + baseWhere
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Query
	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`SELECT id, account_id, user_id, action, entity_type, entity_id, metadata, ip_address, user_agent, created_at
		FROM audit_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var result []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		if err := rows.Scan(&e.ID, &e.AccountID, &e.UserID, &e.Action, &e.EntityType, &e.EntityID, &e.Metadata, &e.IPAddress, &e.UserAgent, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		result = append(result, e)
	}
	return result, total, rows.Err()
}

func (r *AuditLogRepo) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	res, err := r.pool.Exec(ctx, `DELETE FROM audit_logs WHERE created_at < NOW() - ($1 || ' days')::INTERVAL`, days)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old audit logs: %w", err)
	}
	return res.RowsAffected(), nil
}
