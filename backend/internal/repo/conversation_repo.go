package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/internal/dto"
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
	query := `UPDATE conversations SET status = $1, updated_at = NOW()
		WHERE id = $2 AND account_id = $3
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

// Delete removes the conversation and all its messages/participants via
// ON DELETE CASCADE. The audit trail survives in audit_logs; the data is
// permanently gone — there is no soft-delete for conversations.
func (r *ConversationRepo) Delete(ctx context.Context, id, accountID int64) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM conversations WHERE id = $1 AND account_id = $2`,
		id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
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
	TeamID       *int64
	Query        string
	Labels       []string
	SortBy       ConversationSortKey
	ContactID    *int64
	// Unread, quando true, restringe a conversas com pelo menos uma mensagem
	// incoming (message_type=0) posterior ao assignee_last_seen_at — espelha
	// a definição de "não lida" usada na UI.
	Unread bool
	// ConversationType implementa o filtro `unattended` do Chatwoot:
	// first_reply_created_at IS NULL OU waiting_since IS NOT NULL.
	// Hoje só "unattended" é tratado; "mention"/"participating" ficam reservados.
	ConversationType string
	Page             int
	PerPage          int
}

// whereClause builds the WHERE suffix common to listing and counting queries.
// It returns the SQL fragment plus the positional args. Callers that need to
// search by contact name (q) or labels should use ListByAccountFiltered or
// the hydrated list path instead — those filters require JOINs that are not
// available in the simple conversations-only query.
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
	if f.TeamID != nil {
		clause += fmt.Sprintf(" AND team_id = $%d", argN)
		args = append(args, *f.TeamID)
		argN++
	}
	if f.ContactID != nil {
		clause += fmt.Sprintf(" AND contact_id = $%d", argN)
		args = append(args, *f.ContactID)
		argN++
	}

	switch f.AssigneeType {
	case ConversationAssigneeTypeMine:
		if f.CurrentUser != nil {
			clause += fmt.Sprintf(" AND assignee_id = $%d", argN)
			args = append(args, *f.CurrentUser)
			argN++
		} else {
			clause += " AND 1 = 0"
		}
	case ConversationAssigneeTypeUnassigned:
		clause += " AND assignee_id IS NULL"
	}

	// `unattended`: cliente esperando resposta. Como a tabela `conversations`
	// não tem as colunas `first_reply_created_at`/`waiting_since` (Chatwoot
	// tem; o nosso schema não migrou), derivamos do histórico de mensagens:
	// a conversa está não-atendida quando a mensagem mais recente é incoming
	// (message_type=0) — ou seja, ninguém respondeu desde a última fala do
	// cliente. Conversas sem mensagens não entram (LIMIT 1 retorna NULL).
	if f.ConversationType == "unattended" {
		clause += " AND (SELECT m.message_type FROM messages m" +
			" WHERE m.conversation_id = conversations.id" +
			" ORDER BY m.created_at DESC LIMIT 1) = 0"
	}

	// `unread`: pelo menos uma mensagem incoming (message_type=0) ainda não
	// vista pelo agente atribuído. Usa EXISTS pra encerrar a varredura na
	// primeira mensagem candidata. Conversas sem assignee_last_seen_at também
	// contam como não-lidas se houver qualquer mensagem incoming.
	if f.Unread {
		clause += " AND EXISTS (SELECT 1 FROM messages m" +
			" WHERE m.conversation_id = conversations.id" +
			" AND m.message_type = 0" +
			" AND (conversations.assignee_last_seen_at IS NULL OR m.created_at > conversations.assignee_last_seen_at))"
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
// assignee dimension. Keys inside each status are "all", "mine", "unassigned",
// "unread". O bucket "unread" conta conversas com pelo menos 1 mensagem
// incoming pós-assignee_last_seen_at — global, sem particionar por assignee
// (decisão de produto: badge "Não lidas" do frontend é sempre global).
type ConversationMetaCounts struct {
	Open     map[string]int `json:"open"`
	Pending  map[string]int `json:"pending"`
	Resolved map[string]int `json:"resolved"`
	Snoozed  map[string]int `json:"snoozed"`
}

// CountByStatusAndAssignee aggregates conversation counts for every
// (status × assignee_type) combination in a single query. Used to feed the
// conversation list tabs with live counters. CTE `unread_convs` pré-computa
// uma vez os ids de conversas com unread > 0 (em vez de subquery aninhada
// por linha) e o LEFT JOIN materializa o flag pra cada conversa. Mantém o
// custo em uma única passada na tabela de mensagens.
func (r *ConversationRepo) CountByStatusAndAssignee(ctx context.Context, accountID, currentUserID int64, inboxID *int64) (ConversationMetaCounts, error) {
	inboxClause := ""
	args := []any{accountID, currentUserID}
	if inboxID != nil {
		inboxClause = " AND cv.inbox_id = $3"
		args = append(args, *inboxID)
	}

	query := `WITH unread_convs AS (
		SELECT DISTINCT cv.id
		  FROM conversations cv
		  JOIN messages m ON m.conversation_id = cv.id
		 WHERE cv.account_id = $1
		   AND m.message_type = 0
		   AND (cv.assignee_last_seen_at IS NULL OR m.created_at > cv.assignee_last_seen_at)` + inboxClause + `
	)
	SELECT cv.status,
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE cv.assignee_id = $2) AS mine,
		COUNT(*) FILTER (WHERE cv.assignee_id IS NULL) AS unassigned,
		COUNT(*) FILTER (WHERE u.id IS NOT NULL) AS unread_all
		FROM conversations cv
		LEFT JOIN unread_convs u ON u.id = cv.id
		WHERE cv.account_id = $1` + inboxClause + `
		GROUP BY cv.status`

	out := ConversationMetaCounts{
		Open:     map[string]int{"all": 0, "mine": 0, "unassigned": 0, "unread": 0},
		Pending:  map[string]int{"all": 0, "mine": 0, "unassigned": 0, "unread": 0},
		Resolved: map[string]int{"all": 0, "mine": 0, "unassigned": 0, "unread": 0},
		Snoozed:  map[string]int{"all": 0, "mine": 0, "unassigned": 0, "unread": 0},
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return out, fmt.Errorf("failed to count conversation meta: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status model.ConversationStatus
		var all, mine, unassigned, unread int
		if err := rows.Scan(&status, &all, &mine, &unassigned, &unread); err != nil {
			return out, fmt.Errorf("failed to scan conversation meta: %w", err)
		}
		bucket := map[string]int{"all": all, "mine": mine, "unassigned": unassigned, "unread": unread}
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

// ConversationHydrated bundles a Conversation row with the immediate
// relations the Chatwoot-shape DTO needs: contact, inbox, optional assignee
// and team, the last non-activity message, the unread count, and the labels.
// Returned by FindByIDFull and ListByAccountFull so the service layer can
// build dto.ConversationFullRow without further repo calls.
type ConversationHydrated struct {
	Conversation           model.Conversation
	Contact                model.Contact
	Inbox                  model.Inbox
	HmacVerified           bool
	Assignee               *model.User
	Team                   *model.Team
	UnreadCount            int
	Labels                 []string
	LastNonActivityMessage *model.Message
}

const conversationHydratedColumns = `
	cv.id, cv.account_id, cv.inbox_id, cv.status, cv.assignee_id, cv.team_id,
	cv.contact_id, cv.contact_inbox_id, cv.display_id, cv.uuid, cv.pubsub_token,
	cv.last_activity_at, cv.additional_attributes, cv.created_at, cv.updated_at,
	c.id, c.account_id, c.name, c.email, c.phone_number, c.phone_e164,
	c.identifier, c.additional_attributes, c.avatar_url, c.blocked,
	c.last_activity_at, c.created_at, c.updated_at,
	i.id, i.account_id, i.channel_id, i.name, i.channel_type, i.created_at, i.updated_at,
	COALESCE(ci.hmac_verified, false),
	u.id, u.email, u.name, u.avatar_url, u.mfa_enabled, u.created_at, u.updated_at,
	t.id, t.account_id, t.name, t.description, t.allow_auto_assign, t.created_at, t.updated_at,
	(SELECT COUNT(*) FROM messages WHERE conversation_id = cv.id AND message_type = 0
		AND (cv.assignee_last_seen_at IS NULL OR created_at > cv.assignee_last_seen_at))::int,
	lm.id, lm.account_id, lm.inbox_id, lm.conversation_id, lm.message_type, lm.content_type,
	lm.content, lm.source_id, lm.private, lm.status, lm.content_attributes,
	lm.sender_type, lm.sender_id, lm.sender_contact_id, lm.external_source_ids,
	lm.created_at, lm.updated_at, lm.deleted_at`

const conversationHydratedFrom = `
	FROM conversations cv
	INNER JOIN contacts c ON c.id = cv.contact_id AND c.account_id = cv.account_id
	INNER JOIN inboxes i ON i.id = cv.inbox_id AND i.account_id = cv.account_id
	LEFT JOIN contact_inboxes ci ON ci.id = cv.contact_inbox_id
	LEFT JOIN users u ON u.id = cv.assignee_id
	LEFT JOIN teams t ON t.id = cv.team_id AND t.account_id = cv.account_id
	LEFT JOIN LATERAL (
		SELECT * FROM messages m
		 WHERE m.conversation_id = cv.id AND m.message_type < 2 AND m.deleted_at IS NULL
		 ORDER BY m.created_at DESC LIMIT 1
	) lm ON true`

func scanConversationHydrated(scanner conversationScanner, h *ConversationHydrated) error {
	cv := &h.Conversation
	c := &h.Contact
	i := &h.Inbox
	var (
		uID                                                  *int64
		uEmail, uName                                        *string
		uAvatarURL                                           *string
		uMfaEnabled                                          *bool
		uCreatedAt, uUpdatedAt                               *time.Time
		tID, tAccountID                                      *int64
		tName                                                *string
		tDescription                                         *string
		tAllowAutoAssign                                     *bool
		tCreatedAt, tUpdatedAt                               *time.Time
		lmID                                                 *int64
		lmAccountID, lmInboxID, lmConversationID             *int64
		lmMessageType                                        *model.MessageType
		lmContentType                                        *model.MessageContentType
		lmContent, lmSourceID                                *string
		lmPrivate                                            *bool
		lmStatus                                             *model.MessageStatus
		lmContentAttrs                                       *string
		lmSenderType                                         *string
		lmSenderID, lmSenderContactID                        *int64
		lmExternalSourceIDs                                  *string
		lmCreatedAt, lmUpdatedAt                             *time.Time
		lmDeletedAt                                          *time.Time
	)
	if err := scanner.Scan(
		&cv.ID, &cv.AccountID, &cv.InboxID, &cv.Status, &cv.AssigneeID, &cv.TeamID,
		&cv.ContactID, &cv.ContactInboxID, &cv.DisplayID, &cv.UUID, &cv.PubsubToken,
		&cv.LastActivityAt, &cv.AdditionalAttrs, &cv.CreatedAt, &cv.UpdatedAt,
		&c.ID, &c.AccountID, &c.Name, &c.Email, &c.PhoneNumber, &c.PhoneE164,
		&c.Identifier, &c.AdditionalAttrs, &c.AvatarURL, &c.Blocked,
		&c.LastActivityAt, &c.CreatedAt, &c.UpdatedAt,
		&i.ID, &i.AccountID, &i.ChannelID, &i.Name, &i.ChannelType, &i.CreatedAt, &i.UpdatedAt,
		&h.HmacVerified,
		&uID, &uEmail, &uName, &uAvatarURL, &uMfaEnabled, &uCreatedAt, &uUpdatedAt,
		&tID, &tAccountID, &tName, &tDescription, &tAllowAutoAssign, &tCreatedAt, &tUpdatedAt,
		&h.UnreadCount,
		&lmID, &lmAccountID, &lmInboxID, &lmConversationID, &lmMessageType, &lmContentType,
		&lmContent, &lmSourceID, &lmPrivate, &lmStatus, &lmContentAttrs,
		&lmSenderType, &lmSenderID, &lmSenderContactID, &lmExternalSourceIDs,
		&lmCreatedAt, &lmUpdatedAt, &lmDeletedAt,
	); err != nil {
		return err
	}
	if uID != nil {
		h.Assignee = &model.User{
			ID:         *uID,
			Email:      derefStr(uEmail),
			Name:       derefStr(uName),
			AvatarURL:  uAvatarURL,
			MfaEnabled: derefBool(uMfaEnabled),
			CreatedAt:  derefTime(uCreatedAt),
			UpdatedAt:  derefTime(uUpdatedAt),
		}
	}
	if tID != nil {
		h.Team = &model.Team{
			ID:              *tID,
			AccountID:       derefInt64(tAccountID),
			Name:            derefStr(tName),
			Description:     tDescription,
			AllowAutoAssign: derefBool(tAllowAutoAssign),
			CreatedAt:       derefTime(tCreatedAt),
			UpdatedAt:       derefTime(tUpdatedAt),
		}
	}
	if lmID != nil {
		h.LastNonActivityMessage = &model.Message{
			ID:                *lmID,
			AccountID:         derefInt64(lmAccountID),
			InboxID:           derefInt64(lmInboxID),
			ConversationID:    derefInt64(lmConversationID),
			MessageType:       *lmMessageType,
			ContentType:       *lmContentType,
			Content:           lmContent,
			SourceID:          lmSourceID,
			Private:           derefBool(lmPrivate),
			Status:            *lmStatus,
			ContentAttrs:      lmContentAttrs,
			SenderType:        lmSenderType,
			SenderID:          lmSenderID,
			SenderContactID:   lmSenderContactID,
			ExternalSourceIDs: lmExternalSourceIDs,
			CreatedAt:         derefTime(lmCreatedAt),
			UpdatedAt:         derefTime(lmUpdatedAt),
			DeletedAt:         lmDeletedAt,
		}
	}
	h.Labels = []string{}
	return nil
}

// FindByIDFull returns a single conversation with every relation needed by
// dto.ConversationToRespFull hydrated. Account-scoped — the join on
// `cv.account_id = $1` plus `c.account_id = cv.account_id` ensures cross-
// tenant rows cannot leak even via a numeric id collision.
func (r *ConversationRepo) FindByIDFull(ctx context.Context, accountID, id int64) (*ConversationHydrated, error) {
	query := `SELECT ` + conversationHydratedColumns + conversationHydratedFrom +
		` WHERE cv.account_id = $1 AND cv.id = $2`
	row := r.pool.QueryRow(ctx, query, accountID, id)
	var h ConversationHydrated
	if err := scanConversationHydrated(row, &h); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
		}
		return nil, fmt.Errorf("failed to find conversation full: %w", err)
	}
	return &h, nil
}

// ListByAccountFull returns the hydrated rows for the default conversation
// list (account-scoped, ordered by last_activity_at DESC). Caller controls
// pagination. status==nil means no filter; otherwise rows match exactly.
// Additional filters are supported via ConversationFilter; pass the zero
// value for fields you don't need.
func (r *ConversationRepo) ListByAccountFull(ctx context.Context, accountID int64, status *model.ConversationStatus, limit, offset int) ([]ConversationHydrated, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{accountID}
	where := ` WHERE cv.account_id = $1`
	if status != nil {
		where += ` AND cv.status = $2`
		args = append(args, *status)
	}
	query := `SELECT ` + conversationHydratedColumns + conversationHydratedFrom + where +
		fmt.Sprintf(` ORDER BY cv.last_activity_at DESC LIMIT %d OFFSET %d`, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations full: %w", err)
	}
	defer rows.Close()
	var out []ConversationHydrated
	for rows.Next() {
		var h ConversationHydrated
		if err := scanConversationHydrated(rows, &h); err != nil {
			return nil, fmt.Errorf("failed to scan conversation full: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func derefStr(p *string) string  { if p != nil { return *p }; return "" }
func derefBool(p *bool) bool     { if p != nil { return *p }; return false }
func derefInt64(p *int64) int64  { if p != nil { return *p }; return 0 }
func derefTime(p *time.Time) time.Time {
	if p != nil {
		return *p
	}
	return time.Time{}
}

// ConversationHydratedToFullRow bridges the repo's hydrated row to the DTO
// input expected by ConversationToRespFull. Defined here (and exported) so
// the service layer can build realtime payloads without importing handler.
// LastNonActivitySender is left nil (it is resolved by the caller when needed).
func ConversationHydratedToFullRow(h *ConversationHydrated) dto.ConversationFullRow {
	return dto.ConversationFullRow{
		Conversation:           h.Conversation,
		Contact:                h.Contact,
		Inbox:                  h.Inbox,
		HmacVerified:           h.HmacVerified,
		Assignee:               h.Assignee,
		Team:                   h.Team,
		UnreadCount:            h.UnreadCount,
		Labels:                 h.Labels,
		LastNonActivityMessage: h.LastNonActivityMessage,
	}
}

// CountByFilter returns the four assignee-dimension counts (mine, assigned,
// unassigned, all) for the given filter. Unlike CountByStatusAndAssignee this
// is flat — it does not break down by status — and it respects all active
// filters (inbox_id, status, team_id, assignee_id, etc.) so it can be used
// as meta counts in the List endpoint envelope.
func (r *ConversationRepo) CountByFilter(ctx context.Context, f ConversationFilter) (dto.ConversationListMeta, error) {
	where, args := f.whereClause()

	query := `SELECT
		COUNT(*) AS all_count,
		COUNT(*) FILTER (WHERE assignee_id IS NOT NULL) AS assigned_count,
		COUNT(*) FILTER (WHERE assignee_id IS NULL) AS unassigned_count`
	queryArgs := make([]any, len(args))
	copy(queryArgs, args)
	if f.CurrentUser != nil {
		query += fmt.Sprintf(`, COUNT(*) FILTER (WHERE assignee_id = $%d) AS mine_count`, len(args)+1)
		queryArgs = append(queryArgs, *f.CurrentUser)
	} else {
		query += `, 0 AS mine_count`
	}
	query += ` FROM conversations WHERE account_id = $1` + where

	var meta dto.ConversationListMeta
	if err := r.pool.QueryRow(ctx, query, queryArgs...).Scan(&meta.AllCount, &meta.AssignedCount, &meta.UnassignedCount, &meta.MineCount); err != nil {
		return meta, fmt.Errorf("failed to count conversations by filter: %w", err)
	}
	return meta, nil
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

// CountUnread devolve o número de mensagens incoming não vistas pelo
// assignee (mesmo cálculo que conversationHydratedColumns usa). Usado pelo
// broadcast de message.created/updated para que o badge atualize em tempo
// real sem precisar reidratar a conversa inteira.
func (r *ConversationRepo) CountUnread(ctx context.Context, id, accountID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages m
		  JOIN conversations cv ON cv.id = m.conversation_id
		 WHERE cv.id = $1 AND cv.account_id = $2
		   AND m.message_type = 0
		   AND (cv.assignee_last_seen_at IS NULL OR m.created_at > cv.assignee_last_seen_at)`,
		id, accountID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}
	return count, nil
}

// UpdateAssigneeLastSeen marks o agent atribuído como tendo lido a conversa
// até o instante atual. Usado pelo endpoint /update_last_seen quando o agente
// abre a thread no dashboard — zera o unread_count derivado em
// conversationHydratedColumns.
func (r *ConversationRepo) UpdateAssigneeLastSeen(ctx context.Context, id, accountID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET assignee_last_seen_at = NOW() WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to update assignee_last_seen_at: %w", err)
	}
	return nil
}

// UpdateLastActivity bumps last_activity_at to the given timestamp without
// touching updated_at. Called from MessageService.Create after each new
// message so the conversation list ordering reflects message arrival time.
// Guarded by GREATEST so out-of-order events cannot move the timestamp
// backwards.
func (r *ConversationRepo) UpdateLastActivity(ctx context.Context, id int64, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET last_activity_at = GREATEST(last_activity_at, $2) WHERE id = $1`, id, at)
	if err != nil {
		return fmt.Errorf("failed to update conversation last_activity_at: %w", err)
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
	// Direct assignment (not COALESCE) so callers can clear the field by
	// passing nil. The handler always sends both fields from the request body,
	// so this is a full overwrite, not a partial update.
	query := `UPDATE conversations SET
		assignee_id = $3,
		team_id = $4,
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
