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

var ErrChannelTiktokNotFound = errors.New("channel tiktok not found")

const channelTiktokSelectColumns = "id, account_id, business_id, access_token_ciphertext, refresh_token_ciphertext, expires_at, refresh_token_expires_at, display_name, username, requires_reauth, created_at, updated_at"

type channelTiktokScanner interface {
	Scan(dest ...any) error
}

func scanChannelTiktok(scanner channelTiktokScanner, m *model.ChannelTiktok) error {
	return scanner.Scan(
		&m.ID, &m.AccountID, &m.BusinessID,
		&m.AccessTokenCiphertext, &m.RefreshTokenCiphertext,
		&m.ExpiresAt, &m.RefreshTokenExpiresAt,
		&m.DisplayName, &m.Username, &m.RequiresReauth,
		&m.CreatedAt, &m.UpdatedAt,
	)
}

type ChannelTiktokRepo struct {
	pool *pgxpool.Pool
}

func NewChannelTiktokRepo(pool *pgxpool.Pool) *ChannelTiktokRepo {
	return &ChannelTiktokRepo{pool: pool}
}

func (r *ChannelTiktokRepo) Create(ctx context.Context, m *model.ChannelTiktok) error {
	query := `INSERT INTO channels_tiktok (account_id, business_id, access_token_ciphertext, refresh_token_ciphertext, expires_at, refresh_token_expires_at, display_name, username)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.BusinessID, m.AccessTokenCiphertext, m.RefreshTokenCiphertext,
		m.ExpiresAt, m.RefreshTokenExpiresAt, m.DisplayName, m.Username,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel tiktok: %w", err)
	}
	return nil
}

func (r *ChannelTiktokRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelTiktok, error) {
	query := `SELECT ` + channelTiktokSelectColumns + ` FROM channels_tiktok WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelTiktok
	if err := scanChannelTiktok(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTiktokNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel tiktok by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTiktokRepo) FindByIDNoScope(ctx context.Context, id int64) (*model.ChannelTiktok, error) {
	query := `SELECT ` + channelTiktokSelectColumns + ` FROM channels_tiktok WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.ChannelTiktok
	if err := scanChannelTiktok(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTiktokNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel tiktok by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTiktokRepo) FindByBusinessID(ctx context.Context, businessID string) (*model.ChannelTiktok, error) {
	query := `SELECT ` + channelTiktokSelectColumns + ` FROM channels_tiktok WHERE business_id = $1`
	row := r.pool.QueryRow(ctx, query, businessID)
	var m model.ChannelTiktok
	if err := scanChannelTiktok(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTiktokNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel tiktok by business_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTiktokRepo) UpdateTokens(ctx context.Context, id int64, accessCiphertext, refreshCiphertext string, expiresAt, refreshExpiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_tiktok
			SET access_token_ciphertext = $1,
			    refresh_token_ciphertext = $2,
			    expires_at = $3,
			    refresh_token_expires_at = $4,
			    updated_at = NOW()
		 WHERE id = $5`,
		accessCiphertext, refreshCiphertext, expiresAt, refreshExpiresAt, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update tiktok tokens: %w", err)
	}
	return nil
}

func (r *ChannelTiktokRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_tiktok SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelTiktokRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_tiktok WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel tiktok: %w", err)
	}
	return nil
}
