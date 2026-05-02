package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPipelineCardNotFound = errors.New("pipeline card not found")

// CardWithRelations bundles a card with its joined assignee + label IDs.
// Lives here (repo layer) to keep dto package free of repo dependencies.
type CardWithRelations struct {
	Card        model.PipelineCard
	AssigneeIDs []int64
	LabelIDs    []int64
}

const pipelineCardSelectColumns = "id, pipeline_id, stage_id, position, title, description, value_cents, value_currency, due_date, custom_attrs, linked_entity_type, linked_entity_id, created_by, created_at, updated_at"

type cardScanner interface {
	Scan(dest ...any) error
}

func scanCard(scanner cardScanner, m *model.PipelineCard) error {
	return scanner.Scan(
		&m.ID, &m.PipelineID, &m.StageID, &m.Position, &m.Title, &m.Description,
		&m.ValueCents, &m.ValueCurrency, &m.DueDate, &m.CustomAttrs,
		&m.LinkedEntityType, &m.LinkedEntityID, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
	)
}

type PipelineCardRepo struct {
	pool *pgxpool.Pool
}

func NewPipelineCardRepo(pool *pgxpool.Pool) *PipelineCardRepo {
	return &PipelineCardRepo{pool: pool}
}

// Insert creates a new card. CustomAttrs string SHOULD be valid JSON ('{}' if empty).
func (r *PipelineCardRepo) Insert(ctx context.Context, m *model.PipelineCard) error {
	if m.CustomAttrs == "" {
		m.CustomAttrs = "{}"
	}
	query := `INSERT INTO pipeline_cards
		(pipeline_id, stage_id, position, title, description, value_cents, value_currency, due_date, custom_attrs, linked_entity_type, linked_entity_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`
	row := r.pool.QueryRow(ctx, query,
		m.PipelineID, m.StageID, m.Position, m.Title, m.Description,
		m.ValueCents, m.ValueCurrency, m.DueDate, m.CustomAttrs,
		m.LinkedEntityType, m.LinkedEntityID, m.CreatedBy,
	)
	if err := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return fmt.Errorf("failed to insert card: %w", err)
	}
	return nil
}

// FindByID returns the card if it exists and its pipeline belongs to the given account.
func (r *PipelineCardRepo) FindByID(ctx context.Context, id, accountID int64) (*model.PipelineCard, error) {
	query := `SELECT ` + qualifyColumns(pipelineCardSelectColumns, "c") + `
		FROM pipeline_cards c
		JOIN pipelines p ON p.id = c.pipeline_id
		WHERE c.id = $1 AND p.account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.PipelineCard
	if err := scanCard(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrPipelineCardNotFound, err)
		}
		return nil, fmt.Errorf("failed to find card: %w", err)
	}
	return &m, nil
}

// ListByPipelineWithRelations loads all cards for a pipeline with their assignee and label IDs.
func (r *PipelineCardRepo) ListByPipelineWithRelations(ctx context.Context, pipelineID int64) ([]CardWithRelations, error) {
	query := `SELECT ` + qualifyColumns(pipelineCardSelectColumns, "c") + `,
		COALESCE(ARRAY(SELECT user_id FROM pipeline_card_assignees WHERE card_id = c.id ORDER BY assigned_at), '{}'::bigint[]) AS assignee_ids,
		COALESCE(ARRAY(SELECT label_id FROM pipeline_card_labels WHERE card_id = c.id), '{}'::bigint[]) AS label_ids
		FROM pipeline_cards c
		WHERE c.pipeline_id = $1
		ORDER BY c.stage_id, c.position`
	rows, err := r.pool.Query(ctx, query, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cards with relations: %w", err)
	}
	defer rows.Close()
	var out []CardWithRelations
	for rows.Next() {
		var rel CardWithRelations
		if err := rows.Scan(
			&rel.Card.ID, &rel.Card.PipelineID, &rel.Card.StageID, &rel.Card.Position,
			&rel.Card.Title, &rel.Card.Description, &rel.Card.ValueCents, &rel.Card.ValueCurrency,
			&rel.Card.DueDate, &rel.Card.CustomAttrs, &rel.Card.LinkedEntityType, &rel.Card.LinkedEntityID,
			&rel.Card.CreatedBy, &rel.Card.CreatedAt, &rel.Card.UpdatedAt,
			&rel.AssigneeIDs, &rel.LabelIDs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan card with relations: %w", err)
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

// FindByIDWithRelations returns the card with assignees and labels populated.
func (r *PipelineCardRepo) FindByIDWithRelations(ctx context.Context, id, accountID int64) (*CardWithRelations, error) {
	query := `SELECT ` + qualifyColumns(pipelineCardSelectColumns, "c") + `,
		COALESCE(ARRAY(SELECT user_id FROM pipeline_card_assignees WHERE card_id = c.id ORDER BY assigned_at), '{}'::bigint[]) AS assignee_ids,
		COALESCE(ARRAY(SELECT label_id FROM pipeline_card_labels WHERE card_id = c.id), '{}'::bigint[]) AS label_ids
		FROM pipeline_cards c
		JOIN pipelines p ON p.id = c.pipeline_id
		WHERE c.id = $1 AND p.account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var rel CardWithRelations
	err := row.Scan(
		&rel.Card.ID, &rel.Card.PipelineID, &rel.Card.StageID, &rel.Card.Position,
		&rel.Card.Title, &rel.Card.Description, &rel.Card.ValueCents, &rel.Card.ValueCurrency,
		&rel.Card.DueDate, &rel.Card.CustomAttrs, &rel.Card.LinkedEntityType, &rel.Card.LinkedEntityID,
		&rel.Card.CreatedBy, &rel.Card.CreatedAt, &rel.Card.UpdatedAt,
		&rel.AssigneeIDs, &rel.LabelIDs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrPipelineCardNotFound, err)
		}
		return nil, fmt.Errorf("failed to find card with relations: %w", err)
	}
	return &rel, nil
}

// Update writes mutable fields back to the row.
func (r *PipelineCardRepo) Update(ctx context.Context, m *model.PipelineCard) error {
	if m.CustomAttrs == "" {
		m.CustomAttrs = "{}"
	}
	query := `UPDATE pipeline_cards
		SET title = $2, description = $3, value_cents = $4, value_currency = $5, due_date = $6, custom_attrs = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + pipelineCardSelectColumns
	row := r.pool.QueryRow(ctx, query, m.ID, m.Title, m.Description, m.ValueCents, m.ValueCurrency, m.DueDate, m.CustomAttrs)
	if err := scanCard(row, m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrPipelineCardNotFound, err)
		}
		return fmt.Errorf("failed to update card: %w", err)
	}
	return nil
}

// Move atomically updates stage_id + position.
func (r *PipelineCardRepo) Move(ctx context.Context, cardID, stageID int64, position float64) (*model.PipelineCard, error) {
	query := `UPDATE pipeline_cards
		SET stage_id = $2, position = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + pipelineCardSelectColumns
	row := r.pool.QueryRow(ctx, query, cardID, stageID, position)
	var m model.PipelineCard
	if err := scanCard(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrPipelineCardNotFound, err)
		}
		return nil, fmt.Errorf("failed to move card: %w", err)
	}
	return &m, nil
}

func (r *PipelineCardRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM pipeline_cards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrPipelineCardNotFound, pgx.ErrNoRows)
	}
	return nil
}

func (r *PipelineCardRepo) MaxPositionInStage(ctx context.Context, stageID int64) (float64, error) {
	var max *float64
	if err := r.pool.QueryRow(ctx, `SELECT MAX(position) FROM pipeline_cards WHERE stage_id = $1`, stageID).Scan(&max); err != nil {
		return 0, fmt.Errorf("failed to compute max card position: %w", err)
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

func (r *PipelineCardRepo) CountByStage(ctx context.Context, stageID int64) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pipeline_cards WHERE stage_id = $1`, stageID).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count cards in stage: %w", err)
	}
	return n, nil
}

// CountByPipeline returns count of cards per pipeline within an account, used for index list.
func (r *PipelineCardRepo) CountByPipelineForAccount(ctx context.Context, accountID int64) (map[int64]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.pipeline_id, COUNT(*) FROM pipeline_cards c
		 JOIN pipelines p ON p.id = c.pipeline_id
		 WHERE p.account_id = $1
		 GROUP BY c.pipeline_id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to count cards: %w", err)
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var pid int64
		var n int
		if err := rows.Scan(&pid, &n); err != nil {
			return nil, err
		}
		out[pid] = n
	}
	return out, rows.Err()
}

// Rebalance renumbers card positions inside a stage as 100, 200, 300, ...
func (r *PipelineCardRepo) Rebalance(ctx context.Context, stageID int64) ([]model.PipelineCard, error) {
	if err := rebalancePositions(ctx, r.pool, rebalanceCardQueries, stageID); err != nil {
		return nil, err
	}
	return r.listByStageInternal(ctx, stageID)
}

func (r *PipelineCardRepo) listByStageInternal(ctx context.Context, stageID int64) ([]model.PipelineCard, error) {
	query := `SELECT ` + pipelineCardSelectColumns + ` FROM pipeline_cards WHERE stage_id = $1 ORDER BY position ASC`
	rows, err := r.pool.Query(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.PipelineCard
	for rows.Next() {
		var m model.PipelineCard
		if err := scanCard(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ===== Assignees =====

func (r *PipelineCardRepo) AddAssignee(ctx context.Context, cardID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO pipeline_card_assignees (card_id, user_id) VALUES ($1, $2)
		 ON CONFLICT (card_id, user_id) DO NOTHING`,
		cardID, userID)
	if err != nil {
		return fmt.Errorf("failed to add assignee: %w", err)
	}
	return nil
}

func (r *PipelineCardRepo) RemoveAssignee(ctx context.Context, cardID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM pipeline_card_assignees WHERE card_id = $1 AND user_id = $2`,
		cardID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove assignee: %w", err)
	}
	return nil
}

func (r *PipelineCardRepo) ListAssigneeIDs(ctx context.Context, cardID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM pipeline_card_assignees WHERE card_id = $1 ORDER BY assigned_at`, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignees: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ===== Labels =====

func (r *PipelineCardRepo) AddLabel(ctx context.Context, cardID, labelID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO pipeline_card_labels (card_id, label_id) VALUES ($1, $2)
		 ON CONFLICT (card_id, label_id) DO NOTHING`,
		cardID, labelID)
	if err != nil {
		return fmt.Errorf("failed to add card label: %w", err)
	}
	return nil
}

func (r *PipelineCardRepo) RemoveLabel(ctx context.Context, cardID, labelID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM pipeline_card_labels WHERE card_id = $1 AND label_id = $2`,
		cardID, labelID)
	if err != nil {
		return fmt.Errorf("failed to remove card label: %w", err)
	}
	return nil
}

func (r *PipelineCardRepo) ListLabelIDs(ctx context.Context, cardID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT label_id FROM pipeline_card_labels WHERE card_id = $1`, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to list card labels: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// qualifyColumns prefixes each comma-separated column with the given alias.
// Used to disambiguate when JOINing pipelines for account_id checks.
func qualifyColumns(cols, alias string) string {
	out := ""
	start := 0
	for i := 0; i <= len(cols); i++ {
		if i != len(cols) && cols[i] != ',' {
			continue
		}
		seg := cols[start:i]
		for len(seg) > 0 && seg[0] == ' ' {
			seg = seg[1:]
		}
		if out != "" {
			out += ", "
		}
		out += alias + "." + seg
		start = i + 1
	}
	return out
}
