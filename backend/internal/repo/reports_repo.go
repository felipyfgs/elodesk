package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportsRepo struct {
	pool *pgxpool.Pool
}

func NewReportsRepo(pool *pgxpool.Pool) *ReportsRepo {
	return &ReportsRepo{pool: pool}
}

type OverviewReport struct {
	OpenCount               int            `json:"openCount"`
	ResolvedCount           int            `json:"resolvedCount"`
	FirstResponseAvgMinutes *float64       `json:"firstResponseAvgMinutes,omitempty"`
	ResolutionAvgMinutes    *float64       `json:"resolutionAvgMinutes,omitempty"`
	VolumeByDay             []VolumeByDay  `json:"volumeByDay"`
	StatusBreakdown         map[string]int `json:"statusBreakdown"`
}

type VolumeByDay struct {
	Day   time.Time `json:"day"`
	Total int       `json:"total"`
}

func (r *ReportsRepo) Overview(ctx context.Context, accountID int64, from, to time.Time) (*OverviewReport, error) {
	var openCount, resolvedCount, pendingCount, snoozedCount int
	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE status = 0) AS open_count,
			COUNT(*) FILTER (WHERE status = 1) AS resolved_count,
			COUNT(*) FILTER (WHERE status = 2) AS pending_count,
			COUNT(*) FILTER (WHERE status = 3) AS snoozed_count
		 FROM conversations
		 WHERE account_id = $1 AND created_at >= $2 AND created_at < $3`,
		accountID, from, to,
	).Scan(&openCount, &resolvedCount, &pendingCount, &snoozedCount)
	if err != nil {
		return nil, fmt.Errorf("overview status counts: %w", err)
	}

	volRows, err := r.pool.Query(ctx,
		`SELECT date_trunc('day', created_at)::date AS day, COUNT(*)
		 FROM conversations
		 WHERE account_id = $1 AND created_at >= $2 AND created_at < $3
		 GROUP BY day ORDER BY day ASC`,
		accountID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("overview volume: %w", err)
	}
	defer volRows.Close()

	var volume []VolumeByDay
	for volRows.Next() {
		var v VolumeByDay
		if err := volRows.Scan(&v.Day, &v.Total); err != nil {
			return nil, fmt.Errorf("scan volume row: %w", err)
		}
		volume = append(volume, v)
	}

	return &OverviewReport{
		OpenCount:       openCount,
		ResolvedCount:   resolvedCount,
		VolumeByDay:     volume,
		StatusBreakdown: map[string]int{"open": openCount, "resolved": resolvedCount, "pending": pendingCount, "snoozed": snoozedCount},
	}, nil
}

type ConversationReportRow struct {
	ID         int64     `json:"id"`
	DisplayID  int64     `json:"displayId"`
	AccountID  int64     `json:"accountId"`
	InboxID    int64     `json:"inboxId"`
	ContactID  int64     `json:"contactId"`
	AssigneeID *int64    `json:"assigneeId,omitempty"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ConversationReportFilter struct {
	AccountID int64
	From      time.Time
	To        time.Time
	InboxID   *int64
	LabelID   *int64
	Page      int
	PageSize  int
	Sort      string
}

func (r *ReportsRepo) Conversations(ctx context.Context, f ConversationReportFilter) ([]ConversationReportRow, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	where := "WHERE c.account_id = $1 AND c.created_at >= $2 AND c.created_at < $3"
	args := []any{f.AccountID, f.From, f.To}
	idx := 4
	if f.InboxID != nil {
		where += fmt.Sprintf(" AND c.inbox_id = $%d", idx)
		args = append(args, *f.InboxID)
		idx++
	}
	if f.LabelID != nil {
		where += fmt.Sprintf(` AND EXISTS(SELECT 1 FROM label_taggings lt WHERE lt.account_id = $1 AND lt.taggable_type='conversation' AND lt.taggable_id = c.id AND lt.label_id = $%d)`, idx)
		args = append(args, *f.LabelID)
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM conversations c "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count conversations report: %w", err)
	}

	order := "c.created_at DESC"
	switch f.Sort {
	case "created_at":
		order = "c.created_at ASC"
	case "-created_at":
		order = "c.created_at DESC"
	}
	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(`SELECT c.id, c.display_id, c.account_id, c.inbox_id, c.contact_id, c.assignee_id, c.status, c.created_at
		FROM conversations c %s ORDER BY %s LIMIT %d OFFSET %d`, where, order, f.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations report: %w", err)
	}
	defer rows.Close()

	var items []ConversationReportRow
	for rows.Next() {
		var row ConversationReportRow
		if err := rows.Scan(&row.ID, &row.DisplayID, &row.AccountID, &row.InboxID, &row.ContactID, &row.AssigneeID, &row.Status, &row.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan conversation row: %w", err)
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

type EntityMetric struct {
	EntityID   int64  `json:"entityId"`
	EntityName string `json:"entityName"`
	Total      int    `json:"total"`
	Resolved   int    `json:"resolved"`
	Open       int    `json:"open"`
}

func (r *ReportsRepo) EntityReport(ctx context.Context, accountID int64, entity string, from, to time.Time) ([]EntityMetric, error) {
	var query string
	switch entity {
	case "agents":
		query = `SELECT u.id, u.name,
			COUNT(c.id),
			COUNT(c.id) FILTER (WHERE c.status = 1),
			COUNT(c.id) FILTER (WHERE c.status = 0)
			FROM users u
			LEFT JOIN conversations c ON c.assignee_id = u.id AND c.account_id = $1 AND c.created_at >= $2 AND c.created_at < $3
			WHERE EXISTS (SELECT 1 FROM account_users au WHERE au.user_id = u.id AND au.account_id = $1)
			GROUP BY u.id, u.name ORDER BY u.name ASC`
	case "inboxes":
		query = `SELECT i.id, i.name,
			COUNT(c.id),
			COUNT(c.id) FILTER (WHERE c.status = 1),
			COUNT(c.id) FILTER (WHERE c.status = 0)
			FROM inboxes i
			LEFT JOIN conversations c ON c.inbox_id = i.id AND c.created_at >= $2 AND c.created_at < $3
			WHERE i.account_id = $1
			GROUP BY i.id, i.name ORDER BY i.name ASC`
	case "teams":
		query = `SELECT t.id, t.name,
			COUNT(c.id),
			COUNT(c.id) FILTER (WHERE c.status = 1),
			COUNT(c.id) FILTER (WHERE c.status = 0)
			FROM teams t
			LEFT JOIN conversations c ON c.team_id = t.id AND c.account_id = $1 AND c.created_at >= $2 AND c.created_at < $3
			WHERE t.account_id = $1
			GROUP BY t.id, t.name ORDER BY t.name ASC`
	case "labels":
		query = `SELECT l.id, l.title,
			COUNT(DISTINCT c.id),
			COUNT(DISTINCT c.id) FILTER (WHERE c.status = 1),
			COUNT(DISTINCT c.id) FILTER (WHERE c.status = 0)
			FROM labels l
			LEFT JOIN label_taggings lt ON lt.label_id = l.id AND lt.taggable_type = 'conversation'
			LEFT JOIN conversations c ON c.id = lt.taggable_id AND c.account_id = $1 AND c.created_at >= $2 AND c.created_at < $3
			WHERE l.account_id = $1
			GROUP BY l.id, l.title ORDER BY l.title ASC`
	default:
		return nil, fmt.Errorf("unknown entity %q", entity)
	}

	rows, err := r.pool.Query(ctx, query, accountID, from, to)
	if err != nil {
		return nil, fmt.Errorf("entity report query: %w", err)
	}
	defer rows.Close()

	var result []EntityMetric
	for rows.Next() {
		var m EntityMetric
		if err := rows.Scan(&m.EntityID, &m.EntityName, &m.Total, &m.Resolved, &m.Open); err != nil {
			return nil, fmt.Errorf("scan entity row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
