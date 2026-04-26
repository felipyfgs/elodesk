package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrParticipantNotFound = errors.New("participant not found")

// Participant is the in-repo shape of a row in the `participants` table.
// Defined here (not in model/) until other repos need it — keeps the boundary
// minimal until participants are referenced elsewhere.
type Participant struct {
	ID             int64
	AccountID      int64
	ConversationID int64
	ContactID      int64
	Role           string
	CreatedAt      pgxNullableTimeMarker
	UpdatedAt      pgxNullableTimeMarker
}

// pgxNullableTimeMarker exists only so the field type stays free for callers
// that want to scan into time.Time directly. The repo never reads these
// fields from the application side; it only round-trips them.
type pgxNullableTimeMarker = any

// ParticipantWithContact is what List returns: participant fields + the
// contact row hydrated, so handler can build ParticipantResp without a
// follow-up query.
type ParticipantWithContact struct {
	ID        int64
	Role      string
	Contact   model.Contact
}

const participantSelectColumns = "id, account_id, conversation_id, contact_id, role, created_at, updated_at"

type ParticipantRepo struct {
	pool *pgxpool.Pool
}

func NewParticipantRepo(pool *pgxpool.Pool) *ParticipantRepo {
	return &ParticipantRepo{pool: pool}
}

// Create inserts a new participant. Conflicts on (conversation_id, contact_id)
// are surfaced as a wrapped pgx error so the handler can decide between 409
// and silent-upsert depending on the caller.
func (r *ParticipantRepo) Create(ctx context.Context, accountID, convID, contactID int64, role string) (*Participant, error) {
	if role == "" {
		role = "member"
	}
	query := `INSERT INTO participants (account_id, conversation_id, contact_id, role)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + participantSelectColumns
	row := r.pool.QueryRow(ctx, query, accountID, convID, contactID, role)
	var p Participant
	if err := row.Scan(&p.ID, &p.AccountID, &p.ConversationID, &p.ContactID, &p.Role, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to create participant: %w", err)
	}
	return &p, nil
}

// List returns every participant of a conversation with the contact hydrated.
// Account-scoped via the join — a participant whose contact lives in another
// account is not visible.
func (r *ParticipantRepo) List(ctx context.Context, accountID, convID int64) ([]ParticipantWithContact, error) {
	query := `SELECT p.id, p.role, ` +
		"c.id, c.account_id, c.name, c.email, c.phone_number, c.phone_e164, c.identifier, c.additional_attributes, c.avatar_url, c.blocked, c.last_activity_at, c.created_at, c.updated_at " +
		`FROM participants p
		 INNER JOIN contacts c ON c.id = p.contact_id AND c.account_id = p.account_id
		 WHERE p.account_id = $1 AND p.conversation_id = $2
		 ORDER BY p.id ASC`
	rows, err := r.pool.Query(ctx, query, accountID, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	defer rows.Close()

	var out []ParticipantWithContact
	for rows.Next() {
		var pwc ParticipantWithContact
		if err := rows.Scan(
			&pwc.ID, &pwc.Role,
			&pwc.Contact.ID, &pwc.Contact.AccountID, &pwc.Contact.Name, &pwc.Contact.Email,
			&pwc.Contact.PhoneNumber, &pwc.Contact.PhoneE164, &pwc.Contact.Identifier,
			&pwc.Contact.AdditionalAttrs, &pwc.Contact.AvatarURL, &pwc.Contact.Blocked,
			&pwc.Contact.LastActivityAt, &pwc.Contact.CreatedAt, &pwc.Contact.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		out = append(out, pwc)
	}
	return out, rows.Err()
}

// Member is the input shape for SyncMembers — describes the desired state of
// a single participant.
type Member struct {
	ContactID int64
	Role      string
}

// SyncMembers reconciles the participants of a conversation against the
// supplied list within a single transaction: upserts every member by
// (conversation_id, contact_id), refreshing role; deletes participants whose
// contact_id is not in the list. Account-scoped on every statement.
func (r *ParticipantRepo) SyncMembers(ctx context.Context, accountID, convID int64, members []Member) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin sync members tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	desired := make(map[int64]string, len(members))
	for _, m := range members {
		role := m.Role
		if role == "" {
			role = "member"
		}
		desired[m.ContactID] = role
	}

	for cid, role := range desired {
		if _, err := tx.Exec(ctx,
			`INSERT INTO participants (account_id, conversation_id, contact_id, role)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (conversation_id, contact_id) DO UPDATE SET
				 role = EXCLUDED.role,
				 updated_at = NOW()`,
			accountID, convID, cid, role); err != nil {
			return fmt.Errorf("upsert participant: %w", err)
		}
	}

	if len(desired) == 0 {
		if _, err := tx.Exec(ctx,
			`DELETE FROM participants WHERE account_id = $1 AND conversation_id = $2`,
			accountID, convID); err != nil {
			return fmt.Errorf("delete absent participants: %w", err)
		}
	} else {
		ids := make([]int64, 0, len(desired))
		for id := range desired {
			ids = append(ids, id)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM participants
			 WHERE account_id = $1 AND conversation_id = $2 AND contact_id <> ALL($3::bigint[])`,
			accountID, convID, ids); err != nil {
			return fmt.Errorf("delete absent participants: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sync members: %w", err)
	}
	return nil
}

// Delete removes a participant by (conversation_id, contact_id). Returns
// ErrParticipantNotFound when no row matched.
func (r *ParticipantRepo) Delete(ctx context.Context, accountID, convID, contactID int64) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM participants WHERE account_id = $1 AND conversation_id = $2 AND contact_id = $3`,
		accountID, convID, contactID)
	if err != nil {
		return fmt.Errorf("failed to delete participant: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}
	return nil
}
