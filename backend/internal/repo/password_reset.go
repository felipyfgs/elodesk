package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrResetTokenNotFound = errors.New("reset token not found")

type PasswordResetToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type PasswordResetTokenRepo struct {
	pool *pgxpool.Pool
}

func NewPasswordResetTokenRepo(pool *pgxpool.Pool) *PasswordResetTokenRepo {
	return &PasswordResetTokenRepo{pool: pool}
}

func (r *PasswordResetTokenRepo) Create(ctx context.Context, t *PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, t.UserID, t.TokenHash, t.ExpiresAt).
		Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepo) FindByHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	query := `SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
		FROM password_reset_tokens WHERE token_hash = $1`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var t PasswordResetToken
	if err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.ConsumedAt, &t.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrResetTokenNotFound, err)
		}
		return nil, fmt.Errorf("failed to find password reset token: %w", err)
	}
	return &t, nil
}

func (r *PasswordResetTokenRepo) Consume(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET consumed_at = $1 WHERE id = $2 AND consumed_at IS NULL AND expires_at > $1`,
		now, id)
	if err != nil {
		return fmt.Errorf("failed to consume password reset token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResetTokenNotFound
	}
	return nil
}
