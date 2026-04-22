package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInboxNotFound = errors.New("inbox not found")

const inboxSelectColumns = "id, account_id, channel_id, name, channel_type, created_at, updated_at"

type inboxScanner interface {
	Scan(dest ...any) error
}

func scanInbox(scanner inboxScanner, m *model.Inbox) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.ChannelID, &m.Name, &m.ChannelType, &m.CreatedAt, &m.UpdatedAt)
}

type InboxRepo struct {
	pool *pgxpool.Pool
}

func NewInboxRepo(pool *pgxpool.Pool) *InboxRepo {
	return &InboxRepo{pool: pool}
}

func (r *InboxRepo) Create(ctx context.Context, m *model.Inbox) error {
	query := `INSERT INTO inboxes (account_id, channel_id, name, channel_type) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.ChannelID, m.Name, m.ChannelType).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create inbox: %w", err)
	}
	return nil
}

// FindByID enforces tenant scoping at the repo layer. Callers must pass the
// account id from the authenticated request; cross-tenant ids resolve to
// ErrInboxNotFound (never leak existence to the caller).
func (r *InboxRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Inbox, error) {
	query := `SELECT ` + inboxSelectColumns + ` FROM inboxes WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Inbox
	if err := scanInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find inbox by id: %w", err)
	}
	return &m, nil
}

func (r *InboxRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.Inbox, error) {
	query := `SELECT ` + inboxSelectColumns + ` FROM inboxes WHERE account_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list inboxes: %w", err)
	}
	defer rows.Close()

	var inboxes []model.Inbox
	for rows.Next() {
		var m model.Inbox
		if err := scanInbox(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan inbox: %w", err)
		}
		inboxes = append(inboxes, m)
	}
	return inboxes, rows.Err()
}

// AccountIDByID returns only the account_id of an inbox. Used by callers
// (realtime membership check) that need to resolve ownership without fetching
// the full row and without the caller already knowing the account.
func (r *InboxRepo) AccountIDByID(ctx context.Context, id int64) (int64, error) {
	var accountID int64
	err := r.pool.QueryRow(ctx, `SELECT account_id FROM inboxes WHERE id = $1`, id).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: %w", ErrInboxNotFound, err)
		}
		return 0, fmt.Errorf("failed to resolve inbox account: %w", err)
	}
	return accountID, nil
}

func (r *InboxRepo) FindByChannelID(ctx context.Context, channelID int64) (*model.Inbox, error) {
	query := `SELECT ` + inboxSelectColumns + ` FROM inboxes WHERE channel_id = $1`
	row := r.pool.QueryRow(ctx, query, channelID)
	var m model.Inbox
	if err := scanInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find inbox by channel_id: %w", err)
	}
	return &m, nil
}

func (r *InboxRepo) FindByChannel(ctx context.Context, channelType string, channelID int64) (*model.Inbox, error) {
	query := `SELECT ` + inboxSelectColumns + ` FROM inboxes WHERE channel_type = $1 AND channel_id = $2`
	row := r.pool.QueryRow(ctx, query, channelType, channelID)
	var m model.Inbox
	if err := scanInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find inbox by channel: %w", err)
	}
	return &m, nil
}

func (r *InboxRepo) UpdateName(ctx context.Context, id, accountID int64, name string) error {
	query := `UPDATE inboxes SET name = $1, updated_at = now() WHERE id = $2 AND account_id = $3`
	tag, err := r.pool.Exec(ctx, query, name, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to update inbox name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrInboxNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *InboxRepo) FindByIdentifier(ctx context.Context, identifier string) (*model.Inbox, error) {
	query := `SELECT i.` + inboxSelectColumns + ` FROM inboxes i
		JOIN channels_api ca ON ca.id = i.channel_id WHERE ca.identifier = $1`
	row := r.pool.QueryRow(ctx, query, identifier)
	var m model.Inbox
	if err := scanInbox(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInboxNotFound, err)
		}
		return nil, fmt.Errorf("failed to find inbox by identifier: %w", err)
	}
	return &m, nil
}
