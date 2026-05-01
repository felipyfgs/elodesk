package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLabelNotFound = errors.New("label not found")
var ErrLabelTitleTaken = errors.New("label title already taken")

const labelSelectColumns = "id, account_id, title, color, description, show_on_sidebar, created_at, updated_at"

type labelScanner interface {
	Scan(dest ...any) error
}

func scanLabel(scanner labelScanner, m *model.Label) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.Title, &m.Color, &m.Description, &m.ShowOnSidebar, &m.CreatedAt, &m.UpdatedAt)
}

type LabelRepo struct {
	pool *pgxpool.Pool
}

func NewLabelRepo(pool *pgxpool.Pool) *LabelRepo {
	return &LabelRepo{pool: pool}
}

func (r *LabelRepo) Create(ctx context.Context, m *model.Label) error {
	query := `INSERT INTO labels (account_id, title, color, description, show_on_sidebar)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.Title, m.Color, m.Description, m.ShowOnSidebar).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrLabelTitleTaken, err)
		}
		return fmt.Errorf("failed to create label: %w", err)
	}
	return nil
}

func (r *LabelRepo) FindByID(ctx context.Context, id, accountID int64) (*model.Label, error) {
	query := `SELECT ` + labelSelectColumns + ` FROM labels WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.Label
	if err := scanLabel(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrLabelNotFound, err)
		}
		return nil, fmt.Errorf("failed to find label: %w", err)
	}
	return &m, nil
}

func (r *LabelRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.Label, error) {
	query := `SELECT ` + labelSelectColumns + ` FROM labels WHERE account_id = $1 ORDER BY title ASC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	defer rows.Close()

	var labels []model.Label
	for rows.Next() {
		var m model.Label
		if err := scanLabel(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}
		labels = append(labels, m)
	}
	return labels, rows.Err()
}

func (r *LabelRepo) Update(ctx context.Context, m *model.Label) error {
	query := `UPDATE labels SET title = $3, color = $4, description = $5, show_on_sidebar = $6, updated_at = NOW()
		WHERE id = $1 AND account_id = $2
		RETURNING ` + labelSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.AccountID, m.Title, m.Color, m.Description, m.ShowOnSidebar)
	if err := scanLabel(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrLabelNotFound, err)
		}
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %w", ErrLabelTitleTaken, err)
		}
		return fmt.Errorf("failed to update label: %w", err)
	}
	return nil
}

func (r *LabelRepo) Delete(ctx context.Context, id, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM labels WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete label: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrLabelNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *LabelRepo) ExistsByTitle(ctx context.Context, title string, accountID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM labels WHERE lower(title) = lower($1) AND account_id = $2)`,
		title, accountID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check label title: %w", err)
	}
	return exists, nil
}

func (r *LabelRepo) ApplyLabel(ctx context.Context, accountID, labelID int64, taggableType string, taggableID int64) error {
	query := `INSERT INTO label_taggings (account_id, label_id, taggable_type, taggable_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (label_id, taggable_type, taggable_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, accountID, labelID, taggableType, taggableID)
	if err != nil {
		return fmt.Errorf("failed to apply label: %w", err)
	}
	return nil
}

func (r *LabelRepo) RemoveLabel(ctx context.Context, accountID, labelID int64, taggableType string, taggableID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM label_taggings WHERE account_id = $1 AND label_id = $2 AND taggable_type = $3 AND taggable_id = $4`,
		accountID, labelID, taggableType, taggableID)
	if err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrLabelNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *LabelRepo) ListByTaggable(ctx context.Context, accountID int64, taggableType string, taggableID int64) ([]model.Label, error) {
	query := `SELECT l.id, l.account_id, l.title, l.color, l.description, l.show_on_sidebar, l.created_at, l.updated_at FROM labels l
		INNER JOIN label_taggings lt ON lt.label_id = l.id
		WHERE lt.account_id = $1 AND lt.taggable_type = $2 AND lt.taggable_id = $3
		ORDER BY l.title ASC`
	rows, err := r.pool.Query(ctx, query, accountID, taggableType, taggableID)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels by taggable: %w", err)
	}
	defer rows.Close()

	var labels []model.Label
	for rows.Next() {
		var m model.Label
		if err := scanLabel(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}
		labels = append(labels, m)
	}
	return labels, rows.Err()
}
