package database

import (
	"context"
	"fmt"
	"sort"

	"backend/internal/logger"
	"backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return fmt.Errorf("failed to ensure migration tracking table: %w", err)
	}

	applied, err := getAppliedMigrations(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var pending []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !applied[name] {
			pending = append(pending, name)
		}
	}

	sort.Strings(pending)

	for _, name := range pending {
		if err := applyMigration(ctx, pool, name); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", name, err)
		}
		logger.Info().Str("component", "db").Str("file", name).Msg("Migration applied")
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`
	_, err := pool.Exec(ctx, query)
	return err
}

func getAppliedMigrations(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, fileName string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1)"); err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", fileName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}
	if exists {
		logger.Info().Str("component", "db").Str("file", fileName).Msg("Migration already applied, skipping")
		return nil
	}

	sqlBytes, err := migrations.FS.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to read migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
	}

	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", fileName); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", fileName, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", fileName, err)
	}

	return nil
}
