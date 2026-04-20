package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelAPINotFound = errors.New("channel api not found")

const channelApiSelectColumns = "id, account_id, webhook_url, identifier, hmac_token, hmac_mandatory, secret, api_token_hash, created_at, updated_at"

type channelApiScanner interface {
	Scan(dest ...any) error
}

func scanChannelAPI(scanner channelApiScanner, m *model.ChannelAPI) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.WebhookURL, &m.Identifier, &m.HmacToken, &m.HmacMandatory, &m.Secret, &m.ApiTokenHash, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelAPIRepo struct {
	pool *pgxpool.Pool
}

func NewChannelAPIRepo(pool *pgxpool.Pool) *ChannelAPIRepo {
	return &ChannelAPIRepo{pool: pool}
}

func (r *ChannelAPIRepo) Create(ctx context.Context, m *model.ChannelAPI) error {
	query := `INSERT INTO channels_api (account_id, webhook_url, identifier, hmac_token, hmac_mandatory, secret, api_token_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.WebhookURL, m.Identifier, m.HmacToken, m.HmacMandatory, m.Secret, m.ApiTokenHash).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel api: %w", err)
	}
	return nil
}

func (r *ChannelAPIRepo) FindByID(ctx context.Context, id int64) (*model.ChannelAPI, error) {
	query := `SELECT ` + channelApiSelectColumns + ` FROM channels_api WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.ChannelAPI
	if err := scanChannelAPI(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelAPINotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel api by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelAPIRepo) FindByInboxID(ctx context.Context, inboxID int64) (*model.ChannelAPI, error) {
	query := `SELECT ca.id, ca.account_id, ca.webhook_url, ca.identifier, ca.hmac_token, ca.hmac_mandatory, ca.secret, ca.api_token_hash, ca.created_at, ca.updated_at FROM channels_api ca
		JOIN inboxes i ON i.channel_id = ca.id WHERE i.id = $1`
	row := r.pool.QueryRow(ctx, query, inboxID)
	var m model.ChannelAPI
	if err := scanChannelAPI(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelAPINotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel api by inbox id: %w", err)
	}
	return &m, nil
}

// FindByApiTokenHash looks up by deterministic SHA-256 hash of the provided
// plaintext api_access_token. The plaintext itself is never stored.
func (r *ChannelAPIRepo) FindByApiTokenHash(ctx context.Context, tokenHash string) (*model.ChannelAPI, error) {
	query := `SELECT ` + channelApiSelectColumns + ` FROM channels_api WHERE api_token_hash = $1`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var m model.ChannelAPI
	if err := scanChannelAPI(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelAPINotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel api by token hash: %w", err)
	}
	return &m, nil
}

func (r *ChannelAPIRepo) FindByIdentifier(ctx context.Context, identifier string) (*model.ChannelAPI, error) {
	query := `SELECT ` + channelApiSelectColumns + ` FROM channels_api WHERE identifier = $1`
	row := r.pool.QueryRow(ctx, query, identifier)
	var m model.ChannelAPI
	if err := scanChannelAPI(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelAPINotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel api by identifier: %w", err)
	}
	return &m, nil
}

func (r *ChannelAPIRepo) FindAccountByChannelID(ctx context.Context, channelID int64) (*model.Account, error) {
	query := `SELECT a.id, a.name, a.slug, a.created_at, a.updated_at
		FROM accounts a JOIN channels_api ca ON ca.account_id = a.id WHERE ca.id = $1`
	row := r.pool.QueryRow(ctx, query, channelID)
	var m model.Account
	if err := scanAccount(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrAccountNotFound, err)
		}
		return nil, fmt.Errorf("failed to find account by channel id: %w", err)
	}
	return &m, nil
}

func (r *ChannelAPIRepo) FindByInboxIDWithAccount(ctx context.Context, inboxID int64) (*model.ChannelAPI, *model.Account, error) {
	query := `SELECT ca.id, ca.account_id, ca.webhook_url, ca.identifier, ca.hmac_token, ca.hmac_mandatory, ca.secret, ca.api_token_hash, ca.created_at, ca.updated_at,
		a.id, a.name, a.slug, a.created_at, a.updated_at
		FROM channels_api ca
		JOIN inboxes i ON i.channel_id = ca.id
		JOIN accounts a ON a.id = ca.account_id
		WHERE i.id = $1`
	row := r.pool.QueryRow(ctx, query, inboxID)
	var ch model.ChannelAPI
	var ac model.Account
	if err := row.Scan(&ch.ID, &ch.AccountID, &ch.WebhookURL, &ch.Identifier, &ch.HmacToken, &ch.HmacMandatory, &ch.Secret, &ch.ApiTokenHash, &ch.CreatedAt, &ch.UpdatedAt,
		&ac.ID, &ac.Name, &ac.Slug, &ac.CreatedAt, &ac.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("%w: %w", ErrChannelAPINotFound, err)
		}
		return nil, nil, fmt.Errorf("failed to find channel with account: %w", err)
	}
	return &ch, &ac, nil
}
