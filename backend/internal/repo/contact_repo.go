package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrContactNotFound    = errors.New("contact not found")
	ErrSameContactMerge   = errors.New("cannot merge contact with itself")
)

const contactSelectColumns = "id, account_id, name, email, phone_number, phone_e164, identifier, additional_attributes, avatar_url, avatar_hash, blocked, last_activity_at, created_at, updated_at, deleted_at"

type contactScanner interface {
	Scan(dest ...any) error
}

func scanContact(scanner contactScanner, m *model.Contact) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Name, &m.Email, &m.PhoneNumber, &m.PhoneE164, &m.Identifier, &m.AdditionalAttrs, &m.AvatarURL, &m.AvatarHash, &m.Blocked, &m.LastActivityAt, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
}

type ContactRepo struct {
	pool *pgxpool.Pool
}

func NewContactRepo(pool *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{pool: pool}
}

func (r *ContactRepo) Create(ctx context.Context, m *model.Contact) error {
	query := `INSERT INTO contacts (account_id, name, email, phone_number, phone_e164, identifier, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Email, m.PhoneNumber, m.PhoneE164, m.Identifier, m.AdditionalAttrs).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}
	return nil
}

func (r *ContactRepo) Upsert(ctx context.Context, m *model.Contact) error {
	query := `INSERT INTO contacts (account_id, name, email, phone_number, phone_e164, identifier, additional_attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ON CONSTRAINT contacts_pkey DO UPDATE SET
			name = COALESCE(EXCLUDED.name, contacts.name),
			email = COALESCE(EXCLUDED.email, contacts.email),
			phone_number = COALESCE(EXCLUDED.phone_number, contacts.phone_number),
			phone_e164 = COALESCE(EXCLUDED.phone_e164, contacts.phone_e164),
			additional_attributes = COALESCE(EXCLUDED.additional_attributes, contacts.additional_attributes),
			updated_at = NOW()
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Name, m.Email, m.PhoneNumber, m.PhoneE164, m.Identifier, m.AdditionalAttrs).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert contact: %w", err)
	}
	return nil
}

func (r *ContactRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE id = $1 AND account_id = $2 AND deleted_at IS NULL`
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

// FindByIDIncludeDeleted finds a contact even if soft-deleted.
func (r *ContactRepo) FindByIDIncludeDeleted(ctx context.Context, id, accountID int64) (*model.Contact, error) {
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
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE identifier = $1 AND account_id = $2 AND deleted_at IS NULL`
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

func (r *ContactRepo) FindByEmail(ctx context.Context, email string, accountID int64) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE LOWER(email) = LOWER($1) AND account_id = $2 AND deleted_at IS NULL LIMIT 1`
	row := r.pool.QueryRow(ctx, query, email, accountID)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("find contact by email: %w", err)
	}
	return &m, nil
}

func (r *ContactRepo) FindByPhoneE164(ctx context.Context, phoneE164 string, accountID int64) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE phone_e164 = $1 AND account_id = $2 AND deleted_at IS NULL LIMIT 1`
	row := r.pool.QueryRow(ctx, query, phoneE164, accountID)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("find contact by phone_e164: %w", err)
	}
	return &m, nil
}

func (r *ContactRepo) FindByPhone(ctx context.Context, phone string, accountID int64) (*model.Contact, error) {
	query := `SELECT ` + contactSelectColumns + ` FROM contacts WHERE phone_number = $1 AND account_id = $2 AND deleted_at IS NULL LIMIT 1`
	row := r.pool.QueryRow(ctx, query, phone, accountID)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("find contact by phone_number: %w", err)
	}
	return &m, nil
}

type ContactFilter struct {
	AccountID int64
	Query     string
	Email     string
	Phone     string
	Labels    []string
	Page      int
	PerPage   int
}

func (r *ContactRepo) Search(ctx context.Context, f ContactFilter) ([]model.Contact, int, error) {
	countQuery := `SELECT COUNT(*) FROM contacts c WHERE c.account_id = $1 AND c.deleted_at IS NULL`
	var args []any
	args = append(args, f.AccountID)
	argN := 2

	joins := ""
	if f.Query != "" {
		countQuery += fmt.Sprintf(` AND (c.name ILIKE $%d OR c.email ILIKE $%d OR c.phone_number ILIKE $%d)`, argN, argN, argN)
		args = append(args, "%"+f.Query+"%")
		argN++
	}

	if len(f.Labels) > 0 {
		joins += ` JOIN contact_inboxes ci ON ci.contact_id = c.id JOIN conversation_labels cl ON cl.conversation_id IN (SELECT id FROM conversations WHERE contact_inbox_id = ci.id) JOIN labels l ON l.id = cl.label_id`
		placeholders := make([]string, len(f.Labels))
		for i, label := range f.Labels {
			placeholders[i] = fmt.Sprintf("$%d", argN)
			args = append(args, label)
			argN++
		}
		countQuery += fmt.Sprintf(` AND l.title IN (%s)`, strings.Join(placeholders, ", "))
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count contacts: %w", err)
	}

	if total == 0 {
		return []model.Contact{}, 0, nil
	}

	dataQuery := `SELECT c.` + contactSelectColumns + ` FROM contacts c` + joins + ` WHERE c.account_id = $1 AND c.deleted_at IS NULL`
	if f.Query != "" {
		dataQuery += ` AND (c.name ILIKE $2 OR c.email ILIKE $2 OR c.phone_number ILIKE $2)`
	}

	offset := (f.Page - 1) * f.PerPage
	dataQuery += fmt.Sprintf(` ORDER BY c.created_at DESC LIMIT %d OFFSET %d`, f.PerPage, offset)

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

func (r *ContactRepo) UpdateIdentifier(ctx context.Context, id, accountID int64, identifier string, name *string, email *string) (*model.Contact, error) {
	query := `UPDATE contacts SET
		identifier = $3,
		name = COALESCE($4, name),
		email = COALESCE($5, email),
		updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + contactSelectColumns
	row := r.pool.QueryRow(ctx, query, id, accountID, identifier, name, email)
	var m model.Contact
	if err := scanContact(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("failed to update contact identifier: %w", err)
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
	conditions = append(conditions, "deleted_at IS NULL")

	if email != "" {
		conditions = append(conditions, fmt.Sprintf("email = $%d", argN))
		args = append(args, email)
		argN++
	}
	if phone != "" {
		conditions = append(conditions, fmt.Sprintf("phone_number = $%d", argN))
		args = append(args, phone)
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

func (r *ContactRepo) UpdateAdditionalAttrs(ctx context.Context, id, accountID int64, attrs string) (*string, error) {
	query := `UPDATE contacts SET additional_attributes = $3, updated_at = NOW() WHERE id = $1 AND account_id = $2 RETURNING additional_attributes`
	var result *string
	if err := r.pool.QueryRow(ctx, query, id, accountID, attrs).Scan(&result); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrContactNotFound, err)
		}
		return nil, fmt.Errorf("failed to update additional_attributes: %w", err)
	}
	return result, nil
}

type ImportContact struct {
	Name  string
	Email string
	Phone string
}

type ImportResult struct {
	Inserted int
	Updated  int
}

func (r *ContactRepo) ImportBatch(ctx context.Context, accountID int64, batch []ImportContact) (ImportResult, error) {
	if len(batch) == 0 {
		return ImportResult{}, nil
	}

	batchExec := &pgx.Batch{}
	for _, c := range batch {
		var phone *string
		if c.Phone != "" {
			phone = &c.Phone
		}
		var email *string
		if c.Email != "" {
			email = &c.Email
		}
		batchExec.Queue(
			`INSERT INTO contacts (account_id, name, email, phone_number) VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_id, lower(email)) WHERE email IS NOT NULL AND email != '' DO UPDATE SET
				name = COALESCE(EXCLUDED.name, contacts.name),
				phone_number = COALESCE(EXCLUDED.phone_number, contacts.phone_number),
				updated_at = NOW()
			RETURNING (xmax = 0) AS is_insert`,
			accountID, c.Name, email, phone,
		)
	}

	br := r.pool.SendBatch(ctx, batchExec)
	defer func() {
		if cerr := br.Close(); cerr != nil {
			_ = cerr
		}
	}()

	var result ImportResult
	for range batch {
		row := br.QueryRow()
		var isInsert bool
		if err := row.Scan(&isInsert); err != nil {
			return result, fmt.Errorf("failed to import contact row: %w", err)
		}
		if isInsert {
			result.Inserted++
		} else {
			result.Updated++
		}
	}

	return result, nil
}

func (r *ContactRepo) Delete(ctx context.Context, id, accountID int64) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE contacts SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND account_id = $2 AND deleted_at IS NULL`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to soft-delete contact: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}

// HardDelete permanently removes a contact row. Use sparingly — most callers
// should use Delete (soft-delete) instead.
func (r *ContactRepo) HardDelete(ctx context.Context, id, accountID int64) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM contacts WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete contact: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}

// UpdateLastActivity bumps last_activity_at to the given timestamp without
// touching updated_at. Called whenever an incoming message from this contact
// is created, mirroring Chatwoot's contact activity heuristic.
func (r *ContactRepo) UpdateLastActivity(ctx context.Context, id, accountID int64, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET last_activity_at = GREATEST(COALESCE(last_activity_at, to_timestamp(0)), $3) WHERE id = $1 AND account_id = $2`,
		id, accountID, at)
	if err != nil {
		return fmt.Errorf("failed to update contact last_activity_at: %w", err)
	}
	return nil
}

func (r *ContactRepo) UpdateBlocked(ctx context.Context, id, accountID int64, blocked bool) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE contacts SET blocked = $3, updated_at = NOW() WHERE id = $1 AND account_id = $2`, id, accountID, blocked)
	if err != nil {
		return fmt.Errorf("failed to update contact blocked: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}

func (r *ContactRepo) UpdateAvatarURL(ctx context.Context, id, accountID int64, url *string) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE contacts SET avatar_url = $3, updated_at = NOW() WHERE id = $1 AND account_id = $2`, id, accountID, url)
	if err != nil {
		return fmt.Errorf("failed to update contact avatar: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}

// UpdateAvatar stores avatar_url and avatar_hash atomically. When caller
// resolves the new hash (from an HTTP HEAD or provider callback), both
// fields are written together so subsequent UpsertContact calls can detect
// content changes without re-downloading the full payload.
func (r *ContactRepo) UpdateAvatar(ctx context.Context, id, accountID int64, url, hash *string) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE contacts SET avatar_url = $3, avatar_hash = $4, updated_at = NOW() WHERE id = $1 AND account_id = $2`, id, accountID, url, hash)
	if err != nil {
		return fmt.Errorf("failed to update contact avatar: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}

// Merge moves all dependent records from childID to primaryID atomically and
// deletes the child. Returns the updated primary contact.
func (r *ContactRepo) Merge(ctx context.Context, childID, primaryID, accountID int64) (*model.Contact, error) {
	if childID == primaryID {
		return nil, ErrSameContactMerge
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin merge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var childExists, primaryExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND account_id = $2)`, childID, accountID).Scan(&childExists); err != nil {
		return nil, fmt.Errorf("check child contact: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND account_id = $2)`, primaryID, accountID).Scan(&primaryExists); err != nil {
		return nil, fmt.Errorf("check primary contact: %w", err)
	}
	if !childExists || !primaryExists {
		return nil, ErrContactNotFound
	}

	if _, err := tx.Exec(ctx, `UPDATE contact_inboxes SET contact_id = $1 WHERE contact_id = $2`, primaryID, childID); err != nil {
		return nil, fmt.Errorf("merge contact_inboxes: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET contact_id = $1 WHERE contact_id = $2 AND account_id = $3`, primaryID, childID, accountID); err != nil {
		return nil, fmt.Errorf("merge conversations: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE notes SET contact_id = $1 WHERE contact_id = $2 AND account_id = $3`, primaryID, childID, accountID); err != nil {
		return nil, fmt.Errorf("merge notes: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO label_taggings (account_id, label_id, taggable_type, taggable_id)
		 SELECT account_id, label_id, 'contact', $1 FROM label_taggings
		 WHERE taggable_type = 'contact' AND taggable_id = $2 AND account_id = $3
		 ON CONFLICT (label_id, taggable_type, taggable_id) DO NOTHING`,
		primaryID, childID, accountID); err != nil {
		return nil, fmt.Errorf("merge label_taggings insert: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM label_taggings WHERE taggable_type = 'contact' AND taggable_id = $1 AND account_id = $2`, childID, accountID); err != nil {
		return nil, fmt.Errorf("merge label_taggings delete: %w", err)
	}
	// additional_attributes merge: primary wins on conflicting keys.
	if _, err := tx.Exec(ctx,
		`UPDATE contacts p SET additional_attributes = COALESCE(c.additional_attributes, '{}'::jsonb) || COALESCE(p.additional_attributes, '{}'::jsonb),
		 updated_at = NOW()
		 FROM contacts c
		 WHERE p.id = $1 AND c.id = $2 AND p.account_id = $3 AND c.account_id = $3`,
		primaryID, childID, accountID); err != nil {
		return nil, fmt.Errorf("merge additional_attributes: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM contacts WHERE id = $1 AND account_id = $2`, childID, accountID); err != nil {
		return nil, fmt.Errorf("delete child contact: %w", err)
	}

	row := tx.QueryRow(ctx, `SELECT `+contactSelectColumns+` FROM contacts WHERE id = $1 AND account_id = $2`, primaryID, accountID)
	var primary model.Contact
	if err := scanContact(row, &primary); err != nil {
		return nil, fmt.Errorf("load merged primary: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit merge: %w", err)
	}
	return &primary, nil
}

func (r *ContactRepo) ListConversationsByContactID(ctx context.Context, contactID, accountID int64, page, perPage int) ([]model.Conversation, int, error) {
	countQuery := `SELECT COUNT(*) FROM conversations WHERE contact_id = $1 AND account_id = $2`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, contactID, accountID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}
	if total == 0 {
		return []model.Conversation{}, 0, nil
	}

	offset := (page - 1) * perPage
	dataQuery := `SELECT ` + conversationSelectColumns + ` FROM conversations WHERE contact_id = $1 AND account_id = $2
		ORDER BY last_activity_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, dataQuery, contactID, accountID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var convos []model.Conversation
	for rows.Next() {
		var m model.Conversation
		if err := scanConversation(rows, &m); err != nil {
			return nil, 0, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convos = append(convos, m)
	}
	return convos, total, rows.Err()
}
