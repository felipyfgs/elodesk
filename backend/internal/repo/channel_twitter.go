package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelTwitterNotFound = errors.New("channel twitter not found")

const channelTwitterSelectColumns = "id, account_id, profile_id, screen_name, twitter_access_token_ciphertext, twitter_access_token_secret_ciphertext, tweets_enabled, requires_reauth, created_at, updated_at"

type channelTwitterScanner interface {
	Scan(dest ...any) error
}

func scanChannelTwitter(scanner channelTwitterScanner, m *model.ChannelTwitter) error {
	return scanner.Scan(
		&m.ID, &m.AccountID, &m.ProfileID, &m.ScreenName,
		&m.TwitterAccessTokenCiphertext, &m.TwitterAccessTokenSecretCiphertext,
		&m.TweetsEnabled, &m.RequiresReauth,
		&m.CreatedAt, &m.UpdatedAt,
	)
}

type ChannelTwitterRepo struct {
	pool *pgxpool.Pool
}

func NewChannelTwitterRepo(pool *pgxpool.Pool) *ChannelTwitterRepo {
	return &ChannelTwitterRepo{pool: pool}
}

func (r *ChannelTwitterRepo) Create(ctx context.Context, m *model.ChannelTwitter) error {
	query := `INSERT INTO channels_twitter (account_id, profile_id, screen_name, twitter_access_token_ciphertext, twitter_access_token_secret_ciphertext, tweets_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, requires_reauth, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.ProfileID, m.ScreenName,
		m.TwitterAccessTokenCiphertext, m.TwitterAccessTokenSecretCiphertext,
		m.TweetsEnabled,
	).Scan(&m.ID, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel twitter: %w", err)
	}
	return nil
}

func (r *ChannelTwitterRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelTwitter, error) {
	query := `SELECT ` + channelTwitterSelectColumns + ` FROM channels_twitter WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelTwitter
	if err := scanChannelTwitter(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwitterNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twitter by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwitterRepo) FindByIDNoScope(ctx context.Context, id int64) (*model.ChannelTwitter, error) {
	query := `SELECT ` + channelTwitterSelectColumns + ` FROM channels_twitter WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.ChannelTwitter
	if err := scanChannelTwitter(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwitterNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twitter by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwitterRepo) FindByProfileID(ctx context.Context, profileID string) (*model.ChannelTwitter, error) {
	query := `SELECT ` + channelTwitterSelectColumns + ` FROM channels_twitter WHERE profile_id = $1`
	row := r.pool.QueryRow(ctx, query, profileID)
	var m model.ChannelTwitter
	if err := scanChannelTwitter(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwitterNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twitter by profile_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwitterRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_twitter SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelTwitterRepo) SetTweetsEnabled(ctx context.Context, id int64, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_twitter SET tweets_enabled = $1, updated_at = NOW() WHERE id = $2`,
		enabled, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set tweets_enabled: %w", err)
	}
	return nil
}

func (r *ChannelTwitterRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_twitter WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel twitter: %w", err)
	}
	return nil
}
