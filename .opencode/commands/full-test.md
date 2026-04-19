---
description: Run all tests across the entire project — backend (Go) and frontend (Nuxt) — with race detection and coverage
---

Run the full test suite across backend and frontend, reporting results and coverage.

**What this command does:**

1. **Backend tests** — runs Go tests with race detector and coverage
2. **Frontend tests** — runs Nuxt test suite
3. **Summary** — combined pass/fail report

**Steps**

1. **Backend tests with race detector and coverage**
   ```bash
   cd backend && go test -race -coverprofile=coverage.out ./...
   ```
   Then show coverage summary:
   ```bash
   cd backend && go tool cover -func=coverage.out | tail -1
   ```
   Report:
   - Total packages tested
   - Pass/fail per package
   - Overall coverage percentage
   - Any race conditions detected

2. **Backend lint check** (catches issues tests might miss)
   ```bash
   cd backend && golangci-lint run ./...
   ```
   Report any lint errors.

3. **Frontend tests**
   ```bash
   cd frontend && pnpm test
   ```
   Report pass/fail.

4. **Frontend typecheck**
   ```bash
   cd frontend && pnpm typecheck
   ```
   Report any type errors.

5. **Frontend lint**
   ```bash
   cd frontend && pnpm lint
   ```
   Report any lint errors.

**Output**

```
## Full Test Suite Results

### Backend (Go)
| Check       | Status  | Details                    |
|-------------|---------|----------------------------|
| Tests       | pass/fail | X packages, Y% coverage  |
| Race        | clean/detected | —                   |
| Lint        | pass/fail | N issues                 |

### Frontend (Nuxt)
| Check       | Status  | Details                    |
|-------------|---------|----------------------------|
| Tests       | pass/fail | —                        |
| Typecheck   | pass/fail | N errors                 |
| Lint        | pass/fail | N errors                 |

### Overall: PASS / FAIL
```

If any check fails, list the specific errors with file:line references.
