---
name: go-db-migrations
description: Database migration standards - SQL migrations with embed, advisory locks and transactional execution
license: MIT
---

## Purpose

Apply consistent database migration patterns for Go backend projects.

## Migration System

Use embedded SQL migrations with:
- `go:embed` directive for SQL files
- Migration tracking table
- Advisory locks for concurrent safety
- Transactional execution per migration

## File Structure

```
migrations/
├── embed.go          # go:embed directive
├── 0001_init.sql     # Initial schema
├── 0002_feature.sql  # Feature migration
└── ...
```

## Naming Convention

```
NNNN_description.sql
```

- `NNNN` = 4-digit sequential number (0001, 0002, ...)
- `description` = brief snake_case description

## Embed Setup

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

## SQL Patterns

### Create Table
```sql
CREATE TABLE IF NOT EXISTS table_name (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Create Index
```sql
CREATE INDEX IF NOT EXISTS idx_table_column ON table_name(column_name);
```

### Add Column
```sql
ALTER TABLE table_name ADD COLUMN IF NOT EXISTS column_name VARCHAR(500);
```

### Add Foreign Key
```sql
ALTER TABLE table_name ADD CONSTRAINT fk_table_column
    FOREIGN KEY (column_id) REFERENCES other_table(id) ON DELETE CASCADE;
```

### Create Enum
```sql
DO $$ BEGIN
    CREATE TYPE enum_name AS ENUM ('value1', 'value2');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;
```

## Database Naming

- Tables: `snake_case` plural
- Columns: `snake_case`
- Indexes: `idx_<table>_<column>`
- Foreign keys: `fk_<table>_<column>`
- Enums: `snake_case` singular

## Type Mapping

| Go Type | PostgreSQL Type |
|---------|----------------|
| `int64` | `BIGINT` / `BIGSERIAL` |
| `string` | `VARCHAR(n)` / `TEXT` |
| `time.Time` | `TIMESTAMPTZ` |
| `bool` | `BOOLEAN` |
| `[]byte` | `BYTEA` |
| `json.RawMessage` | `JSONB` |

## Required Columns

All tables must have:
```sql
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

All data tables must have tenant scope:
```sql
account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
```

## Rules

1. Always use `IF NOT EXISTS` for idempotent migrations
2. Each migration runs in its own transaction
3. Never modify production data without a separate migration
4. Always include `account_id` in data tables
5. Always include `created_at`/`updated_at` in all tables
6. Index foreign keys and frequently queried columns
7. Use `ON DELETE CASCADE` for referential integrity
8. Never skip or reuse migration numbers

## Workflow Commands

### Create new migration file
```bash
cd backend/migrations && ls *.sql | tail -1 | cut -d_ -f1 | awk '{printf "%04d_new_feature.sql\n", $1+1}'
```

### List existing migrations
```bash
cd backend && ls migrations/*.sql
```

### Verify embed directive
```bash
cd backend && grep -n 'go:embed' migrations/embed.go
```

### Check tables without tenant scope
```bash
cd backend && grep -l 'CREATE TABLE' migrations/*.sql | xargs grep -L 'account_id' | grep -v "0001_init"
```

### Run migrations
```bash
cd backend && go run cmd/backend/main.go
```
