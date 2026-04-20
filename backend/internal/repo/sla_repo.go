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

var ErrSLANotFound = errors.New("sla policy not found")

const slaSelectColumns = "id, account_id, name, first_response_minutes, resolution_minutes, business_hours_only, created_at, updated_at"
const slaBindingSelectColumns = "id, sla_id, inbox_id, label_id"

type slaScanner interface {
	Scan(dest ...any) error
}

func scanSLA(scanner slaScanner, m *model.SLAPolicy) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Name, &m.FirstResponseMinutes, &m.ResolutionMinutes, &m.BusinessHoursOnly, &m.CreatedAt, &m.UpdatedAt)
}

func scanSLABinding(scanner slaScanner, b *model.SLABinding) error {
	return scanner.Scan(&b.ID, &b.SlaID, &b.InboxID, &b.LabelID)
}

type SLARepo struct {
	pool *pgxpool.Pool
}

func NewSLARepo(pool *pgxpool.Pool) *SLARepo {
	return &SLARepo{pool: pool}
}

func (r *SLARepo) Create(ctx context.Context, m *model.SLAPolicy) error {
	query := `INSERT INTO sla_policies (account_id, name, first_response_minutes, resolution_minutes, business_hours_only)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.FirstResponseMinutes, m.ResolutionMinutes, m.BusinessHoursOnly).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create sla policy: %w", err)
	}
	return nil
}

func (r *SLARepo) FindByID(ctx context.Context, id, accountID int64) (*model.SLAPolicy, error) {
	query := `SELECT ` + slaSelectColumns + ` FROM sla_policies WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.SLAPolicy
	if err := scanSLA(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrSLANotFound, err)
		}
		return nil, fmt.Errorf("failed to find sla policy: %w", err)
	}
	return &m, nil
}

func (r *SLARepo) ListByAccount(ctx context.Context, accountID int64) ([]model.SLAPolicy, error) {
	query := `SELECT ` + slaSelectColumns + ` FROM sla_policies WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sla policies: %w", err)
	}
	defer rows.Close()

	var result []model.SLAPolicy
	for rows.Next() {
		var m model.SLAPolicy
		if err := scanSLA(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan sla policy: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *SLARepo) Update(ctx context.Context, m *model.SLAPolicy) error {
	query := `UPDATE sla_policies SET name = $1, first_response_minutes = $2, resolution_minutes = $3, business_hours_only = $4, updated_at = NOW() WHERE id = $5 AND account_id = $6`
	_, err := r.pool.Exec(ctx, query, m.Name, m.FirstResponseMinutes, m.ResolutionMinutes, m.BusinessHoursOnly, m.ID, m.AccountID)
	if err != nil {
		return fmt.Errorf("failed to update sla policy: %w", err)
	}
	return nil
}

func (r *SLARepo) Delete(ctx context.Context, id, accountID int64) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM sla_policies WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete sla policy: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrSLANotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *SLARepo) SetBindings(ctx context.Context, slaID int64, bindings []model.SLABinding) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DELETE FROM sla_bindings WHERE sla_id = $1`, slaID)
	if err != nil {
		return fmt.Errorf("failed to delete old bindings: %w", err)
	}

	for _, b := range bindings {
		if b.InboxID == nil && b.LabelID == nil {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO sla_bindings (sla_id, inbox_id, label_id) VALUES ($1, $2, $3)`, slaID, b.InboxID, b.LabelID)
		if err != nil {
			return fmt.Errorf("failed to insert binding: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *SLARepo) GetBindings(ctx context.Context, slaIDs []int64) ([]model.SLABinding, error) {
	if len(slaIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + slaBindingSelectColumns + ` FROM sla_bindings WHERE sla_id = ANY($1)`
	rows, err := r.pool.Query(ctx, query, slaIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get sla bindings: %w", err)
	}
	defer rows.Close()

	var result []model.SLABinding
	for rows.Next() {
		var b model.SLABinding
		if err := scanSLABinding(rows, &b); err != nil {
			return nil, fmt.Errorf("failed to scan binding: %w", err)
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

type SLAReport struct {
	Total    int
	Met      int
	Breached int
	ByPolicy []SLAReportByPolicy
}

type SLAReportByPolicy struct {
	PolicyID   int64
	PolicyName string
	Total      int
	Breached   int
}

func (r *SLARepo) Report(ctx context.Context, accountID int64, from, to string) (*SLAReport, error) {
	where := "WHERE c.account_id = $1 AND c.sla_policy_id IS NOT NULL"
	args := []any{accountID}
	idx := 2
	if from != "" {
		where += fmt.Sprintf(" AND c.created_at >= $%d", idx)
		args = append(args, from)
		idx++
	}
	if to != "" {
		where += fmt.Sprintf(" AND c.created_at <= $%d", idx)
		args = append(args, to)
	}

	var total, breached int
	row := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE TRUE), COUNT(*) FILTER (WHERE c.sla_breached = TRUE) FROM conversations c `+where,
		args...)
	if err := row.Scan(&total, &breached); err != nil {
		return nil, fmt.Errorf("failed to aggregate sla report: %w", err)
	}

	byPolicyQuery := `SELECT sp.id, sp.name, COUNT(c.id), COUNT(c.id) FILTER (WHERE c.sla_breached = TRUE)
		FROM conversations c
		JOIN sla_policies sp ON sp.id = c.sla_policy_id
		` + where + ` GROUP BY sp.id, sp.name ORDER BY sp.name`
	rows, err := r.pool.Query(ctx, byPolicyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sla report by policy: %w", err)
	}
	defer rows.Close()

	var byPolicy []SLAReportByPolicy
	for rows.Next() {
		var p SLAReportByPolicy
		if err := rows.Scan(&p.PolicyID, &p.PolicyName, &p.Total, &p.Breached); err != nil {
			return nil, fmt.Errorf("failed to scan sla report policy: %w", err)
		}
		byPolicy = append(byPolicy, p)
	}
	return &SLAReport{
		Total:    total,
		Met:      total - breached,
		Breached: breached,
		ByPolicy: byPolicy,
	}, nil
}

// AttachIfUnset resolves the SLA policy for a conversation's inbox (via
// sla_bindings) and sets sla_policy_id, first-response and resolution due-at
// columns if they are still NULL. Returns the applied policy id (0 when none
// matched). The operation is idempotent — subsequent calls are no-ops once a
// policy is attached.
func (r *SLARepo) AttachIfUnset(ctx context.Context, accountID, conversationID int64) (int64, error) {
	query := `UPDATE conversations c
		SET sla_policy_id = sp.id,
		    sla_first_response_due_at = NOW() + (sp.first_response_minutes || ' minutes')::INTERVAL,
		    sla_resolution_due_at = NOW() + (sp.resolution_minutes || ' minutes')::INTERVAL,
		    updated_at = NOW()
		FROM sla_bindings sb
		JOIN sla_policies sp ON sp.id = sb.sla_id
		WHERE c.id = $1 AND c.account_id = $2
		  AND c.sla_policy_id IS NULL
		  AND sb.inbox_id = c.inbox_id
		  AND sp.account_id = c.account_id
		RETURNING sp.id`
	var policyID int64
	err := r.pool.QueryRow(ctx, query, conversationID, accountID).Scan(&policyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to attach sla policy: %w", err)
	}
	return policyID, nil
}

// BreachedCandidates returns conversations in the given account that are past
// their SLA due-at and not yet flagged as breached. Used by the periodic breach
// detection job.
type BreachCandidate struct {
	ID         int64
	AccountID  int64
	InboxID    int64
	AssigneeID *int64
	PolicyID   int64
	DueAt      time.Time
	Kind       string // "first_response" or "resolution"
}

// ListBreachCandidates returns conversations whose first-response or resolution
// due-at has passed and that are not yet flagged as breached. Used by the
// periodic breach detection job. Results are limited to avoid unbounded scans.
func (r *SLARepo) ListBreachCandidates(ctx context.Context, limit int) ([]BreachCandidate, error) {
	if limit <= 0 {
		limit = 500
	}
	query := `SELECT id, account_id, inbox_id, assignee_id, sla_policy_id,
		sla_first_response_due_at, sla_resolution_due_at
		FROM conversations
		WHERE sla_breached = FALSE
		  AND sla_policy_id IS NOT NULL
		  AND status IN (0, 2)
		  AND (
		    (sla_first_response_due_at IS NOT NULL AND sla_first_response_due_at < NOW())
		    OR
		    (sla_resolution_due_at IS NOT NULL AND sla_resolution_due_at < NOW())
		  )
		ORDER BY sla_first_response_due_at NULLS LAST
		LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list breach candidates: %w", err)
	}
	defer rows.Close()

	var result []BreachCandidate
	for rows.Next() {
		var c BreachCandidate
		var firstDue, resDue *time.Time
		if err := rows.Scan(&c.ID, &c.AccountID, &c.InboxID, &c.AssigneeID, &c.PolicyID, &firstDue, &resDue); err != nil {
			return nil, fmt.Errorf("failed to scan breach candidate: %w", err)
		}
		now := time.Now().UTC()
		switch {
		case firstDue != nil && now.After(*firstDue):
			c.DueAt = *firstDue
			c.Kind = "first_response"
		case resDue != nil && now.After(*resDue):
			c.DueAt = *resDue
			c.Kind = "resolution"
		default:
			continue
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// MarkBreached flags a conversation as SLA-breached. Idempotent.
func (r *SLARepo) MarkBreached(ctx context.Context, conversationID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET sla_breached = TRUE, updated_at = NOW() WHERE id = $1 AND sla_breached = FALSE`,
		conversationID)
	if err != nil {
		return fmt.Errorf("failed to mark sla breached: %w", err)
	}
	return nil
}

func (r *SLARepo) FindByInbox(ctx context.Context, accountID, inboxID int64) (*model.SLAPolicy, error) {
	query := `SELECT ` + slaSelectColumns + ` FROM sla_policies sp
		JOIN sla_bindings sb ON sb.sla_id = sp.id
		WHERE sp.account_id = $1 AND sb.inbox_id = $2
		LIMIT 1`
	row := r.pool.QueryRow(ctx, query, accountID, inboxID)
	var m model.SLAPolicy
	if err := scanSLA(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find sla by inbox: %w", err)
	}
	return &m, nil
}
