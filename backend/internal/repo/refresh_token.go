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

var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrRefreshTokenReused = errors.New("refresh token reuse detected")

const refreshTokenSelectColumns = "id, user_id, token_hash, family_id, revoked_at, expires_at, created_at"

type refreshTokenScanner interface {
	Scan(dest ...any) error
}

func scanRefreshToken(scanner refreshTokenScanner, m *model.RefreshToken) error {
	return scanner.Scan(&m.ID, &m.UserID, &m.TokenHash, &m.FamilyID, &m.RevokedAt, &m.ExpiresAt, &m.CreatedAt)
}

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, m *model.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, m.UserID, m.TokenHash, m.FamilyID, m.ExpiresAt).
		Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `SELECT ` + refreshTokenSelectColumns + ` FROM refresh_tokens WHERE token_hash = $1`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var m model.RefreshToken
	if err := scanRefreshToken(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrRefreshTokenNotFound, err)
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	return &m, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: token %d", ErrRefreshTokenNotFound, id)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeByFamily(ctx context.Context, userID int64, familyID string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND family_id = $3 AND revoked_at IS NULL`,
		now, userID, familyID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token family: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`,
		now, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
	}
	return nil
}
