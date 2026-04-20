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

var ErrChannelInstagramNotFound = errors.New("channel instagram not found")

const channelInstagramSelectColumns = "id, account_id, instagram_id, access_token_ciphertext, expires_at, requires_reauth, created_at, updated_at"

type channelInstagramScanner interface {
	Scan(dest ...any) error
}

func scanChannelInstagram(scanner channelInstagramScanner, m *model.ChannelInstagram) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InstagramID, &m.AccessTokenCiphertext, &m.ExpiresAt, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelInstagramRepo struct {
	pool *pgxpool.Pool
}

func NewChannelInstagramRepo(pool *pgxpool.Pool) *ChannelInstagramRepo {
	return &ChannelInstagramRepo{pool: pool}
}

func (r *ChannelInstagramRepo) Create(ctx context.Context, m *model.ChannelInstagram) error {
	query := `INSERT INTO channels_instagram (account_id, instagram_id, access_token_ciphertext, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.InstagramID, m.AccessTokenCiphertext, m.ExpiresAt).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel instagram: %w", err)
	}
	return nil
}

func (r *ChannelInstagramRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelInstagram, error) {
	query := `SELECT ` + channelInstagramSelectColumns + ` FROM channels_instagram WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelInstagram
	if err := scanChannelInstagram(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelInstagramNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel instagram by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelInstagramRepo) FindByInstagramID(ctx context.Context, instagramID string) (*model.ChannelInstagram, error) {
	query := `SELECT ` + channelInstagramSelectColumns + ` FROM channels_instagram WHERE instagram_id = $1`
	row := r.pool.QueryRow(ctx, query, instagramID)
	var m model.ChannelInstagram
	if err := scanChannelInstagram(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelInstagramNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel instagram by instagram_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelInstagramRepo) UpdateToken(ctx context.Context, id int64, ciphertext string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_instagram SET access_token_ciphertext = $1, expires_at = $2, updated_at = NOW() WHERE id = $3`,
		ciphertext, expiresAt, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update instagram token: %w", err)
	}
	return nil
}

func (r *ChannelInstagramRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_instagram SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}
