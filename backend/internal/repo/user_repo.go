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

var ErrUserNotFound = errors.New("user not found")
var ErrUserEmailExists = errors.New("user email already exists")

const userSelectColumns = "id, email, name, password_hash, created_at, updated_at"

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner, m *model.User) error {
	return scanner.Scan(&m.ID, &m.Email, &m.Name, &m.PasswordHash, &m.CreatedAt, &m.UpdatedAt)
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

func isUniqueViolation(err error) bool {
	var pgErr interface{ Code() string }
	return errors.As(err, &pgErr) && pgErr.Code() == "23505"
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
