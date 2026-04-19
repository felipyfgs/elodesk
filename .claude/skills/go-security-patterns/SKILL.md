---
name: go-security-patterns
description: Use when handling auth, secrets, tokens or multi-tenant queries — JWT HS256 + refresh token rotation with family revoke, Argon2id passwords, AES-256-GCM via crypto/kek.go, HMAC webhook verify, RBAC (Owner=2/Admin=1/Agent=0), account_id scoping in every repo query, json:"-" on sensitive struct fields.
license: MIT
---

## Purpose

Apply security best practices across Go backend code.

## Authentication

### JWT Access Tokens
- Use HS256 signing method
- Short TTL (e.g., 15 minutes)
- Include minimal claims: subject, email, name, expiry, issued-at

### Refresh Token Rotation
- Hash tokens with SHA-256 before storage (never store raw tokens)
- Use token families for replay detection
- On refresh: revoke old token BEFORE generating new one
- If a revoked token is reused, revoke the entire family
- Abort rotation if old token cannot be revoked

## Password Hashing

Use Argon2id with default parameters:
```go
hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
match, err := argon2id.ComparePasswordAndHash(password, hash)
```

Never use bcrypt, MD5, SHA1, or plain SHA256 for passwords.

## Encryption

Use AES-256-GCM for reversible encryption of sensitive fields:
- Key must be at least 32 bytes
- Key should be base64-encoded in environment
- Validate key length at startup

## Token Storage Strategy

| Token Type | Storage Method |
|------------|---------------|
| API tokens | SHA-256 hash (irreversible) |
| HMAC secrets | AES-256-GCM encrypted (reversible) |
| Passwords | Argon2id hash (irreversible) |
| Refresh tokens | SHA-256 hash (irreversible) |

## HMAC Webhook Verification

Verify webhook signatures using HMAC-SHA256:
- Compute HMAC of request body with shared secret
- Use constant-time comparison (`hmac.Equal`)
- Reject requests with invalid signatures

## Authorization

### JWT Middleware
- Extract Bearer token from Authorization header
- Validate token and extract user info
- Store user in request locals

### RBAC Middleware
- Check user role against required roles for route
- Return 401 if not authenticated
- Return 403 if insufficient permissions

### API Token Middleware
- Extract token from query parameter
- Hash and lookup in database
- Associate request with the channel/inbox

## Tenant Isolation

**All database queries must include tenant/account scope:**
```sql
SELECT ... FROM table WHERE account_id = $1 AND ...
```

Never query without tenant scoping in multi-tenant tables.

## Sensitive Fields

Mark sensitive struct fields with `json:"-"` to prevent serialization:
- Password hashes
- Token values
- Encryption keys
- HMAC secrets

## Rules

1. Never store tokens in plain text
2. Never log sensitive data
3. Always rotate refresh tokens with replay detection
4. Always scope queries to tenant
5. Validate secret key lengths at startup
6. Verify HMAC on all inbound webhooks
7. Rate limit public endpoints
8. Use RBAC on all protected routes
9. Never expose internal error details to clients

## Workflow Commands

### Audit sensitive fields without json:"-
```bash
cd backend && grep -rn 'Password\|Token\|Secret\|Key' internal/model/ | grep -v 'json:"-"' | grep -v '_test.go'
```

### Check queries without tenant scoping
```bash
cd backend && grep -rn 'FROM.*WHERE' internal/repo/ | grep -v "account_id" | grep -v "_test.go" | grep -v "schema_migrations"
```

### Check for logged sensitive data
```bash
cd backend && grep -rn 'logger\.\(Info\|Warn\|Error\)' internal/ | grep -iE 'password|token|secret|key' | grep -v "_test.go" | grep -v "REDACTED"
```

### Verify correct password hashing
```bash
cd backend && grep -rn 'bcrypt\|md5\|sha1\|sha256.*password' internal/ | grep -v "_test.go"
```
