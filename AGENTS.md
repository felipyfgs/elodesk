# AGENTS.md — Elodesk

Hub de mensagens multi-canal. Go backend + Nuxt 4 frontend.

## Idioma

**Sempre interaja com o usuário em português do Brasil (pt-BR).** Código, comentários, commits e docs técnicas em inglês.

## Commands

### Backend (`backend/`)
```
make dev       # go run (migrations auto-run on startup)
make build     # compile to bin/backend
make test      # go test -race ./...
make lint      # golangci-lint run ./...
make docs      # swag init -g main.go -o docs --parseInternal --useStructName -d cmd/backend,internal
make tidy      # go mod tidy
make clean     # remove bin/
make seed      # stub (not implemented)
make install-tools  # golangci-lint + swag
```

### Frontend (`frontend/`)
```
pnpm dev         # nuxt dev (port 3000)
pnpm build       # production build
pnpm preview     # preview production build
pnpm lint        # eslint .
pnpm typecheck   # nuxt typecheck
pnpm test        # no-op — "(frontend tests TBD)"
```

### Infra
```
docker compose up -d   # 6 services: Postgres 16, Redis 7, MinIO, backend, worker, frontend
```
- Postgres: user=`wzap`, password=`wzap`, db=`wzap`, port **5432**
- Redis: port **6379**
- MinIO: user=`minio`, password=`minio12345`, API port **9010**, console port **9011** (not default 9000/9001)
- Backend: port **3001**, hot-reload via `.air.toml`
- Worker: asynq worker, hot-reload via `.air.worker.toml`
- Frontend: port **3000**, SPA (`ssr: false`)

### CI
Not versioned in repo. No `.github/workflows/`.

## Architecture

### Backend (`backend/internal/`)
- **Module name**: `backend` — import as `backend/internal/...`
- **Entrypoints**: `cmd/backend/main.go` (HTTP server, 3001) + `cmd/worker/main.go` (asynq worker)
- **Layers**: `handler/` → `service/` → `repo/` (pgx)
- **DI + routes**: `server/router.go` (single source of truth, ~600 lines)
- **Channels**: `channel/` registry pattern (`Kind → Channel` interface: `Kind`, `HandleInbound`, `SendOutbound`, `SyncTemplates`)
  - **11 kinds**: `Api`, `Whatsapp` (Cloud API/Dialog360), `Sms` (Bandwidth/Zenvia/Twilio legacy), `Instagram`, `FacebookPage`, `Telegram`, `WebWidget` (SSE), `Line`, `Tiktok`, `Twilio` (dual SMS/WhatsApp), `Twitter`
  - Multi-provider channels use sub-registry: WhatsApp (2 providers), SMS (3 providers — Bandwidth, Twilio, Zenvia)
  - `channel/meta/` — shared Meta (Facebook/Instagram) logic
  - Feature-flagged kinds: `Tiktok`, `Twitter`, `Twilio` medium selection (`FEATURE_CHANNEL_TIKTOK`, `FEATURE_CHANNEL_TWITTER`, `FEATURE_CHANNEL_TWILIO_WHATSAPP`, `FEATURE_TWILIO_SMS_MEDIUM`)
  - **Email** (`channel/email/`) exists as a separate package but is NOT a registered channel Kind and has NO routes in `router.go`. Do not treat as an active channel.
- **Middleware**: JWT auth, org scope (`X-Account-Id`), RBAC (Owner=2, Admin=1, Agent=0), api_token SHA-256 lookup, HMAC, widget CORS + rate limit
- **Realtime**: WebSocket hub (`realtime/`) — single goroutine, rooms `account:N`/`inbox:N`/`conversation:N`, membership fail-closed
- **Webhooks**: `webhook/outbound_processor.go` — asynq queue, HMAC signing, 5 retries (1s, 5s, 30s, 2m, 10m), 5xx retry / 4xx dead-letter
- **Crypto**: `crypto/kek.go` — AES-256-GCM cipher + SHA-256 hash
- **DB**: `database/migrations.go` — `go:embed` + advisory lock, forward-only, no rollback. 45 SQL migrations (`0001_init.sql` – `0046_pipelines.sql`); **migration 0018 is intentionally skipped**.

### Frontend (`frontend/app/`)
- **SSR disabled** (`ssr: false` in nuxt.config.ts) — SPA mode
- **Composables** (12): `useApi`, `useAttachmentSrc`, `useAuth`, `useContactSearch`, `useConversationFilters`, `useConversationRealtime`, `useDashboard`, `useDetailsSidebar`, `useErrorHandler`, `useFilterAttributes`, `useRealtime`, `useResponsive`
- **Stores** (20 Pinia stores): `accounts`, `agents`, `audioPlayer`, `auth`, `cannedResponses`, `contacts`, `conversations`, `customAttributes`, `inboxes`, `labels`, `macros`, `messages`, `notes`, `notifications`, `pipelineCards`, `pipelines`, `savedFilters`, `sla`, `teams`, `webhooks`
- **Validation**: Zod schemas in `app/schemas/` — multi-step wizard forms split into per-step schemas
- **i18n**: pt-BR + en via `@nuxtjs/i18n` (`no_prefix` strategy, default `pt-BR`)
- **UI**: `@nuxt/ui` v4 + Tailwind CSS v4 — all UI primitives sourced from Nuxt UI (no custom wrappers unless adding domain behavior)
- **Reports**: `app/components/reports/` — Nuxt UI components + Unovis charts (`VisArea`, `VisLine`, `VisGroupedBar`, `VisTooltip`, `VisCrosshair`)

## Domain Model

- **Account** → top-level tenant (multi-tenant)
- **Inbox** → central abstraction, `channel_type` + channel-specific record, one per account
- **ContactInbox** → bridges `Contact` to `Inbox` via `source_id` (channel-specific identifier)
- **Conversation** → belongs to `ContactInbox` (not directly to `Contact`), has `display_id` (sequential per account)
- **Message** → belongs to `Conversation`, `sender_type`/`sender_id` (polymorphic: User or Contact)
- **Pipeline** → kanban board per account; **PipelineStage** → column; **PipelineCard** → card linking a Conversation (migration 0046)
- **JSONB columns**: `additional_attributes`, `custom_attributes`, `content_attributes`, `provider_config`
- **Status**: Open(0), Resolved(1), Pending(2), Snoozed(3)
- **Message types**: Incoming(0), Outgoing(1), Activity(2), Template(3)

## Critical Gotchas

- **Migrations run on startup** — no manual `migrate`. Fail = fatal exit. Forward-only, no rollback.
- **Migration 0018 is intentionally skipped** — gap between 0017 and 0019. Do not fill it.
- **MinIO ports**: `9010:9000` and `9011:9001` (not default 9000/9001). Credentials: `minio`/`minio12345`.
- **docker-compose Postgres**: user=`wzap`, password=`wzap`, db=`wzap`
- **Backend `.env`**: `JWT_SECRET` (≥32 chars), `BACKEND_KEK` (base64 ≥32 bytes). `openssl rand -base64 32` for KEK.
- **Go 1.25** — `go.mod` uses toolchain go1.25.0.
- **Makefile is in `backend/`**, not at repo root.
- **Home page routing**: `/` redirects authenticated users to `/accounts/{primaryId}` (home dashboard), not conversations. The home page lives at `/accounts/[accountId]/index.vue`.
- **Email channel code exists but is not wired** — `channel/email/` has handler and service code but no routes in `router.go`. Not an active channel kind.
- **Two binaries**: `cmd/backend/` (HTTP server) and `cmd/worker/` (asynq worker). The worker is a separate Docker service with its own air config (`.air.worker.toml`).
- **CI not versioned** — no `.github/workflows/` in repo. Local quality checks via `/go-quality`, `/frontend-quality`, `/full-test`.

## Key Flows

### Auth
JWT access (HS256, 15m) + refresh tokens (48 random bytes, SHA-256 at rest). Rotation with family revocation on replay.
- Register: Argon2id → User + Account + AccountUser (Owner) in tx → token pair
- Login: email lookup → Argon2id compare → resolve primary account → token pair
- Refresh: SHA-256 hash → lookup → if revoked, revoke family → revoke current → new pair
- Frontend `useApi()` intercepts 401 → `/auth/refresh` (deduplicated) → retry → redirect `/login` on failure

### Channel Creation
- `Channel::Api`: `POST /api/v1/accounts/:aid/inboxes` → 3 random 48-byte tokens (identifier, api_token, hmac_token) → api_token stored as SHA-256, hmac_token as AES-GCM ciphertext → plaintext returned ONCE
- Channel-specific: own provision endpoints (e.g. `POST /inboxes/telegram`)
- Public API auth: `api_access_token` header → SHA-256 → lookup in `channels_api`

### Realtime
- `GET /realtime` with JWT in query or `Sec-WebSocket-Protocol`
- Client joins: `join.account|inbox|conversation` with `{id: ...}`
- Ping every 54s, pong timeout 60s. Frontend: 30s heartbeat, 10 retries.
- Event names follow `resource.action` (see `backend/internal/realtime/events.go`):
  - `message.created`, `message.updated`, `message.deleted`
  - `conversation.created`, `conversation.updated`, `conversation.deleted`
- Message events are emitted from a **single** point: `service.MessageService`. Channels (WhatsApp, SMS, …) delegate message creation to `MessageService` and never broadcast directly.
- `message.created` / `message.updated` payloads embed a `conversation` summary (`assigneeId`, `teamId`, `unreadCount`, `lastActivityAt`) and echo `echoId` (when sent in `POST /messages`) for optimistic reconciliation in the composer.

### Uploads (MinIO)
- Presigned PUT/GET (15m expiry). PUT path must begin with `{accountId}/`.

## Style

Code-style rules are maintained in skills loaded on demand (`.opencode/skills/`):
- Go: `go-backend`, `go-errors`, `go-logging`, `go-security`, `go-testing`, `go-migrations`
- Frontend: `nuxt-frontend`
- Companion file: `CLAUDE.md` at repo root supplements this file with additional context.

## OpenSpec Workflow

Commands in `.opencode/commands/`:
- `/opsx:propose` — create change + artifacts
- `/opsx:explore` — think mode (no code changes)
- `/opsx:apply` — implement tasks
- `/opsx:archive` — archive to `openspec/changes/archive/YYYY-MM-DD-<name>/`
- `/go-quality` — backend quality review + auto-fix
- `/frontend-quality` — frontend quality review + auto-fix
- `/full-test` — run all tests
- `/dev-setup` — environment setup + health checks

## API Routes

| Prefix | Auth | Purpose |
|--------|------|---------|
| `GET /health`, `GET /docs/*` | none | Health, Swagger |
| `POST /api/v1/auth/*` | none | Register, login, refresh, logout, forgot, reset, mfa, invitations/:token/accept |
| `GET /realtime` | JWT (query/WS header) | WebSocket (rooms: account, inbox, conversation, user) |
| `PUT /api/v1/users/:id` | JWT (self) | Profile edit + password change |
| `/api/v1/users/:id/notification_preferences` | JWT (self) | GET/PUT user notification preferences |
| `/api/v1/accounts/:aid/*` | JWT + org scope | Inboxes, contacts, conversations, messages, uploads, labels, teams, canned, attributes, filters, agents, macros, slas, webhooks, audit_logs, notifications, reports, pipelines |
| `/public/api/v1/inboxes/:identifier/*` | api_token (SHA-256) | Contacts, conversations, messages |
| `/webhooks/*` | none | SMS, Instagram, Facebook, Telegram, Line, Tiktok, Twilio, Twitter |
| `/widget/:token`, `/widget/:token/ws` | CORS + rate limit | SSE widget |
| `/api/v1/widget/*` | widget auth | Sessions, messages, identify, attachments |

## Background jobs

The backend runs three in-process ticker-based jobs alongside the HTTP server:
- **SLA breach detection** — every 60s scans conversations past their `sla_*_due_at`, flags `sla_breached=true`, emits `sla.breached`, persists notification + audit log.
- **Audit retention** — every 24h deletes `audit_logs` older than 90 days.
- **Twilio content templates sync** — every 24h refreshes `channels_twilio.content_templates` for WhatsApp-medium channels (via `/v1/Content` pagination).

### Asynq Worker (`cmd/worker/`)

Separate binary, runs as its own Docker Compose service. Uses `.air.worker.toml` for hot-reload.
Queues: `webhook:outbound` (5 retries, HMAC signed), `channel:wa:send` (WhatsApp outbound via `WaSendProcessor`).

### Outbound Webhook Notifier

`OutboundWebhookNotifier` is wired into `MessageService` — when a non-private outgoing message is created in a `Channel::Api` inbox, it dispatches the `message_created` event to the configured webhook URL via asynq.

## Channels: Api, Line, Tiktok, Twilio, Twitter

| Kind | Provisioning | Public webhook | Envs / flags |
|------|--------------|----------------|--------------|
| `Channel::Api` | `POST /api/v1/accounts/:aid/inboxes` (type `api`); `POST /inboxes/:id/rotate_token`; public traffic via `/public/api/v1/inboxes/:identifier/*` (api_token SHA-256) | none (uses `public/api/v1`) | — |
| `Channel::Line` | `POST /api/v1/accounts/:aid/inboxes/line` (channel ID + secret + token persisted via KEK) | `POST /webhooks/line/:line_channel_id` (verifies `X-Line-Signature`) | — |
| `Channel::Tiktok` | OAuth 2.0: `POST /inboxes/tiktok/authorize` → `{url}`; `GET /inboxes/tiktok/oauth/callback` (state nonce in Redis 10m) | `POST /webhooks/tiktok/:business_id` (verifies `Tiktok-Signature: t=…,s=…` HMAC-SHA256 with 5m skew) | `FEATURE_CHANNEL_TIKTOK` (default off), `TIKTOK_CLIENT_KEY`, `TIKTOK_CLIENT_SECRET` |
| `Channel::Twilio` | `POST /api/v1/accounts/:aid/inboxes/twilio` (XOR phone_number/messaging_service_sid; credentials validated via `GET /Accounts/{sid}.json`); `POST /inboxes/:id/twilio_templates` | `POST /webhooks/twilio/:identifier` (message) + `POST /webhooks/twilio/:identifier/status` (verifies `X-Twilio-Signature` HMAC-SHA1) | `FEATURE_CHANNEL_TWILIO_WHATSAPP` (default on), `FEATURE_TWILIO_SMS_MEDIUM` (default off) |
| `Channel::Twitter` | OAuth 1.0a: `POST /inboxes/twitter/authorize` → `{url}`; `GET /inboxes/twitter/oauth/callback` | `GET /webhooks/twitter/:profile_id` (CRC, returns `sha256=<hmac>`); `POST /webhooks/twitter/:profile_id` (verifies `x-twitter-webhooks-signature`); only `direct_message_events` ingested | `FEATURE_CHANNEL_TWITTER` (default off), `TWITTER_CONSUMER_KEY`, `TWITTER_CONSUMER_SECRET` |

**WhatsApp providers**: `Channel::Whatsapp` supports Meta Cloud API (`provider=whatsapp_cloud`) and 360Dialog (`provider=default_360dialog`) via `ProviderForType()` factory. Wzap engine integration uses `Channel::Api` — configure `wz_elodesk` on the Wzap side.

**Twilio coexistence**: the legacy `channel/sms/twilio/` sub-provider remains registered in `sms.Registry` so existing `channel_type=Channel::Sms` inboxes (with `provider=twilio`) keep delivering inbound/outbound. New Twilio SMS inboxes via `POST /inboxes/sms?provider=twilio` return `400 unsupported_provider` — use `POST /inboxes/twilio` with `medium="sms"`.
