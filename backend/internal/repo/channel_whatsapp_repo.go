package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelWhatsappNotFound = errors.New("channel whatsapp not found")

const channelWhatsappSelectColumns = "id, account_id, provider, phone_number, phone_number_id, business_account_id, api_key_ciphertext, webhook_verify_token_ciphertext, message_templates, message_templates_synced_at, requires_reauth, created_at, updated_at"

type channelWhatsappScanner interface {
	Scan(dest ...any) error
}

func scanChannelWhatsapp(scanner channelWhatsappScanner, m *model.ChannelWhatsapp) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Provider, &m.PhoneNumber, &m.PhoneNumberID, &m.BusinessAccountID, &m.ApiKeyCiphertext, &m.WebhookVerifyTokenCiphertext, &m.MessageTemplates, &m.MessageTemplatesSyncedAt, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelWhatsappRepo struct {
	pool *pgxpool.Pool
}

func NewChannelWhatsappRepo(pool *pgxpool.Pool) *ChannelWhatsappRepo {
	return &ChannelWhatsappRepo{pool: pool}
}

func (r *ChannelWhatsappRepo) Create(ctx context.Context, m *model.ChannelWhatsapp) error {
	query := `INSERT INTO channels_whatsapp (account_id, provider, phone_number, phone_number_id, business_account_id, api_key_ciphertext, webhook_verify_token_ciphertext, message_templates)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.Provider, m.PhoneNumber, m.PhoneNumberID, m.BusinessAccountID,
		m.ApiKeyCiphertext, m.WebhookVerifyTokenCiphertext, m.MessageTemplates,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel whatsapp: %w", err)
	}
	return nil
}

func (r *ChannelWhatsappRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelWhatsapp, error) {
	query := `SELECT ` + channelWhatsappSelectColumns + ` FROM channels_whatsapp WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelWhatsapp
	if err := scanChannelWhatsapp(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWhatsappNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel whatsapp by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelWhatsappRepo) FindByPhoneNumber(ctx context.Context, accountID int64, phone string) (*model.ChannelWhatsapp, error) {
	query := `SELECT ` + channelWhatsappSelectColumns + ` FROM channels_whatsapp WHERE account_id = $1 AND phone_number = $2`
	row := r.pool.QueryRow(ctx, query, accountID, phone)
	var m model.ChannelWhatsapp
	if err := scanChannelWhatsapp(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWhatsappNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel whatsapp by phone: %w", err)
	}
	return &m, nil
}

func (r *ChannelWhatsappRepo) FindByPhoneNumberID(ctx context.Context, phoneNumberID string) (*model.ChannelWhatsapp, error) {
	query := `SELECT ` + channelWhatsappSelectColumns + ` FROM channels_whatsapp WHERE phone_number_id = $1`
	row := r.pool.QueryRow(ctx, query, phoneNumberID)
	var m model.ChannelWhatsapp
	if err := scanChannelWhatsapp(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelWhatsappNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel whatsapp by phone_number_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelWhatsappRepo) UpdateTemplates(ctx context.Context, id int64, templates *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_whatsapp SET message_templates = $1, message_templates_synced_at = NOW(), updated_at = NOW() WHERE id = $2`,
		templates, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update whatsapp templates: %w", err)
	}
	return nil
}

func (r *ChannelWhatsappRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_whatsapp SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}
