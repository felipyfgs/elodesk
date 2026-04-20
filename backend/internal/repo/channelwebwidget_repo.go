package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelWebWidgetNotFound = errors.New("channel web widget not found")

const channelWebWidgetSelectColumns = "id, account_id, inbox_id, website_token, hmac_token_ciphertext, website_url, widget_color, welcome_title, welcome_tagline, reply_time, feature_flags, requires_reauth, created_at, updated_at"

type channelWebWidgetScanner interface {
	Scan(dest ...any) error
}

func scanChannelWebWidget(scanner channelWebWidgetScanner, m *model.ChannelWebWidget) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.WebsiteToken, &m.HmacTokenCiphertext, &m.WebsiteURL, &m.WidgetColor, &m.WelcomeTitle, &m.WelcomeTagline, &m.ReplyTime, &m.FeatureFlags, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelWebWidgetRepo struct {
	pool *pgxpool.Pool
}

func NewChannelWebWidgetRepo(pool *pgxpool.Pool) *ChannelWebWidgetRepo {
	return &ChannelWebWidgetRepo{pool: pool}
}

func (r *ChannelWebWidgetRepo) Create(ctx context.Context, m *model.ChannelWebWidget) error {
	query := `INSERT INTO channels_web_widget (account_id, inbox_id, website_token, hmac_token_ciphertext, website_url, widget_color, welcome_title, welcome_tagline, reply_time, feature_flags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.InboxID, m.WebsiteToken, m.HmacTokenCiphertext,
		m.WebsiteURL, m.WidgetColor, m.WelcomeTitle, m.WelcomeTagline,
		m.ReplyTime, m.FeatureFlags,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel web widget: %w", err)
	}
	return nil
}

func (r *ChannelWebWidgetRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelWebWidget, error) {
	query := `SELECT ` + channelWebWidgetSelectColumns + ` FROM channels_web_widget WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelWebWidget
	if err := scanChannelWebWidget(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWebWidgetNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel web widget by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelWebWidgetRepo) FindByWebsiteToken(ctx context.Context, websiteToken string) (*model.ChannelWebWidget, error) {
	query := `SELECT ` + channelWebWidgetSelectColumns + ` FROM channels_web_widget WHERE website_token = $1`
	row := r.pool.QueryRow(ctx, query, websiteToken)
	var m model.ChannelWebWidget
	if err := scanChannelWebWidget(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWebWidgetNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel web widget by website_token: %w", err)
	}
	return &m, nil
}

func (r *ChannelWebWidgetRepo) FindByInboxID(ctx context.Context, inboxID int64) (*model.ChannelWebWidget, error) {
	query := `SELECT ` + channelWebWidgetSelectColumns + ` FROM channels_web_widget WHERE inbox_id = $1`
	row := r.pool.QueryRow(ctx, query, inboxID)
	var m model.ChannelWebWidget
	if err := scanChannelWebWidget(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWebWidgetNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel web widget by inbox_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelWebWidgetRepo) UpdateHmacToken(ctx context.Context, id int64, ciphertext string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_web_widget SET hmac_token_ciphertext = $1, updated_at = NOW() WHERE id = $2`,
		ciphertext, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update hmac_token: %w", err)
	}
	return nil
}

func (r *ChannelWebWidgetRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_web_widget WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel web widget: %w", err)
	}
	return nil
}

func (r *ChannelWebWidgetRepo) UpdateInboxID(ctx context.Context, id int64, inboxID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_web_widget SET inbox_id = $1, updated_at = NOW() WHERE id = $2`,
		inboxID, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update inbox_id: %w", err)
	}
	return nil
}
