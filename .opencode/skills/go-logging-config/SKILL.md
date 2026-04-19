---
name: go-logging-config
description: Go logging and configuration standards - structured logging, env validation and config patterns
license: MIT
---

## Purpose

Apply consistent logging and configuration patterns across Go backend code.

## Logging

### Structured Logging

Use a structured logger with consistent fields:
- Always include a `component` field identifying the subsystem
- Use `.Err(err)` for error logging to capture stack traces
- Never log sensitive data (passwords, tokens, secrets)

### Log Levels

- **Info**: Normal operational events (server started, request processed)
- **Warn**: Unexpected but non-critical situations (token expiring, retry attempts)
- **Error**: Failures that need attention (query failed, external service error)
- **Debug**: Detailed information for troubleshooting
- **Fatal**: Unrecoverable errors (config validation failure)

### Log Format

- Development: Human-readable console output with timestamps
- Production: JSON format for log aggregation systems
- Output to stderr in both modes

### Sensitive Data Redaction

Implement a header redaction function that masks known sensitive headers before logging.

## Configuration

### Config Struct

Define a single config struct with all application settings. Group related fields together.

### Loading Pattern

1. Load `.env` file (optional, silent if missing)
2. Read environment variables with sensible defaults
3. Validate required fields
4. Fail fast on validation errors

### Required Variables

Validate critical variables at startup:
- Database connection string
- Cache/connection URLs
- Secret keys (minimum length checks)
- Encryption keys (valid encoding, minimum length)

### Helper Functions

Provide helpers for common env var patterns:
- `getEnv(key, fallback)` - string with default
- `getEnvAsBool(key, fallback)` - boolean parsing
- `getEnvAsInt(key, fallback)` - integer parsing
- `mustDuration(key, fallback)` - duration parsing (exit on failure)

## Rules

1. Always include `component` field in all log entries
2. Never log passwords, tokens, or encryption keys
3. Use `.Err(err)` for error logging
4. Fail fast on config validation - exit with clear error message
5. `.env` file is optional - provide sensible defaults
6. Validate TTL durations at parse time - never silently use zero

## Workflow Commands

### Check for logs missing component field
```bash
cd backend && grep -rn 'logger\.\(Info\|Warn\|Error\|Debug\|Fatal\)()' internal/ | grep -v "_test.go" | grep -v 'Str("component"'
```

### Check for error logs without .Err()
```bash
cd backend && grep -rn 'logger.Error()' internal/ | grep -v "_test.go" | grep -v '\.Err('
```

### Check for sensitive data in logs
```bash
cd backend && grep -rn 'password\|token\|secret' internal/ | grep -i "log\|print" | grep -v "_test.go" | grep -v "REDACTED"
```

### Verify required env vars in .env.example
```bash
cd backend && for var in DATABASE_URL REDIS_URL JWT_SECRET ENCRYPTION_KEY; do grep -q "$var" .env.example && echo "OK: $var" || echo "MISSING: $var"; done
```
