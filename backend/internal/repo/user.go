package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserEmailExists = errors.New("user email already exists")

const userSelectColumns = "id, email, name, password_hash, avatar_url, mfa_enabled, mfa_secret_ciphertext, created_at, updated_at"

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner, m *model.User) error {
	return scanner.Scan(&m.ID, &m.Email, &m.Name, &m.PasswordHash, &m.AvatarURL, &m.MfaEnabled, &m.MfaSecretCiphertext, &m.CreatedAt, &m.UpdatedAt)
}

type AuthUser struct {
	ID    int64
	Email string
	Name  string
}

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Pool() *pgxpool.Pool { return r.pool }

func (r *UserRepo) CreateTx(ctx context.Context, tx pgx.Tx, m *model.User) error {
	query := `INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	err := tx.QueryRow(ctx, query, m.Email, m.Name, m.PasswordHash).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrUserEmailExists, err)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *UserRepo) Create(ctx context.Context, m *model.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := r.CreateTx(ctx, tx, m); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.User
	if err := scanUser(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrUserNotFound, err)
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return &m, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)
	var m model.User
	if err := scanUser(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrUserNotFound, err)
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &m, nil
}

func (r *UserRepo) UpdateMfaSecret(ctx context.Context, userID int64, secretCiphertext string, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET mfa_secret_ciphertext = $1, mfa_enabled = $2, updated_at = NOW() WHERE id = $3`,
		secretCiphertext, enabled, userID)
	if err != nil {
		return fmt.Errorf("failed to update user mfa secret: %w", err)
	}
	return nil
}

func (r *UserRepo) EnableMfa(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET mfa_enabled = TRUE, updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to enable mfa: %w", err)
	}
	return nil
}

func (r *UserRepo) DisableMfa(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET mfa_enabled = FALSE, mfa_secret_ciphertext = NULL, updated_at = NOW() WHERE id = $1`,
		userID)
	if err != nil {
		return fmt.Errorf("failed to disable mfa: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateName(ctx context.Context, userID int64, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET name = $1, updated_at = NOW() WHERE id = $2`, name, userID)
	if err != nil {
		return fmt.Errorf("failed to update user name: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateEmail(ctx context.Context, userID int64, email string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`, email, userID)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrUserEmailExists, err)
		}
		return fmt.Errorf("failed to update user email: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL *string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2`, avatarURL, userID)
	if err != nil {
		return fmt.Errorf("failed to update user avatar: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdatePasswordHash(ctx context.Context, userID int64, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		hash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}
	return nil
}

func (r *UserRepo) HasUsers(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users LIMIT 1)`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if users exist: %w", err)
	}
	return exists, nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
