---
name: go-testing-patterns
description: Use when writing or editing *_test.go files — Test<Feature>_<Scenario> naming, table-driven subtests with t.Run, t.Helper/t.Cleanup in helpers, miniredis + httptest mocks (never require real infra), -race flag, export_test.go for unexported testing. Forbid var _ = ... suppressors.
license: MIT
---

## Purpose

Apply consistent testing patterns for Go backend code.

## Test Structure

- Co-locate tests with implementation (same package)
- File naming: `*_test.go`
- One test file per source file being tested

## Test Naming

### Test Functions
```
Test<Feature>_<Scenario>
```

Examples:
```go
func TestDedupLock_Acquire_First(t *testing.T)
func TestDedupLock_Acquire_Duplicate(t *testing.T)
func TestVerifySignature_Valid(t *testing.T)
```

### Test Helpers
Always use `t.Helper()` and `t.Cleanup()`:
```go
func setupTest(t *testing.T) *SomeService {
    t.Helper()
    // setup...
    t.Cleanup(func() { /* cleanup */ })
    return service
}
```

## Table-Driven Tests

Use table-driven tests for multiple test cases:
```go
func TestSomeFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "hello", "HELLO", false},
        {"empty input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := someFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Mock Patterns

### Redis Tests
Use `miniredis` for Redis-dependent tests:
```go
redis := miniredis.RunT(t)
client := miniredis.NewClient(redis.Addr())
t.Cleanup(func() { client.Close(); redis.Close() })
```

### HTTP Tests
Use `httptest` for HTTP handlers:
```go
req := httptest.NewRequest("POST", "/path", body)
rec := httptest.NewRecorder()
app.Test(req)
```

## Integration Tests

Skip integration tests with `testing.Short()`:
```go
func TestSomething_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // test requiring real infrastructure
}
```

## Export Test Pattern

For testing unexported functions, create `export_test.go`:
```go
package mypackage

var ExportPrivateFunc = privateFunc
```

## Rules

1. Always use `t.Helper()` in test helper functions
2. Always use `t.Cleanup()` for resource cleanup
3. Prefer table-driven tests for multiple cases
4. Use `t.Run()` for named subtests
5. Use mock services (miniredis, httptest) - never require real infrastructure
6. Always run tests with `-race` flag
7. Never suppress unused import warnings with `var _ = ...`

## Workflow Commands

### Run all tests with race detector
```bash
cd backend && go test -race ./...
```

### Run tests with coverage
```bash
cd backend && go test -race -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
```

### Run only short tests (skip integration)
```bash
cd backend && go test -race -short ./...
```

### Run a specific test
```bash
cd backend && go test -race -run TestName ./path/to/package/...
```

### Check coverage by package
```bash
cd backend && go test -cover ./...
```

### Find tests that always skip
```bash
cd backend && grep -rn 't.Skip(' internal/*_test.go | grep -v "testing.Short"
```
