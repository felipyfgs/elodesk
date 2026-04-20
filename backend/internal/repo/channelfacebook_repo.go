package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelFacebookNotFound = errors.New("channel facebook not found")

const channelFacebookSelectColumns = "id, account_id, page_id, page_access_token_ciphertext, user_access_token_ciphertext, instagram_id, requires_reauth, created_at, updated_at"

type channelFacebookScanner interface {
	Scan(dest ...any) error
}

func scanChannelFacebook(scanner channelFacebookScanner, m *model.ChannelFacebookPage) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.PageID, &m.PageAccessTokenCiphertext, &m.UserAccessTokenCiphertext, &m.InstagramID, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelFacebookRepo struct {
	pool *pgxpool.Pool
}

func NewChannelFacebookRepo(pool *pgxpool.Pool) *ChannelFacebookRepo {
	return &ChannelFacebookRepo{pool: pool}
}

func (r *ChannelFacebookRepo) Create(ctx context.Context, m *model.ChannelFacebookPage) error {
	query := `INSERT INTO channels_facebook_page (account_id, page_id, page_access_token_ciphertext, user_access_token_ciphertext, instagram_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.PageID, m.PageAccessTokenCiphertext, m.UserAccessTokenCiphertext, m.InstagramID).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel facebook page: %w", err)
	}
	return nil
}

func (r *ChannelFacebookRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelFacebookPage, error) {
	query := `SELECT ` + channelFacebookSelectColumns + ` FROM channels_facebook_page WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelFacebookPage
	if err := scanChannelFacebook(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelFacebookNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel facebook by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelFacebookRepo) FindByPageID(ctx context.Context, pageID string) (*model.ChannelFacebookPage, error) {
	query := `SELECT ` + channelFacebookSelectColumns + ` FROM channels_facebook_page WHERE page_id = $1`
	row := r.pool.QueryRow(ctx, query, pageID)
	var m model.ChannelFacebookPage
	if err := scanChannelFacebook(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelFacebookNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel facebook by page_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelFacebookRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_facebook_page SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}
