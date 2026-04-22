package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

const conversationSelectColumns = "id, account_id, inbox_id, status, assignee_id, team_id, contact_id, contact_inbox_id, display_id, uuid, pubsub_token, last_activity_at, additional_attributes, created_at, updated_at"

type conversationScanner interface {
	Scan(dest ...any) error
}

func scanConversation(scanner conversationScanner, m *model.Conversation) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.Status, &m.AssigneeID, &m.TeamID, &m.ContactID, &m.ContactInboxID, &m.DisplayID, &m.UUID, &m.PubsubToken, &m.LastActivityAt, &m.AdditionalAttrs, &m.CreatedAt, &m.UpdatedAt)
}

type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

func (r *ConversationRepo) Create(ctx context.Context, m *model.Conversation) error {
	query := `INSERT INTO conversations (account_id, inbox_id, status, assignee_id, contact_id, contact_inbox_id, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, display_id, uuid, last_activity_at, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.Status, m.AssigneeID, m.ContactID, m.ContactInboxID, m.AdditionalAttrs).
		Scan(&m.ID, &m.DisplayID, &m.UUID, &m.LastActivityAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Conversation, error) {
	query := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}
	return &m, nil
}

func (r *ConversationRepo) ToggleStatus(ctx context.Context, id, accountID int64, newStatus model.ConversationStatus) (*model.Conversation, error) {
	query := `UPDATE conversations SET status = $1, updated_at = NOW() WHERE id = $2 AND account_id = $3
		RETURNING ` + conversationSelectColumns
	row := r.pool.QueryRow(ctx, query, newStatus, id, accountID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to toggle conversation status: %w", err)
	}
	return &m, nil
}

type ConversationAssigneeType string

const (
	ConversationAssigneeTypeAll        ConversationAssigneeType = "all"
	ConversationAssigneeTypeMine       ConversationAssigneeType = "mine"
	ConversationAssigneeTypeUnassigned ConversationAssigneeType = "unassigned"
)

type ConversationSortKey string

const (
	ConversationSortLastActivityDesc ConversationSortKey = "last_activity_desc"
	ConversationSortLastActivityAsc  ConversationSortKey = "last_activity_asc"
	ConversationSortCreatedDesc      ConversationSortKey = "created_desc"
	ConversationSortCreatedAsc       ConversationSortKey = "created_asc"
)

func (s ConversationSortKey) orderClause() string {
	switch s {
	case ConversationSortLastActivityAsc:
		return "last_activity_at ASC"
	case ConversationSortCreatedDesc:
		return "created_at DESC"
	case ConversationSortCreatedAsc:
		return "created_at ASC"
	case ConversationSortLastActivityDesc:
		fallthrough
	default:
		return "last_activity_at DESC"
	}
}

type ConversationFilter struct {
	AccountID    int64
	InboxID      *int64
	Status       *model.ConversationStatus
	AssigneeID   *int64
	AssigneeType ConversationAssigneeType
	CurrentUser  *int64
	SortBy       ConversationSortKey
	ContactID    *int64
	Page         int
	PerPage      int
}

// conversationFilterClause builds the WHERE suffix common to listing and
// counting queries. It returns the SQL fragment plus the positional args.
func (f ConversationFilter) whereClause() (string, []any) {
	clause := ""
	args := []any{f.AccountID}
	argN := 2

	if f.InboxID != nil {
		clause += fmt.Sprintf(" AND inbox_id = $%d", argN)
		args = append(args, *f.InboxID)
		argN++
	}
	if f.Status != nil {
		clause += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *f.Status)
		argN++
	}
	if f.AssigneeID != nil {
		clause += fmt.Sprintf(" AND assignee_id = $%d", argN)
		args = append(args, *f.AssigneeID)
		argN++
	}

	switch f.AssigneeType {
	case ConversationAssigneeTypeMine:
		if f.CurrentUser != nil {
			clause += fmt.Sprintf(" AND assignee_id = $%d", argN)
			args = append(args, *f.CurrentUser)
		} else {
			// fail-closed: 'mine' without an authenticated user must return
			// no rows rather than degrading to 'all' (would leak tenant data).
			clause += " AND 1 = 0"
		}
	case ConversationAssigneeTypeUnassigned:
		clause += " AND assignee_id IS NULL"
	}

	return clause, args
}

// ListByAccountFiltered runs a parameterized filter clause (generated by
// filterquery.BuildSQL with startArgN=2) against conversations, scoped to
// account_id ($1). Reuses the same row scanner as ListByAccount so the DTO
// shape matches the regular listing endpoint.
func (r *ConversationRepo) ListByAccountFiltered(ctx context.Context, accountID int64, where string, filterArgs []any, sortBy ConversationSortKey, page, perPage int) ([]model.Conversation, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	args := append([]any{accountID}, filterArgs...)

	countQuery := "SELECT COUNT(*) FROM conversations WHERE account_id = $1"
	if where != "" {
		countQuery += " AND " + where
	}
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count filtered conversations: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	offset := (page - 1) * perPage
	dataQuery := "SELECT " + conversationSelectColumns + " FROM conversations WHERE account_id = $1"
	if where != "" {
		dataQuery += " AND " + where
	}
	dataQuery += fmt.Sprintf(" ORDER BY %s LIMIT %d OFFSET %d", sortBy.orderClause(), perPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list filtered conversations: %w", err)
	}
	defer rows.Close()

	var out []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan filtered conversation: %w", err)
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

func (r *ConversationRepo) ListByAccount(ctx context.Context, f ConversationFilter) ([]model.Conversation, int, error) {
	where, args := f.whereClause()

	var total int
	countQuery := `SELECT COUNT(*) FROM conversations WHERE account_id = $1` + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE account_id = $1` + where +
		fmt.Sprintf(" ORDER BY %s LIMIT %d OFFSET %d", f.SortBy.orderClause(), f.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var convos []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convos = append(convos, m)
	}
	return convos, total, rows.Err()
}

// ConversationMetaCounts holds conversation counts broken down by status and
// assignee dimension. Keys inside each status are "all", "mine", "unassigned".
type ConversationMetaCounts struct {
	Open     map[string]int `json:"open"`
	Pending  map[string]int `json:"pending"`
	Resolved map[string]int `json:"resolved"`
	Snoozed  map[string]int `json:"snoozed"`
}

// CountByStatusAndAssignee aggregates conversation counts for every
// (status × assignee_type) combination in a single query. Used to feed the
// conversation list tabs with live counters.
func (r *ConversationRepo) CountByStatusAndAssignee(ctx context.Context, accountID, currentUserID int64, inboxID *int64) (ConversationMetaCounts, error) {
	query := `SELECT status,
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE assignee_id = $2) AS mine,
		COUNT(*) FILTER (WHERE assignee_id IS NULL) AS unassigned
		FROM conversations
		WHERE account_id = $1`
	args := []any{accountID, currentUserID}
	if inboxID != nil {
		query += " AND inbox_id = $3"
		args = append(args, *inboxID)
	}
	query += " GROUP BY status"

	out := ConversationMetaCounts{
		Open:     map[string]int{"all": 0, "mine": 0, "unassigned": 0},
		Pending:  map[string]int{"all": 0, "mine": 0, "unassigned": 0},
		Resolved: map[string]int{"all": 0, "mine": 0, "unassigned": 0},
		Snoozed:  map[string]int{"all": 0, "mine": 0, "unassigned": 0},
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return out, fmt.Errorf("failed to count conversation meta: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status model.ConversationStatus
		var all, mine, unassigned int
		if err := rows.Scan(&status, &all, &mine, &unassigned); err != nil {
			return out, fmt.Errorf("failed to scan conversation meta: %w", err)
		}
		bucket := map[string]int{"all": all, "mine": mine, "unassigned": unassigned}
		switch status {
		case model.ConversationOpen:
			out.Open = bucket
		case model.ConversationPending:
			out.Pending = bucket
		case model.ConversationResolved:
			out.Resolved = bucket
		case model.ConversationSnoozed:
			out.Snoozed = bucket
		}
	}
	return out, rows.Err()
}

// AccountIDByID returns only the owning account id for a conversation.
// Used by callers that need to verify tenant ownership without fetching the
// full row and without already knowing the account (e.g. realtime joins).
func (r *ConversationRepo) AccountIDByID(ctx context.Context, id int64) (int64, error) {
	var accountID int64
	err := r.pool.QueryRow(ctx, `SELECT account_id FROM conversations WHERE id = $1`, id).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return 0, fmt.Errorf("failed to resolve conversation account: %w", err)
	}
	return accountID, nil
}

// FindByUUID resolves a conversation by its UUID, scoped to the given account
// and inbox to prevent cross-inbox thread hijacking.
func (r *ConversationRepo) FindByUUID(ctx context.Context, uuid string, accountID, inboxID int64) (*model.Conversation, error) {
	query := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE uuid = $1 AND account_id = $2 AND inbox_id = $3`
	row := r.pool.QueryRow(ctx, query, uuid, accountID, inboxID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("find conversation by uuid: %w", err)
	}
	return &m, nil
}

// FindByConvID is an alias used by the thread finder to hydrate a conversation
// from a message's conversation_id.
func (r *ConversationRepo) FindByConvID(ctx context.Context, convID, accountID int64) (*model.Conversation, error) {
	return r.FindByID(ctx, convID, accountID)
}

func (r *ConversationRepo) UpdateLastSeen(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET last_activity_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}
	return nil
}

func (r *ConversationRepo) UpdateAdditionalAttrs(ctx context.Context, id, accountID int64, attrs string) (*string, error) {
	query := `UPDATE conversations SET additional_attributes = $3, updated_at = NOW() WHERE id = $1 AND account_id = $2 RETURNING additional_attributes`
	var result *string
	if err := r.pool.QueryRow(ctx, query, id, accountID, attrs).Scan(&result); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to update additional_attributes: %w", err)
	}
	return result, nil
}

func (r *ConversationRepo) UpdateAssignment(ctx context.Context, id, accountID int64, assigneeID, teamID *int64) (*model.Conversation, error) {
	query := `UPDATE conversations SET
		assignee_id = COALESCE($3, assignee_id),
		team_id = COALESCE($4, team_id),
		updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + conversationSelectColumns
	row := r.pool.QueryRow(ctx, query, id, accountID, assigneeID, teamID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}
	return &m, nil
}

func (r *ConversationRepo) EnsureOpen(ctx context.Context, accountID, inboxID, contactID int64) (*model.Conversation, error) {
	query := `WITH existing AS (
		SELECT ` + conversationSelectColumns + ` FROM conversations
		WHERE account_id = $1 AND inbox_id = $2 AND contact_id = $3
		ORDER BY id DESC LIMIT 1
	)
	INSERT INTO conversations (account_id, inbox_id, status, contact_id)
	SELECT $1, $2, 0, $3
	WHERE NOT EXISTS (SELECT 1 FROM existing)
	ON CONFLICT DO NOTHING
	RETURNING ` + conversationSelectColumns
	row := r.pool.QueryRow(ctx, query, accountID, inboxID, contactID)
	var convo model.Conversation
	err := scanConversation(row, &convo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var found model.Conversation
			err2 := r.pool.QueryRow(ctx,
				`SELECT `+conversationSelectColumns+` FROM conversations
				 WHERE account_id = $1 AND inbox_id = $2 AND contact_id = $3
				 ORDER BY id DESC LIMIT 1`,
				accountID, inboxID, contactID,
			).Scan(&found.ID, &found.AccountID, &found.InboxID, &found.Status, &found.AssigneeID, &found.TeamID, &found.ContactID, &found.ContactInboxID, &found.DisplayID, &found.UUID, &found.PubsubToken, &found.LastActivityAt, &found.AdditionalAttrs, &found.CreatedAt, &found.UpdatedAt)
			if err2 != nil {
				return nil, fmt.Errorf("failed to ensure open conversation: %w", err2)
			}
			return &found, nil
		}
		return nil, fmt.Errorf("failed to ensure open conversation: %w", err)
	}
	return &convo, nil
}

func (r *ConversationRepo) FindByPubsubToken(ctx context.Context, pubsubToken string) (*model.Conversation, error) {
	query := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE pubsub_token = $1`
	row := r.pool.QueryRow(ctx, query, pubsubToken)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find conversation by pubsub_token: %w", err)
	}
	return &m, nil
}

func (r *ConversationRepo) CreateWithPubsubToken(ctx context.Context, m *model.Conversation) error {
	query := `INSERT INTO conversations (account_id, inbox_id, status, contact_id, pubsub_token)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, display_id, uuid, last_activity_at, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.InboxID, m.Status, m.ContactID, m.PubsubToken,
	).Scan(&m.ID, &m.DisplayID, &m.UUID, &m.LastActivityAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conversation with pubsub_token: %w", err)
	}
	return nil
}

func (r *ConversationRepo) UpdatePubsubToken(ctx context.Context, id int64, token *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET pubsub_token = $1, updated_at = NOW() WHERE id = $2`,
		token, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update pubsub_token: %w", err)
	}
	return nil
}

func (r *ConversationRepo) ListByContactID(ctx context.Context, contactID, accountID int64, page, perPage int) ([]model.Conversation, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	countQuery := `SELECT COUNT(*) FROM conversations WHERE contact_id = $1 AND account_id = $2`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, contactID, accountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations by contact: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	offset := (page - 1) * perPage
	dataQuery := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE contact_id = $1 AND account_id = $2 ORDER BY last_activity_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, dataQuery, contactID, accountID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations by contact: %w", err)
	}
	defer rows.Close()

	var convos []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convos = append(convos, m)
	}
	return convos, total, rows.Err()
}

func (r *ConversationRepo) ListByContactInboxID(ctx context.Context, ciID, accountID int64, page, perPage int) ([]model.Conversation, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	countQuery := `SELECT COUNT(*) FROM conversations WHERE contact_inbox_id = $1 AND account_id = $2`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, ciID, accountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations by contact_inbox: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	offset := (page - 1) * perPage
	dataQuery := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE contact_inbox_id = $1 AND account_id = $2 ORDER BY last_activity_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, dataQuery, ciID, accountID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations by contact_inbox: %w", err)
	}
	defer rows.Close()

	var convos []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convos = append(convos, m)
	}
	return convos, total, rows.Err()
}

func (r *ConversationRepo) FindLatestByContactInbox(ctx context.Context, ciID, accountID int64) (*model.Conversation, error) {
	query := `SELECT ` + conversationSelectColumns + ` FROM conversations
		WHERE contact_inbox_id = $1 AND account_id = $2 AND status IN (0, 2, 3)
		ORDER BY last_activity_at DESC LIMIT 1`
	row := r.pool.QueryRow(ctx, query, ciID, accountID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find latest conversation by contact_inbox: %w", err)
	}
	return &m, nil
}

func (r *ConversationRepo) UpdateContactID(ctx context.Context, id, accountID, contactID int64) (*model.Conversation, error) {
	query := `UPDATE conversations SET contact_id = $3, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + conversationSelectColumns
	row := r.pool.QueryRow(ctx, query, id, accountID, contactID)
	var m model.Conversation
	if err := scanConversation(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to update conversation contact_id: %w", err)
	}
	return &m, nil
}
