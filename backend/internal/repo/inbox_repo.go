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

func (r *InboxRepo) FindByID(ctx context.Context, id int64) (*model.Inbox, error) {
	query := `SELECT ` + inboxSelectColumns + ` FROM inboxes WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
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
