package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelTelegramNotFound = errors.New("channel telegram not found")

const channelTelegramSelectColumns = "id, account_id, bot_token_ciphertext, bot_name, webhook_identifier, secret_token_ciphertext, requires_reauth, created_at, updated_at"

type channelTelegramScanner interface {
	Scan(dest ...any) error
}

func scanChannelTelegram(scanner channelTelegramScanner, m *model.ChannelTelegram) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.BotTokenCiphertext, &m.BotName, &m.WebhookIdentifier, &m.SecretTokenCiphertext, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

type ChannelTelegramRepo struct {
	pool *pgxpool.Pool
}

func NewChannelTelegramRepo(pool *pgxpool.Pool) *ChannelTelegramRepo {
	return &ChannelTelegramRepo{pool: pool}
}

func (r *ChannelTelegramRepo) Create(ctx context.Context, m *model.ChannelTelegram) error {
	query := `INSERT INTO channels_telegram (account_id, bot_token_ciphertext, bot_name, webhook_identifier, secret_token_ciphertext)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.BotTokenCiphertext, m.BotName, m.WebhookIdentifier, m.SecretTokenCiphertext).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel telegram: %w", err)
	}
	return nil
}

func (r *ChannelTelegramRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelTelegram, error) {
	query := `SELECT ` + channelTelegramSelectColumns + ` FROM channels_telegram WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelTelegram
	if err := scanChannelTelegram(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTelegramNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel telegram by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelTelegramRepo) FindByWebhookIdentifier(ctx context.Context, identifier string) (*model.ChannelTelegram, error) {
	query := `SELECT ` + channelTelegramSelectColumns + ` FROM channels_telegram WHERE webhook_identifier = $1`
	row := r.pool.QueryRow(ctx, query, identifier)
	var m model.ChannelTelegram
	if err := scanChannelTelegram(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelTelegramNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel telegram by webhook identifier: %w", err)
	}
	return &m, nil
}

func (r *ChannelTelegramRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_telegram SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelTelegramRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_telegram WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel telegram: %w", err)
	}
	return nil
}
