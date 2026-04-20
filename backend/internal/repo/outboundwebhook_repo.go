package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrWebhookNotFound = errors.New("outbound webhook not found")

const webhookSelectColumns = "id, account_id, url, subscriptions, secret, is_active, created_at, updated_at"

type webhookScanner interface {
	Scan(dest ...any) error
}

func scanOutboundWebhook(scanner webhookScanner, m *model.OutboundWebhook) error {
	return scanner.Scan(&m.ID, &m.AccountID, &m.URL, &m.Subscriptions, &m.Secret, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
}

type OutboundWebhookRepo struct {
	pool *pgxpool.Pool
}

func NewOutboundWebhookRepo(pool *pgxpool.Pool) *OutboundWebhookRepo {
	return &OutboundWebhookRepo{pool: pool}
}

func (r *OutboundWebhookRepo) Create(ctx context.Context, m *model.OutboundWebhook) error {
	query := `INSERT INTO outbound_webhooks (account_id, url, subscriptions, secret)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, m.AccountID, m.URL, m.Subscriptions, m.Secret).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}
	return nil
}

func (r *OutboundWebhookRepo) FindByID(ctx context.Context, id, accountID int64) (*model.OutboundWebhook, error) {
	query := `SELECT ` + webhookSelectColumns + ` FROM outbound_webhooks WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.OutboundWebhook
	if err := scanOutboundWebhook(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrWebhookNotFound, err)
		}
		return nil, fmt.Errorf("failed to find webhook: %w", err)
	}
	return &m, nil
}

func (r *OutboundWebhookRepo) ListByAccount(ctx context.Context, accountID int64) ([]model.OutboundWebhook, error) {
	query := `SELECT ` + webhookSelectColumns + ` FROM outbound_webhooks WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	var result []model.OutboundWebhook
	for rows.Next() {
		var m model.OutboundWebhook
		if err := scanOutboundWebhook(rows, &m); err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *OutboundWebhookRepo) Update(ctx context.Context, m *model.OutboundWebhook) error {
	query := `UPDATE outbound_webhooks SET url = $1, subscriptions = $2, is_active = $3, updated_at = NOW() WHERE id = $4 AND account_id = $5`
	_, err := r.pool.Exec(ctx, query, m.URL, m.Subscriptions, m.IsActive, m.ID, m.AccountID)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	return nil
}

func (r *OutboundWebhookRepo) Delete(ctx context.Context, id, accountID int64) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM outbound_webhooks WHERE id = $1 AND account_id = $2`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: %w", ErrWebhookNotFound, pgx.ErrNoRows)
	}
	return nil
}
