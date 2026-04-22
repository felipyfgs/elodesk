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

// generateToken creates a cryptographically random 48-byte token encoded as base64url (64 chars)
func generateToken() (string, error) {
	bytes := make([]byte, 48)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Create generates and inserts a new access token for the given owner
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
			// A token already exists for this owner (conflict); return it.
			return r.FindByOwner(ctx, ownerType, ownerID)
		}
		return nil, fmt.Errorf("failed to create user access token: %w", err)
	}

	m.OwnerType = ownerType
	m.OwnerID = ownerID
	m.Token = token
	return &m, nil
}

// CreateTx creates a token within a transaction
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
			// Conflict: token already exists for this owner; return it.
			return r.FindByOwner(ctx, ownerType, ownerID)
		}
		return nil, fmt.Errorf("failed to create user access token: %w", err)
	}

	m.OwnerType = ownerType
	m.OwnerID = ownerID
	m.Token = token
	return &m, nil
}

// FindByToken looks up a token by its plaintext value
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

// FindByOwner finds the token for a specific owner (polymorphic)
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

// Regenerate deletes the old token and creates a new one for the owner
func (r *UserAccessTokenRepo) Regenerate(ctx context.Context, ownerType string, ownerID int64) (*model.UserAccessToken, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete existing token
	_, err = tx.Exec(ctx,
		`DELETE FROM user_access_tokens WHERE owner_type = $1 AND owner_id = $2`,
		ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old token: %w", err)
	}

	// Create new token
	newToken, err := r.CreateTx(ctx, tx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit token regeneration: %w", err)
	}

	return newToken, nil
}

// EnsureForUser creates a token for a user if they don't already have one
// Used for backfilling existing users
func (r *UserAccessTokenRepo) EnsureForUser(ctx context.Context, userID int64) (*model.UserAccessToken, error) {
	// Try to find existing token
	existing, err := r.FindByOwner(ctx, "User", userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrUserAccessTokenNotFound) {
		return nil, err
	}

	// Create new token
	return r.Create(ctx, "User", userID)
}

// ListUsersWithoutToken returns all user IDs that don't have an access token
func (r *UserAccessTokenRepo) ListUsersWithoutToken(ctx context.Context) ([]int64, error) {
	query := `
		SELECT u.id FROM users u
		LEFT JOIN user_access_tokens uat ON uat.owner_type = 'User' AND uat.owner_id = u.id
		WHERE uat.id IS NULL`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users without token: %w", err)
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan user id: %w", err)
		}
		userIDs = append(userIDs, id)
	}

	return userIDs, rows.Err()
}
