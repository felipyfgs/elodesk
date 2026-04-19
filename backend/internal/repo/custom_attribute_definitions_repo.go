package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/model"
)

var ErrCustomAttributeDefinitionNotFound = errors.New("custom attribute definition not found")

const customAttrDefSelectColumns = "id, account_id, attribute_key, attribute_display_name, attribute_display_type, attribute_model, attribute_values, attribute_description, regex_pattern, default_value, created_at, updated_at"

func scanCustomAttrDef(scanner interface{ Scan(dest ...any) error }, m *model.CustomAttributeDefinition) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.AttributeKey, &m.AttributeDisplayName, &m.AttributeDisplayType, &m.AttributeModel, &m.AttributeValues, &m.AttributeDescription, &m.RegexPattern, &m.DefaultValue, &m.CreatedAt, &m.UpdatedAt)
}

type CustomAttributeDefinitionRepo struct {
	pool *pgxpool.Pool
}

func NewCustomAttributeDefinitionRepo(pool *pgxpool.Pool) *CustomAttributeDefinitionRepo {
	return &CustomAttributeDefinitionRepo{pool: pool}
}

func (r *CustomAttributeDefinitionRepo) Create(ctx context.Context, m *model.CustomAttributeDefinition) error {
	query := `INSERT INTO custom_attribute_definitions (account_id, attribute_key, attribute_display_name, attribute_display_type, attribute_model, attribute_values, attribute_description, regex_pattern, default_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.AttributeKey, m.AttributeDisplayName, m.AttributeDisplayType, m.AttributeModel, m.AttributeValues, m.AttributeDescription, m.RegexPattern, m.DefaultValue).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("attribute_key already exists for this model: %w", err)
		}
		return fmt.Errorf("failed to create custom attribute definition: %w", err)
	}
	return nil
}

func (r *CustomAttributeDefinitionRepo) FindByID(ctx context.Context, id, accountID int64) (*model.CustomAttributeDefinition, error) {
	query := `SELECT ` + customAttrDefSelectColumns + ` FROM custom_attribute_definitions WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.CustomAttributeDefinition
	if err := scanCustomAttrDef(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrCustomAttributeDefinitionNotFound, err)
		}
		return nil, fmt.Errorf("failed to find custom attribute definition: %w", err)
	}
	return &m, nil
}

func (r *CustomAttributeDefinitionRepo) ListByAccount(ctx context.Context, accountID int64, attributeModel string) ([]model.CustomAttributeDefinition, error) {
	query := `SELECT ` + customAttrDefSelectColumns + ` FROM custom_attribute_definitions WHERE account_id = $1`
	var args []any
	args = append(args, accountID)
	if attributeModel != "" {
		query += ` AND attribute_model = $2`
		args = append(args, attributeModel)
	}
	query += ` ORDER BY attribute_display_name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom attribute definitions: %w", err)
	}
	defer rows.Close()

	var defs []model.CustomAttributeDefinition
	for rows.Next() {
		var m model.CustomAttributeDefinition
		if err := scanCustomAttrDef(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan custom attribute definition: %w", err)
		}
		defs = append(defs, m)
	}
	return defs, rows.Err()
}

func (r *CustomAttributeDefinitionRepo) Update(ctx context.Context, m *model.CustomAttributeDefinition) error {
	query := `UPDATE custom_attribute_definitions SET
		attribute_key = $3, attribute_display_name = $4, attribute_display_type = $5,
		attribute_model = $6, attribute_values = $7, attribute_description = $8,
		regex_pattern = $9, default_value = $10, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + customAttrDefSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.AttributeKey, m.AttributeDisplayName, m.AttributeDisplayType, m.AttributeModel, m.AttributeValues, m.AttributeDescription, m.RegexPattern, m.DefaultValue)
	if err := scanCustomAttrDef(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrCustomAttributeDefinitionNotFound, err)
		}
		return fmt.Errorf("failed to update custom attribute definition: %w", err)
	}
	return nil
}

func (r *CustomAttributeDefinitionRepo) Delete(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM custom_attribute_definitions WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete custom attribute definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrCustomAttributeDefinitionNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *CustomAttributeDefinitionRepo) FindByKeyAndModel(ctx context.Context, accountID int64, attributeKey, attributeModel string) (*model.CustomAttributeDefinition, error) {
	query := `SELECT ` + customAttrDefSelectColumns + ` FROM custom_attribute_definitions
		WHERE account_id = $1 AND attribute_key = $2 AND attribute_model = $3`
	row := r.pool.QueryRow(ctx, query, accountID, attributeKey, attributeModel)
	var m model.CustomAttributeDefinition
	if err := scanCustomAttrDef(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrCustomAttributeDefinitionNotFound, err)
		}
		return nil, fmt.Errorf("failed to find custom attribute definition: %w", err)
	}
	return &m, nil
}

func (r *CustomAttributeDefinitionRepo) ListKeysByModel(ctx context.Context, accountID int64, attributeModel string) ([]string, error) {
	query := `SELECT attribute_key FROM custom_attribute_definitions WHERE account_id = $1 AND attribute_model = $2`
	rows, err := r.pool.Query(ctx, query, accountID, attributeModel)
	if err != nil {
		return nil, fmt.Errorf("failed to list attribute keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("failed to scan attribute key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
