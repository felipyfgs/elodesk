package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInboxBusinessHoursNotFound = errors.New("inbox business hours not found")

const inboxBusinessHoursSelectColumns = "id, account_id, inbox_id, timezone, schedule, created_at, updated_at"

type InboxBusinessHoursRepo struct {
	pool *pgxpool.Pool
}

func NewInboxBusinessHoursRepo(pool *pgxpool.Pool) *InboxBusinessHoursRepo {
	return &InboxBusinessHoursRepo{pool: pool}
}

func scanInboxBusinessHours(scanner interface{ Scan(dest ...any) error }, m *model.InboxBusinessHours) error {
	var scheduleBytes []byte
	if err := scanner.Scan(&m.ID, &m.AccountID, &m.InboxID, &m.Timezone, &scheduleBytes, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return err
	}
	if len(scheduleBytes) == 0 {
		m.Schedule = map[string]model.BusinessHoursSlot{}
		return nil
	}
	if err := json.Unmarshal(scheduleBytes, &m.Schedule); err != nil {
		return fmt.Errorf("decode inbox business hours schedule: %w", err)
	}
	return nil
}

func (r *InboxBusinessHoursRepo) FindByInbox(ctx context.Context, inboxID, accountID int64) (*model.InboxBusinessHours, error) {
	query := `SELECT ` + inboxBusinessHoursSelectColumns + `
		FROM inbox_business_hours
		WHERE inbox_id = $1 AND account_id = $2`
	var m model.InboxBusinessHours
	if err := scanInboxBusinessHours(r.pool.QueryRow(ctx, query, inboxID, accountID), &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrInboxBusinessHoursNotFound, err)
		}
		return nil, fmt.Errorf("find inbox business hours: %w", err)
	}
	return &m, nil
}

func (r *InboxBusinessHoursRepo) Upsert(ctx context.Context, m *model.InboxBusinessHours) error {
	scheduleBytes, err := json.Marshal(m.Schedule)
	if err != nil {
		return fmt.Errorf("encode inbox business hours schedule: %w", err)
	}
	query := `INSERT INTO inbox_business_hours (account_id, inbox_id, timezone, schedule)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (inbox_id) DO UPDATE SET
			timezone = EXCLUDED.timezone,
			schedule = EXCLUDED.schedule,
			updated_at = NOW()
		WHERE inbox_business_hours.account_id = EXCLUDED.account_id
		RETURNING ` + inboxBusinessHoursSelectColumns
	if err := scanInboxBusinessHours(r.pool.QueryRow(ctx, query, m.AccountID, m.InboxID, m.Timezone, scheduleBytes), m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrInboxBusinessHoursNotFound, err)
		}
		return fmt.Errorf("upsert inbox business hours: %w", err)
	}
	return nil
}
