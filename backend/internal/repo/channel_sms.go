package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelSMSNotFound = errors.New("channel sms not found")

const channelSMSSelectColumns = "id, account_id, inbox_id, provider, phone_number, webhook_identifier, provider_config_ciphertext, messaging_service_sid, requires_reauth, created_at, updated_at"

type channelSMSScanner interface {
	Scan(dest ...any) error
}

func scanChannelSMS(scanner channelSMSScanner, m *model.ChannelSMS) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.Provider, &m.PhoneNumber, &m.WebhookIdentifier, &m.ProviderConfigCiphertext, &m.MessagingServiceSid, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelSMSRepo struct {
	pool *pgxpool.Pool
}

func NewChannelSMSRepo(pool *pgxpool.Pool) *ChannelSMSRepo {
	return &ChannelSMSRepo{pool: pool}
}

func (r *ChannelSMSRepo) Create(ctx context.Context, m *model.ChannelSMS) error {
	query := `INSERT INTO channels_sms (account_id, inbox_id, provider, phone_number, webhook_identifier, provider_config_ciphertext, messaging_service_sid)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.Provider, m.PhoneNumber, m.WebhookIdentifier, m.ProviderConfigCiphertext, m.MessagingServiceSid).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel sms: %w", err)
	}
	return nil
}

func (r *ChannelSMSRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelSMS, error) {
	query := `SELECT ` + channelSMSSelectColumns + ` FROM channels_sms WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelSMS
	if err := scanChannelSMS(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelSMSNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel sms by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelSMSRepo) FindByWebhookIdentifier(ctx context.Context, identifier string) (*model.ChannelSMS, error) {
	query := `SELECT ` + channelSMSSelectColumns + ` FROM channels_sms WHERE webhook_identifier = $1`
	row := r.pool.QueryRow(ctx, query, identifier)
	var m model.ChannelSMS
	if err := scanChannelSMS(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelSMSNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel sms by webhook_identifier: %w", err)
	}
	return &m, nil
}

func (r *ChannelSMSRepo) UpdateInboxID(ctx context.Context, id int64, inboxID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_sms SET inbox_id = $1, updated_at = NOW() WHERE id = $2`,
		inboxID, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update channel sms inbox_id: %w", err)
	}
	return nil
}

func (r *ChannelSMSRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_sms SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelSMSRepo) FindByAccountID(ctx context.Context, accountID int64) ([]*model.ChannelSMS, error) {
	query := `SELECT ` + channelSMSSelectColumns + ` FROM channels_sms WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list channel sms by account_id: %w", err)
	}
	defer rows.Close()

	var channels []*model.ChannelSMS
	for rows.Next() {
		var m model.ChannelSMS
		if err := scanChannelSMS(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan channel sms: %w", err)
		}
		channels = append(channels, &m)
	}
	return channels, rows.Err()
}
