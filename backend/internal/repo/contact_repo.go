package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrContactNotFound = errors.New("contact not found")

const contactSelectColumns = "id, account_id, name, email, phone_number, identifier, additional_attributes, last_activity_at, created_at, updated_at"

type contactScanner interface {
	Scan(dest ...any) error
}

func scanContact(scanner contactScanner, m *model.Contact) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Name, &m.Email, &m.PhoneNumber, &m.Identifier, &m.AdditionalAttrs, &m.LastActivityAt, &m.CreatedAt, &m.UpdatedAt)
}

type ContactRepo struct {
	pool *pgxpool.Pool
}

func NewContactRepo(pool *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{pool: pool}
}

func (r *ContactRepo) Create(ctx context.Context, m *model.Contact) error {
	query := `INSERT INTO contacts (account_id, name, email, phone_number, identifier, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Email, m.PhoneNumber, m.Identifier, m.AdditionalAttrs).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}
	return nil
}

func (r *ContactRepo) Upsert(ctx context.Context, m *model.Contact) error {
	query := `INSERT INTO contacts (account_id, name, email, phone_number, identifier, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT ON CONSTRAINT contacts_pkey DO UPDATE SET
			name = COALESCE(EXCLUDED.name, contacts.name),
			email = COALESCE(EXCLUDED.email, contacts.email),
			phone_number = COALESCE(EXCLUDED.phone_number, contacts.phone_number),
			additional_attributes = COALESCE(EXCLUDED.additional_attributes, contacts.additional_attributes),
			updated_at = NOW()
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Email, m.PhoneNumber, m.Identifier, m.AdditionalAttrs).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert contact: %w", err)
	}
	return nil
}

func (r *ContactRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("failed to find contact: %w", err)
	}
	return &m, nil
}

func (r *ContactRepo) FindByIdentifier(ctx context.Context, identifier, accountID string) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE identifier = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, identifier, accountID)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("failed to find contact by identifier: %w", err)
	}
	return &m, nil
}

type ContactFilter struct {
	AccountID int64
	Query     string
	Email     string
	Phone     string
	Page      int
	PerPage   int
}

func (r *ContactRepo) Search(ctx context.Context, f ContactFilter) ([]model.Contact, int, error) {
	countQuery := `SELECT COUNT(*) FROM contacts WHERE account_id = $1`
	var args []any
	args = append(args, f.AccountID)

	if f.Query != "" {
		countQuery += ` AND (name ILIKE $2 OR email ILIKE $2 OR phone_number ILIKE $2)`
		args = append(args, "%"+f.Query+"%")
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count contacts: %w", err)
	}

	if total == 0 {
		return []model.Contact{}, 0, nil
	}

	dataQuery := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE account_id = $1`
	if f.Query != "" {
		dataQuery += ` AND (name ILIKE $2 OR email ILIKE $2 OR phone_number ILIKE $2)`
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search contacts: %w", err)
	}
	defer rows.Close()

	var contacts []model.Contact
	for rows.Next() {
		var m model.Contact
		if err := scanContact(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, m)
	}
	return contacts, total, rows.Err()
}

func (r *ContactRepo) Update(ctx context.Context, id, accountID int64, name, email, phone *string) (*model.Contact, error) {
	query := `UPDATE contacts SET
		name = COALESCE($3, name),
		email = COALESCE($4, email),
		phone_number = COALESCE($5, phone_number),
		updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + contactSelectColumns
	row := r.pool.QueryRow(ctx, query, id, accountID, name, email, phone)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return &m, nil
}

func (r *ContactRepo) Filter(ctx context.Context, accountID int64, email, phone string) ([]model.Contact, error) {
	var conditions []string
	var args []any
	argN := 1

	conditions = append(conditions, fmt.Sprintf("account_id = $%d", argN))
	args = append(args, accountID)
	argN++

	if email != "" {
		conditions = append(conditions, fmt.Sprintf("email = $%d", argN))
		args = append(args, email)
		argN++
	}
	if phone != "" {
		conditions = append(conditions, fmt.Sprintf("phone_number = $%d", argN))
		args = append(args, phone)
		argN++
	}

	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE ` + strings.Join(conditions, " AND ")
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to filter contacts: %w", err)
	}
	defer rows.Close()

	var contacts []model.Contact
	for rows.Next() {
		var m model.Contact
		if err := scanContact(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, m)
	}
	return contacts, rows.Err()
}
