---
name: go-error-handling
description: Go error handling standards - sentinel errors, wrapping, HTTP mapping and validation helpers
license: MIT
---

## Purpose

Apply consistent error handling patterns across Go backend code.

## Error Hierarchy

Define sentinel errors at each layer:
- **Repository layer**: Domain-specific not-found and conflict errors
- **Service layer**: Business logic errors (invalid credentials, forbidden actions)
- **Handler layer**: HTTP status mapping helpers

## Sentinel Errors

Define as package-level variables with descriptive names:
```go
var ErrNotFound = errors.New("resource not found")
```

Provide a centralized `IsErrNotFound(err error) bool` helper that checks all sentinel errors.

## Error Wrapping

Always wrap errors with context using `%w`:
```go
return fmt.Errorf("failed to query resource: %w", err)
```

For sentinel errors with context:
```go
return fmt.Errorf("%w: id=%d", ErrNotFound, id)
```

Never use `%v` or `%s` when the error needs to be checked with `errors.Is`.

## Handler Error Helpers

### Parse and Validate
- Parse request body into DTO
- Validate using struct tags
- Return appropriate HTTP 400 response
- Use a sentinel `errResponseSent` to signal response already written

### Service Error Mapping
- Map sentinel errors to appropriate HTTP status codes
- Never expose internal error details to clients
- Log unhandled errors before returning 500
- Use a switch statement with `errors.Is` for clean mapping

### Not Found Helper
- Check if error matches any not-found sentinel
- Return 404 for not-found, 500 for unexpected errors
- Log the original error on 500 responses

## Response Patterns

### Error Response
```json
{"success": false, "error": "Not Found", "message": "resource not found"}
```

### Success Response
```json
{"success": true, "payload": {...}}
```

## Rules

1. Never return raw errors from repository - always wrap with context
2. Never expose internal details to clients
3. Always log unhandled errors before returning 500
4. Use sentinel errors for expected failures, `fmt.Errorf` for unexpected
5. Service layer returns errors, handler layer logs them
6. Use `%w` for wrapping, never `%v` when `errors.Is` is needed

## Workflow Commands

### Check for unwrapped errors
```bash
cd backend && grep -rn "return.*err" internal/handler/ | grep -v "fmt.Errorf" | grep -v "return nil"
```

### Verify error wrapping uses %w
```bash
cd backend && grep -rn 'fmt.Errorf.*%v.*err' internal/ | grep -v "_test.go"
```

### Check for unlogged 500 responses
```bash
cd backend && grep -B5 "StatusInternalServerError" internal/handler/*.go | grep -v "logger"
```

### Static analysis for unhandled errors
```bash
cd backend && golangci-lint run --enable=errcheck ./...
```
