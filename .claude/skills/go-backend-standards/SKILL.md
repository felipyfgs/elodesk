---
name: go-backend-standards
description: Use when creating or refactoring Go backend code under backend/internal/ — handler/service/repo layers, DI wiring in server/router.go, struct/package naming, channel registry interfaces, DTO Req/Resp suffixes. Apply on new files, renames and cross-layer refactors.
license: MIT
---

## Purpose

Apply Go backend standards when creating, modifying or reviewing Go code in a layered architecture project.

## Architecture

Follow a strict layered architecture:

```
HTTP Handler → Service → Repository → Database
```

Each layer has a single responsibility:
- **Handler**: Parse requests, validate input, format responses
- **Service**: Business logic and orchestration
- **Repository**: Database queries and data mapping
- **DTO**: Request/response shapes
- **Model**: Domain entities

## Naming Conventions

### Files
- Go files: `snake_case.go`
- Test files: `*_test.go`

### Packages
- Lowercase, single word when possible
- No underscores, no mixed case

### Types
- Structs: `PascalCase`
- Interfaces: `PascalCase`, capability-describing names
- Constants: `PascalCase` (exported), `camelCase` (unexported)
- Sentinel errors: `PascalCase` with `Err` prefix

### Functions
- Exported: `PascalCase`
- Unexported: `camelCase`
- Constructors: `New<Type>(...) *Type`

### Variables
- `camelCase` for package/function scope
- Short names in tight scopes: `ctx`, `db`, `cfg`, `err`

### JSON Tags
- `camelCase` in tags
- Sensitive fields: `json:"-"`

### DTOs
- Requests: `*Req` suffix
- Responses: `*Resp` suffix

### Database
- Tables: `snake_case` plural
- Columns: `snake_case`
- Indexes: `idx_<table>_<column>`
- Migrations: `NNNN_description.sql`

## Dependency Injection

Manual constructor-based DI:
1. Repositories first (depend only on database pool)
2. Services next (depend on repositories)
3. Handlers last (depend on services)

## Code Patterns

### Handler Pattern
- Parse and validate request body
- Call service layer
- Return formatted JSON response
- Use helper functions for error mapping

### Repository Pattern
- Accept `context.Context` as first parameter
- Use positional parameters (`$1`, `$2`) for queries
- Wrap errors with context using `%w`
- Return sentinel errors for expected failures

### Channel/Strategy Pattern
- Define small interface with 2-4 methods
- Register implementations in a thread-safe registry
- Each implementation lives in its own subpackage

## Rules

1. Never use ORM - raw SQL only
2. Always scope queries to tenant/account
3. Always wrap errors with context
4. Always pass `context.Context` as first parameter
5. Always use `defer` for cleanup
6. Never export more than necessary
7. Keep interfaces small and consumer-defined

## Workflow Commands

### Full lint + format + tidy
```bash
cd backend && golangci-lint run ./... && gofmt -w . && go mod tidy
```

### Quick pre-commit check
```bash
cd backend && go vet ./... && gofmt -d .
```

### Generate Swagger docs
```bash
cd backend && swag init -g main.go -o docs --parseInternal --useStructName
```

### Build for production
```bash
cd backend && make build
```
