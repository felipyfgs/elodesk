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

var ErrChannelTwilioNotFound = errors.New("channel twilio not found")

const channelTwilioSelectColumns = "id, account_id, medium, account_sid, auth_token_ciphertext, api_key_sid, phone_number, messaging_service_sid, content_templates, content_templates_last_updated, webhook_identifier, requires_reauth, created_at, updated_at"

type channelTwilioScanner interface {
	Scan(dest ...any) error
}

func scanChannelTwilio(scanner channelTwilioScanner, m *model.ChannelTwilio) error {
	var medium string
	err := scanner.Scan(
		&m.ID, &m.AccountID, &medium, &m.AccountSID, &m.AuthTokenCiphertext,
		&m.APIKeySID, &m.PhoneNumber, &m.MessagingServiceSID,
		&m.ContentTemplates, &m.ContentTemplatesLastUpdated,
		&m.WebhookIdentifier, &m.RequiresReauth,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return err
	}
	m.Medium = model.TwilioMedium(medium)
	return nil
}

type ChannelTwilioRepo struct {
	pool *pgxpool.Pool
}

func NewChannelTwilioRepo(pool *pgxpool.Pool) *ChannelTwilioRepo {
	return &ChannelTwilioRepo{pool: pool}
}

func (r *ChannelTwilioRepo) Create(ctx context.Context, m *model.ChannelTwilio) error {
	templates := "[]"
	if m.ContentTemplates != nil && *m.ContentTemplates != "" {
		templates = *m.ContentTemplates
	}
	query := `INSERT INTO channels_twilio
		(account_id, medium, account_sid, auth_token_ciphertext, api_key_sid, phone_number, messaging_service_sid, content_templates, webhook_identifier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, string(m.Medium), m.AccountSID, m.AuthTokenCiphertext,
		m.APIKeySID, m.PhoneNumber, m.MessagingServiceSID,
		templates, m.WebhookIdentifier,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel twilio: %w", err)
	}
	if m.ContentTemplates == nil {
		m.ContentTemplates = &templates
	}
	return nil
}

func (r *ChannelTwilioRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelTwilio, error) {
	query := `SELECT ` + channelTwilioSelectColumns + ` FROM channels_twilio WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelTwilio
	if err := scanChannelTwilio(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwilioNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twilio by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwilioRepo) FindByIDNoScope(ctx context.Context, id int64) (*model.ChannelTwilio, error) {
	query := `SELECT ` + channelTwilioSelectColumns + ` FROM channels_twilio WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.ChannelTwilio
	if err := scanChannelTwilio(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwilioNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twilio by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwilioRepo) FindByWebhookIdentifier(ctx context.Context, identifier string) (*model.ChannelTwilio, error) {
	query := `SELECT ` + channelTwilioSelectColumns + ` FROM channels_twilio WHERE webhook_identifier = $1`
	row := r.pool.QueryRow(ctx, query, identifier)
	var m model.ChannelTwilio
	if err := scanChannelTwilio(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTwilioNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel twilio by webhook_identifier: %w", err)
	}
	return &m, nil
}

func (r *ChannelTwilioRepo) UpdateAuthToken(ctx context.Context, id int64, authTokenCiphertext string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_twilio SET auth_token_ciphertext = $1, updated_at = NOW() WHERE id = $2`,
		authTokenCiphertext, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update twilio auth token: %w", err)
	}
	return nil
}

func (r *ChannelTwilioRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_twilio SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelTwilioRepo) UpdateContentTemplates(ctx context.Context, id int64, templatesJSON string, syncedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_twilio SET content_templates = $1, content_templates_last_updated = $2, updated_at = NOW() WHERE id = $3`,
		templatesJSON, syncedAt, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update content templates: %w", err)
	}
	return nil
}

func (r *ChannelTwilioRepo) ListStaleTemplates(ctx context.Context, olderThan time.Time) ([]*model.ChannelTwilio, error) {
	query := `SELECT ` + channelTwilioSelectColumns + ` FROM channels_twilio
		WHERE medium = 'whatsapp'
		AND (content_templates_last_updated IS NULL OR content_templates_last_updated < $1)`
	rows, err := r.pool.Query(ctx, query, olderThan)
	if err != nil {
		return nil, fmt.Errorf("list stale twilio templates: %w", err)
	}
	defer rows.Close()

	var result []*model.ChannelTwilio
	for rows.Next() {
		m := &model.ChannelTwilio{}
		if err := scanChannelTwilio(rows, m); err != nil {
			return nil, fmt.Errorf("scan twilio row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *ChannelTwilioRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_twilio WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel twilio: %w", err)
	}
	return nil
}
