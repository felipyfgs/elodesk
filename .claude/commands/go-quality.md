---
name: "go-quality"
description: Quality review Go backend — lint, format, test -race, wrap errors (%w), structured logs (.Str component), json:"-" on secrets, tenant scoping, dead code. Auto-fixes everything possible.
allowed-tools: Bash(cd backend:*), Bash(gofmt:*), Bash(go:*), Bash(golangci-lint:*), Bash(swag:*), Bash(make:*), Grep, Glob, Read, Edit
---

Run a comprehensive quality review on the Go backend. This checks code health across all standards and **automatically fixes every issue it can**.

**What this command does:**

1. **Lint** — runs golangci-lint, reports errors
2. **Format** — applies gofmt and go mod tidy automatically
3. **Tests** — runs all tests with the race detector
4. **Error wrapping** — finds and fixes `%v` → `%w` in error wrapping
5. **Error logging** — finds and adds missing logger calls before 500 responses
6. **Structured logging** — finds and adds missing `component` field to log calls
7. **Sensitive fields** — finds and adds `json:"-"` to password/token/secret fields
8. **Tenant scoping** — flags queries missing `account_id` for manual review
9. **Code smells** — removes suppressed imports (`var _ = ...`) in production code

**Steps**

1. **Format and tidy** (auto-fix)
   ```bash
   cd backend && gofmt -w . && go mod tidy
   ```
   Report which files were changed.

2. **Lint the codebase**
   ```bash
   cd backend && golangci-lint run ./...
   ```
   If fixable errors are found, apply fixes automatically. Report remaining unfixable errors.

3. **Run tests with race detector**
   ```bash
   cd backend && go test -race ./...
   ```
   Report any test failures.

4. **Fix error wrapping — replace `%v` with `%w`**
   Scan for `fmt.Errorf("...%v...", err)` in production code and change to `%w` so errors are properly wrapped for `errors.Is` checks.
   ```bash
   cd backend && grep -rn 'fmt.Errorf.*%v.*err' internal/ | grep -v "_test.go"
   ```
   For each match, edit the file to use `%w` instead of `%v`.

5. **Fix unlogged 500 responses**
   Find handler code that returns 500 without logging the error first. Add a `logger.Error()` call before the response.
   ```bash
   cd backend && grep -B5 "StatusInternalServerError" internal/handler/*.go | grep -v "logger"
   ```
   For each match, add `logger.Error().Str("component", "<relevant>").Err(err).Msg("<description>")` before the 500 return.

6. **Fix logs missing component field**
   Find log calls without `.Str("component", "...")` and add the appropriate component identifier.
   ```bash
   cd backend && grep -rn 'logger\.\(Info\|Warn\|Error\|Debug\|Fatal\)()' internal/ | grep -v "_test.go" | grep -v 'Str("component"'
   ```
   For each match, determine the correct component from the file context (auth, db, server, webhook, channel, redis, media, realtime) and add `.Str("component", "<name>")`.

7. **Fix sensitive fields missing json:"-**
   Find struct fields containing Password, Token, Secret, or Key that are missing `json:"-"` tag.
   ```bash
   cd backend && grep -rn 'Password\|Token\|Secret\|Key' internal/model/ | grep -v 'json:"-"' | grep -v '_test.go'
   ```
   For each match, add `json:"-"` tag to prevent serialization.

8. **Flag queries without tenant scoping** (manual review)
   Find database queries that don't filter by `account_id`.
   ```bash
   cd backend && grep -rn 'FROM.*WHERE' internal/repo/ | grep -v "account_id" | grep -v "_test.go" | grep -v "schema_migrations"
   ```
   Report these for manual review — do not auto-fix as this requires understanding the query intent.

9. **Remove suppressed imports**
   Find and remove `var _ = ...` lines in production code that suppress unused import warnings.
   ```bash
   cd backend && grep -rn 'var _ = ' internal/ | grep -v "_test.go"
   ```
   For each match, remove the suppression line and the unused import.

**Output**

Summarize all actions taken in a clear report:

```
## Go Backend Quality Review — Auto-Fix Applied

| Check          | Status  | Fixed | Remaining |
|----------------|---------|-------|-----------|
| Format         | applied | N     | —         |
| Lint           | pass/fail | N   | N         |
| Tests          | pass/fail | —   | N         |
| Error wrapping | fixed   | N     | 0         |
| Error logging  | fixed   | N     | 0         |
| Structured log | fixed   | N     | 0         |
| Sensitive data | fixed   | N     | 0         |
| Tenant scoping | flagged | —     | N (review)|
| Code smells    | fixed   | N     | 0         |
```

List every file that was modified with a brief description of what was fixed. Flag any issues that require manual review with `file:line` references and explanation.

If everything is clean, report: "All quality checks passed — code is clean."
