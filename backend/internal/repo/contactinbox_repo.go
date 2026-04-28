package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrContactInboxNotFound = errors.New("contact inbox not found")

const contactInboxSelectColumns = "id, contact_id, inbox_id, source_id, hmac_verified, created_at, updated_at"

type contactInboxScanner interface {
	Scan(dest ...any) error
}

func scanContactInbox(scanner contactInboxScanner, m *model.ContactInbox) error {
	return scanner.Scan(&m.ID, &m.ContactID, &m.InboxID, &m.SourceID, &m.HmacVerified, &m.CreatedAt, &m.UpdatedAt)
}

type ContactInboxRepo struct {
	pool *pgxpool.Pool
}

func NewContactInboxRepo(pool *pgxpool.Pool) *ContactInboxRepo {
	return &ContactInboxRepo{pool: pool}
}

func (r *ContactInboxRepo) Create(ctx context.Context, m *model.ContactInbox) error {
	query := `INSERT INTO contact_inboxes (contact_id, inbox_id, source_id, hmac_verified)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.ContactID, m.InboxID, m.SourceID, m.HmacVerified).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create contact inbox: %w", err)
	}
	return nil
}

// FindByID resolves a contact_inbox by id, scoped to the given accountID via
// the inbox JOIN. The contact_inboxes table has no account_id of its own, so
// the join through inboxes is the multi-tenant guard — without it any caller
// holding a numeric id could fetch another tenant's contact_inbox.
func (r *ContactInboxRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ContactInbox, error) {
	query := `SELECT ci.id, ci.contact_id, ci.inbox_id, ci.source_id, ci.hmac_verified, ci.created_at, ci.updated_at
		FROM contact_inboxes ci
		INNER JOIN inboxes i ON i.id = ci.inbox_id
		WHERE ci.id = $1 AND i.account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ContactInbox
	if err := scanContactInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find contact inbox by id: %w", err)
	}
	return &m, nil
}

func (r *ContactInboxRepo) FindBySourceID(ctx context.Context, sourceID string, inboxID int64) (*model.ContactInbox, error) {
	query := `SELECT ` + contactInboxSelectColumns + ` FROM contact_inboxes WHERE source_id = $1 AND inbox_id = $2`
	row := r.pool.QueryRow(ctx, query, sourceID, inboxID)
	var m model.ContactInbox
	if err := scanContactInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find contact inbox by source_id: %w", err)
	}
	return &m, nil
}

func (r *ContactInboxRepo) FindByContactAndInbox(ctx context.Context, contactID, inboxID int64) (*model.ContactInbox, error) {
	query := `SELECT ` + contactInboxSelectColumns + ` FROM contact_inboxes WHERE contact_id = $1 AND inbox_id = $2`
	row := r.pool.QueryRow(ctx, query, contactID, inboxID)
	var m model.ContactInbox
	if err := scanContactInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find contact inbox: %w", err)
	}
	return &m, nil
}

func (r *ContactInboxRepo) UpdateHmacVerified(ctx context.Context, id int64, verified bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contact_inboxes SET hmac_verified = $1, updated_at = NOW() WHERE id = $2`,
		verified, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update hmac_verified: %w", err)
	}
	return nil
}
