# elodesk

Chatwoot-compatible messaging hub. Supports multiple conversation channels — including **Channel::Email** (IMAP/SMTP + OAuth Gmail/Microsoft 365), **Channel::WhatsApp**, **Channel::Instagram** (DMs via Meta Graph API), **Channel::FacebookPage** (Messenger), **Channel::Sms** (Twilio, Bandwidth, Zenvia), **Channel::Telegram**, **Channel::WebWidget** (embeddable site chat), and the generic **Channel::Api** — and surfaces them to human agents through a Nuxt frontend.

## Stack

| Layer      | Tech                                                      |
|------------|-----------------------------------------------------------|
| Backend    | Go 1.22 · Fiber v2 · pgx v5 · asynq · gorilla/websocket   |
| Storage    | PostgreSQL 16 · Redis 7 · MinIO                           |
| Frontend   | Nuxt 4 · Pinia · `@vueuse/useWebSocket` · Nuxt UI         |
| Auth       | JWT (HS256 access 15m) · refresh rotation · Argon2id      |
| Secrets    | AES-256-GCM at rest (BACKEND_KEK) · SHA-256 lookups       |

## Layout

```
elodesk/
├── backend/        Go service (cmd/backend, internal/**, migrations/*.sql)
├── frontend/       Nuxt 4 app (app/, server/, i18n/)
├── widget/         Vue 3 embeddable chat widget (Vite library mode)
├── openspec/       Active and archived change proposals
├── docker-compose.yml   Postgres + Redis + MinIO for local dev
└── .github/workflows/   CI (go test -race, golangci-lint, pnpm lint/typecheck/test)
```

## Quickstart (local dev)

```bash
# 1. infra
docker compose up -d

# 2. backend
cd backend
cp .env.example .env            # fill JWT_SECRET and BACKEND_KEK
make dev                         # http://localhost:3001

# 3. frontend
cd ../frontend
pnpm install
pnpm dev                         # http://localhost:3000
```

## Env (backend)

| Variable         | Notes                                                            |
|------------------|------------------------------------------------------------------|
| `DATABASE_URL`   | `postgres://user:pass@host:5432/db?sslmode=disable`              |
| `REDIS_URL`      | `host:port` (asynq + cache)                                      |
| `JWT_SECRET`     | ≥32 chars; signs access tokens (HS256, 15m)                      |
| `JWT_ACCESS_TTL` | Go duration, default `15m`                                       |
| `JWT_REFRESH_TTL`| Go duration, default `720h`                                      |
| `BACKEND_KEK`    | base64 ≥32 bytes; AES-256-GCM key for `hmac_token` at rest       |
| `MINIO_*`        | endpoint/port/access/secret/bucket/use_ssl                       |
| `API_URL`        | public origin (swagger)                                          |
| `CORS_ORIGINS`   | comma list or `*`                                                |

## Architecture overview

```
providers (wzap, …)          agents (browser)
        │                           │
        ▼                           ▼
  POST /public/api/v1/…      GET /realtime  (WS, JWT at handshake)
        │                           │
        └── backend (Go) ───────────┘
              │  ├── handlers → services → repos (pgx)
              │  ├── asynq (outbound webhooks, stable X-Delivery-Id)
              │  ├── realtime hub (rooms: account/inbox/conversation)
              │  └── MinIO (attachments, presigned 15m)
              ▼
          Postgres
```

### Security posture

- `api_token`: generated once, handed back in the inbox creation response,
  stored as SHA-256 hash only; authentication is a deterministic hash lookup.
- `hmac_token`: per-channel HMAC key stored as AES-GCM ciphertext, decrypted
  on demand (inbound HMAC middleware, outbound webhook signing).
- Refresh tokens: 48 random bytes, SHA-256 at rest, rotation with family
  revocation on replay.
- Realtime joins (`join.account|inbox|conversation`) validate membership
  against `account_users`; cross-tenant ids fail closed.
- Upload presigned URLs: `PUT` path must begin with `{accountId}/`; `GET`
  requires the attachment to belong to the authenticated account.

## Active change

See `openspec/changes/rewrite-backend-in-go/` for the proposal/design/specs
that guided this repo. Security + correctness follow-ups from the review are
tracked there.

## Related

- [wzap](https://github.com/felipyfgs/wzap) — WhatsApp engine that speaks the
  Channel::Api contract this hub exposes.
