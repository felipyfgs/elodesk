# elodesk backend

Go service behind the elodesk hub. Speaks the Chatwoot `Channel::Api`
contract so providers (wzap, Evolution API, Meta Cloud, …) plug in without
changes, and exposes a JWT + WebSocket API for the agent frontend.

## Layout

```
backend/
├── cmd/backend/main.go           entrypoint
├── internal/
│   ├── config/                   env loading + validation (fatal on bad TTL/KEK)
│   ├── crypto/                   AES-256-GCM (KEK) + SHA-256 HashLookup
│   ├── database/                 pgx v5 pool + embedded migrations
│   ├── dto/                      request/response shapes (validator tags)
│   ├── handler/                  Fiber handlers (parseAndValidate → service)
│   ├── logger/                   zerolog singleton (WithComponent + redact)
│   ├── media/                    MinIO client + upload helpers
│   ├── middleware/               jwt, api_token, hmac, org_scope, roles
│   ├── model/                    domain structs (json:"-" on secrets)
│   ├── realtime/                 WS hub + MembershipChecker
│   ├── repo/                     pgx scanners, tenant-scoped queries
│   ├── server/                   router wiring + NotFoundHandler
│   ├── service/                  business logic (auth, inbox, conversation, …)
│   └── webhook/                  outbound asynq processor
├── migrations/*.sql              // embedded via //go:embed
├── Dockerfile                    // multi-stage, CGO_ENABLED=0
└── Makefile                      // dev build test lint docs tidy
```

## Dev

```bash
cp .env.example .env              # fill JWT_SECRET and BACKEND_KEK
# Infra (Postgres + Redis + MinIO) — from the elodesk root:
docker compose -f ../docker-compose.yml up -d

make dev                           # go run ./cmd/backend
make build                         # bin/backend
make test                          # go test -race ./...
make lint                          # golangci-lint run ./...
make docs                          # swag init → docs/swagger.yaml
```

## Env

Validated at boot (`internal/config`):

- `DATABASE_URL`, `REDIS_URL` — required
- `JWT_SECRET` — required, ≥32 chars
- `JWT_ACCESS_TTL` (default `15m`), `JWT_REFRESH_TTL` (default `720h`)
- `BACKEND_KEK` — required, base64 decoding to ≥32 bytes
- `MINIO_ENDPOINT`/`MINIO_PORT`/`MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY`/`MINIO_BUCKET`/`MINIO_USE_SSL`
- `PORT` (default `3001`), `SERVER_HOST` (default `0.0.0.0`)
- `API_URL`, `CORS_ORIGINS`, `LOG_LEVEL`, `GO_ENV`

## HTTP surface

- `GET /health` — overall + db + redis (503 on degraded)
- `GET /docs/` — Swagger UI
- `GET /realtime` — WS upgrade, JWT via `?token=` or `Sec-WebSocket-Protocol`
- `POST /api/v1/auth/{register,login,refresh,logout}`
- `*  /api/v1/accounts/:aid/…` — JWT + OrgScope + Roles
- `*  /public/api/v1/inboxes/:identifier/…` — `api_access_token` + optional HMAC

## Security

| Field              | At rest                       | Used for                       |
|--------------------|-------------------------------|--------------------------------|
| `users.password_hash` | Argon2id                   | login                          |
| `refresh_tokens`   | SHA-256 hex                   | refresh rotation + family revoke |
| `channels_api.api_token_hash` | SHA-256 hex        | provider auth lookup           |
| `channels_api.hmac_token`     | AES-256-GCM ciphertext | inbound verify + outbound sign |
| outbound webhook payload in Redis | hmac key = ciphertext | plaintext never in Redis       |

Plaintext `api_token` / `hmac_token` are returned **once** in the inbox
creation response (`POST /api/v1/accounts/:aid/inboxes`) and never again.

## Migrations

`migrations/*.sql` are embedded via `//go:embed` and applied in filename
order at startup. `schema_migrations` tracks applied versions. An advisory
lock serialises concurrent startups. Rerunning is idempotent.

## Known gaps

The archived change `rewrite-backend-in-go` ran before an automated-test
suite existed. The following items were marked done but not actually exercised
by tests and remain worthwhile follow-ups:

- `go test -race ./...` has no tests today. CI runs it but no suite exists.
- Integration tests against ephemeral Postgres/Redis/MinIO for: auth register/login/refresh-rotation/replay, org scope (cross-tenant), Channel::Api idempotency, outbound webhook signing + delivery id stability, realtime membership (cross-tenant joins denied), upload path ownership.
- Golden smoke path: register → create inbox → wzap posts contact/message via Channel::Api → agent responds → outbound webhook HMAC matches → event reaches provider.

Open them as a new OpenSpec change when ready (`/opsx:propose`).

## Observability

- Structured logs (`zerolog`) with `component=` on every line; sensitive
  headers are redacted via `logger.redactHeaders`.
- Outbound webhooks log delivery id + retry count + status.
- `/health` drives liveness/readiness probes.
