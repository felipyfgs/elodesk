package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountSlugExists = errors.New("account slug already exists")

const accountSelectColumns = "id, name, slug, created_at, updated_at"

type accountScanner interface {
	Scan(dest ...any) error
}

func scanAccount(scanner accountScanner, m *model.Account) error {
	return scanner.Scan(&m.ID, &m.Name, &m.Slug, &m.CreatedAt, &m.UpdatedAt)
}

type AccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

func (r *AccountRepo) CreateTx(ctx context.Context, tx pgx.Tx, m *model.Account) error {
	query := `INSERT INTO accounts (name, slug) VALUES ($1, $2)
		RETURNING id, created_at, updated_at`
	err := tx.QueryRow(ctx, query, m.Name, m.Slug).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrAccountSlugExists, err)
		}
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (r *AccountRepo) Create(ctx context.Context, m *model.Account) error {
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

func (r *AccountRepo) FindByID(ctx context.Context, id int64) (*model.Account, error) {
	query := `SELECT ` + accountSelectColumns + ` FROM accounts WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var m model.Account
	if err := scanAccount(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrAccountNotFound, err)
		}
		return nil, fmt.Errorf("failed to find account by id: %w", err)
	}
	return &m, nil
}

func (r *AccountRepo) FindBySlug(ctx context.Context, slug string) (*model.Account, error) {
	query := `SELECT ` + accountSelectColumns + ` FROM accounts WHERE slug = $1`
	row := r.pool.QueryRow(ctx, query, slug)
	var m model.Account
	if err := scanAccount(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrAccountNotFound, err)
		}
		return nil, fmt.Errorf("failed to find account by slug: %w", err)
	}
	return &m, nil
}

const accountUserSelectColumns = "id, account_id, user_id, role, created_at, updated_at"

type accountUserScanner interface {
	Scan(dest ...any) error
}

func scanAccountUser(scanner accountUserScanner, m *model.AccountUser) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt)
}

func (r *AccountRepo) FindAccountUser(ctx context.Context, accountID, userID int64) (*model.AccountUser, error) {
	query := `SELECT ` + accountUserSelectColumns + ` FROM account_users WHERE account_id = $1 AND user_id = $2`
	row := r.pool.QueryRow(ctx, query, accountID, userID)
	var au model.AccountUser
	if err := scanAccountUser(row, &au); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrAccountNotFound, err)
		}
		return nil, fmt.Errorf("failed to find account user: %w", err)
	}
	return &au, nil
}

func (r *AccountRepo) AddUserTx(ctx context.Context, tx pgx.Tx, accountID, userID int64, role model.Role) (*model.AccountUser, error) {
	query := `INSERT INTO account_users (account_id, user_id, role) VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	var au model.AccountUser
	err := tx.QueryRow(ctx, query, accountID, userID, role).
		Scan(&au.ID, &au.CreatedAt, &au.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add user to account: %w", err)
	}
	au.AccountID = accountID
	au.UserID = userID
	au.Role = role
	return &au, nil
}

func (r *AccountRepo) AddUser(ctx context.Context, accountID, userID int64, role model.Role) (*model.AccountUser, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	au, err := r.AddUserTx(ctx, tx, accountID, userID, role)
	if err != nil {
		return nil, err
	}
	return au, tx.Commit(ctx)
}
