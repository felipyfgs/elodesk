package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRecoveryCodeNotFound = errors.New("recovery code not found")

type MfaRecoveryCode struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	CodeHash   string     `json:"-"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type MfaRecoveryCodeRepo struct {
	pool *pgxpool.Pool
}

func NewMfaRecoveryCodeRepo(pool *pgxpool.Pool) *MfaRecoveryCodeRepo {
	return &MfaRecoveryCodeRepo{pool: pool}
}

func (r *MfaRecoveryCodeRepo) Create(ctx context.Context, userID int64, codeHashes []string) error {
	if len(codeHashes) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, h := range codeHashes {
		batch.Queue(`INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)`, userID, h)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer func() {
		if cerr := br.Close(); cerr != nil {
			_ = cerr
		}
	}()

	for range codeHashes {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to create mfa recovery code: %w", err)
		}
	}
	return nil
}

func (r *MfaRecoveryCodeRepo) FindByHash(ctx context.Context, codeHash string) (*MfaRecoveryCode, error) {
	query := `SELECT id, user_id, code_hash, consumed_at, created_at
		FROM mfa_recovery_codes WHERE code_hash = $1 AND consumed_at IS NULL`
	row := r.pool.QueryRow(ctx, query, codeHash)
	var c MfaRecoveryCode
	if err := row.Scan(&c.ID, &c.UserID, &c.CodeHash, &c.ConsumedAt, &c.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrRecoveryCodeNotFound, err)
		}
		return nil, fmt.Errorf("failed to find mfa recovery code: %w", err)
	}
	return &c, nil
}

func (r *MfaRecoveryCodeRepo) Consume(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx,
		`UPDATE mfa_recovery_codes SET consumed_at = $1 WHERE id = $2 AND consumed_at IS NULL`,
		now, id)
	if err != nil {
		return fmt.Errorf("failed to consume mfa recovery code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRecoveryCodeNotFound
	}
	return nil
}

func (r *MfaRecoveryCodeRepo) DeleteAllByUserID(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete mfa recovery codes: %w", err)
	}
	return nil
}
