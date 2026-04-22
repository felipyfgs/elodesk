package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelLineNotFound = errors.New("channel line not found")

const channelLineSelectColumns = "id, account_id, line_channel_id, line_channel_secret_ciphertext, line_channel_token_ciphertext, bot_basic_id, bot_display_name, requires_reauth, created_at, updated_at"

type channelLineScanner interface {
	Scan(dest ...any) error
}

func scanChannelLine(scanner channelLineScanner, m *model.ChannelLine) error {
	return scanner.Scan(
		&m.ID, &m.AccountID, &m.LineChannelID,
		&m.LineChannelSecretCiphertext, &m.LineChannelTokenCiphertext,
		&m.BotBasicID, &m.BotDisplayName, &m.RequiresReauth,
		&m.CreatedAt, &m.UpdatedAt,
	)
}

type ChannelLineRepo struct {
	pool *pgxpool.Pool
}

func NewChannelLineRepo(pool *pgxpool.Pool) *ChannelLineRepo {
	return &ChannelLineRepo{pool: pool}
}

func (r *ChannelLineRepo) Create(ctx context.Context, m *model.ChannelLine) error {
	query := `INSERT INTO channels_line (account_id, line_channel_id, line_channel_secret_ciphertext, line_channel_token_ciphertext, bot_basic_id, bot_display_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		m.AccountID, m.LineChannelID, m.LineChannelSecretCiphertext,
		m.LineChannelTokenCiphertext, m.BotBasicID, m.BotDisplayName,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel line: %w", err)
	}
	return nil
}

func (r *ChannelLineRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelLine, error) {
	query := `SELECT ` + channelLineSelectColumns + ` FROM channels_line WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelLine
	if err := scanChannelLine(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelLineNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel line by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelLineRepo) FindByIDNoScope(ctx context.Context, id int64) (*model.ChannelLine, error) {
	query := `SELECT ` + channelLineSelectColumns + ` FROM channels_line WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.ChannelLine
	if err := scanChannelLine(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelLineNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel line by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelLineRepo) FindByLineChannelID(ctx context.Context, lineChannelID string) (*model.ChannelLine, error) {
	query := `SELECT ` + channelLineSelectColumns + ` FROM channels_line WHERE line_channel_id = $1`
	row := r.pool.QueryRow(ctx, query, lineChannelID)
	var m model.ChannelLine
	if err := scanChannelLine(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelLineNotFound, err)
		}
		return nil, fmt.Errorf("failed to find channel line by line_channel_id: %w", err)
	}
	return &m, nil
}

func (r *ChannelLineRepo) UpdateCredentials(ctx context.Context, id int64, secretCiphertext, tokenCiphertext string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_line SET line_channel_secret_ciphertext = $1, line_channel_token_ciphertext = $2, updated_at = NOW() WHERE id = $3`,
		secretCiphertext, tokenCiphertext, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update line credentials: %w", err)
	}
	return nil
}

func (r *ChannelLineRepo) SetRequiresReauth(ctx context.Context, id int64, requires bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_line SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`,
		requires, id,
	)
	if err != nil {
		return fmt.Errorf("failed to set requires_reauth: %w", err)
	}
	return nil
}

func (r *ChannelLineRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels_line WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel line: %w", err)
	}
	return nil
}
