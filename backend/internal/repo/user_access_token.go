package repo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserAccessTokenNotFound = errors.New("user access token not found")

const userAccessTokenSelectColumns = "id, owner_type, owner_id, token, created_at, updated_at"

type userAccessTokenScanner interface {
	Scan(dest ...any) error
}

func scanUserAccessToken(scanner userAccessTokenScanner, m *model.UserAccessToken) error {
	return scanner.Scan(&m.ID, &m.OwnerType, &m.OwnerID, &m.Token, &m.CreatedAt, &m.UpdatedAt)
}

type UserAccessTokenRepo struct {
	pool *pgxpool.Pool
}

func NewUserAccessTokenRepo(pool *pgxpool.Pool) *UserAccessTokenRepo {
	return &UserAccessTokenRepo{pool: pool}
}

func (r *UserAccessTokenRepo) Pool() *pgxpool.Pool { return r.pool }

func generateToken() (string, error) {
	bytes := make([]byte, 48)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (r *UserAccessTokenRepo) Create(ctx context.Context, ownerType string, ownerID int64) (*model.UserAccessToken, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO user_access_tokens (owner_type, owner_id, token)
		VALUES ($1, $2, $3)
		ON CONFLICT (owner_type, owner_id) DO NOTHING
		RETURNING id, created_at, updated_at`

	var m model.UserAccessToken
	err = r.pool.QueryRow(ctx, query, ownerType, ownerID, token).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.FindByOwner(ctx, ownerType, ownerID)
		}
		return nil, fmt.Errorf("failed to create user access token: %w", err)
	}

	m.OwnerType = ownerType
	m.OwnerID = ownerID
	m.Token = token
	return &m, nil
}

func (r *UserAccessTokenRepo) CreateTx(ctx context.Context, tx pgx.Tx, ownerType string, ownerID int64) (*model.UserAccessToken, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO user_access_tokens (owner_type, owner_id, token)
		VALUES ($1, $2, $3)
		ON CONFLICT (owner_type, owner_id) DO NOTHING
		RETURNING id, created_at, updated_at`

	var m model.UserAccessToken
	err = tx.QueryRow(ctx, query, ownerType, ownerID, token).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.FindByOwner(ctx, ownerType, ownerID)
		}
		return nil, fmt.Errorf("failed to create user access token: %w", err)
	}

	m.OwnerType = ownerType
	m.OwnerID = ownerID
	m.Token = token
	return &m, nil
}

func (r *UserAccessTokenRepo) FindByToken(ctx context.Context, token string) (*model.UserAccessToken, error) {
	query := `SELECT ` + userAccessTokenSelectColumns + ` FROM user_access_tokens WHERE token = $1`
	row := r.pool.QueryRow(ctx, query, token)
	var m model.UserAccessToken
	if err := scanUserAccessToken(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrUserAccessTokenNotFound, err)
		}
		return nil, fmt.Errorf("failed to find user access token: %w", err)
	}
	return &m, nil
}

func (r *UserAccessTokenRepo) FindByOwner(ctx context.Context, ownerType string, ownerID int64) (*model.UserAccessToken, error) {
	query := `SELECT ` + userAccessTokenSelectColumns + ` FROM user_access_tokens WHERE owner_type = $1 AND owner_id = $2`
	row := r.pool.QueryRow(ctx, query, ownerType, ownerID)
	var m model.UserAccessToken
	if err := scanUserAccessToken(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrUserAccessTokenNotFound, err)
		}
		return nil, fmt.Errorf("failed to find user access token by owner: %w", err)
	}
	return &m, nil
}

func (r *UserAccessTokenRepo) Regenerate(ctx context.Context, ownerType string, ownerID int64) (*model.UserAccessToken, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`DELETE FROM user_access_tokens WHERE owner_type = $1 AND owner_id = $2`,
		ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old token: %w", err)
	}

	newToken, err := r.CreateTx(ctx, tx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit token regeneration: %w", err)
	}

	return newToken, nil
}

func (r *UserAccessTokenRepo) EnsureForUser(ctx context.Context, userID int64) (*model.UserAccessToken, error) {
	existing, err := r.FindByOwner(ctx, "User", userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrUserAccessTokenNotFound) {
		return nil, err
	}

	return r.Create(ctx, "User", userID)
}
