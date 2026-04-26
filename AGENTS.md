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
make seed      # stub (not implemented)
make install-tools  # golangci-lint + swag
```

### Frontend (`frontend/`)
```
pnpm dev         # nuxt dev (port 3000)
pnpm build       # production build
pnpm lint        # eslint .
pnpm typecheck   # nuxt typecheck
pnpm test        # no-op — "(frontend tests TBD)"
```

### Infra
```
docker compose up -d   # Postgres 16, Redis 7, MinIO (ports 5432, 6379, 9010/9011)
```

### CI
Push/PR → 2 jobs: `go test -race` + `golangci-lint` (Go 1.25) | `pnpm lint` + `typecheck` + `test` (Node 22, pnpm 10).

## Architecture

### Backend (`backend/internal/`)
- **Module name**: `backend` — import as `backend/internal/...`
- **Entrypoint**: `cmd/backend/main.go`
- **Layers**: `handler/` → `service/` → `repo/` (pgx)
- **DI + routes**: `server/router.go` (single source of truth)
- **Channels**: `channel/` registry pattern (`Kind → Channel` interface: `Kind`, `HandleInbound`, `SendOutbound`, `SyncTemplates`)
  - 12 kinds: `Api`, `Whatsapp` (Cloud API/Dialog360), `Sms` (Bandwidth/Zenvia; legacy Twilio kept for back-compat), `Instagram`, `FacebookPage`, `Telegram`, `WebWidget` (SSE), `Email` (IMAP/SMTP/OAuth), `Line`, `Tiktok`, `Twilio` (dual SMS/WhatsApp), `Twitter`
  - Multi-provider channels use sub-registry (WhatsApp, SMS)
  - `channel/meta/` — shared Meta (Facebook/Instagram) logic
  - Feature-flagged kinds: `Tiktok`, `Twitter`, `Twilio` medium selection (`FEATURE_CHANNEL_TIKTOK`, `FEATURE_CHANNEL_TWITTER`, `FEATURE_CHANNEL_TWILIO_WHATSAPP`, `FEATURE_TWILIO_SMS_MEDIUM`)
- **Middleware**: JWT auth, org scope (`X-Account-Id`), RBAC (Owner=2, Admin=1, Agent=0), api_token SHA-256 lookup, HMAC, widget CORS + rate limit
- **Realtime**: WebSocket hub (`realtime/`) — single goroutine, rooms `account:N`/`inbox:N`/`conversation:N`, membership fail-closed
- **Webhooks**: `webhook/outbound_processor.go` — asynq queue, HMAC signing, 5 retries (1s, 5s, 30s, 2m, 10m), 5xx retry / 4xx dead-letter
- **Crypto**: `crypto/kek.go` — AES-256-GCM cipher + SHA-256 hash
- **DB**: `database/migrations.go` — `go:embed` + advisory lock, forward-only, no rollback

### Frontend (`frontend/app/`)
- **Composables**: `useApi.ts` ($fetch + JWT + auto 401 retry via `/auth/refresh`), `useAuth.ts`, `useRealtime.ts` (WebSocket, rooms, auto-reconnect)
- **Stores**: 11 Pinia stores (`auth`, `accounts`, `inboxes`, `conversations`, `messages`, `labels`, `notes`, `teams`, `cannedResponses`, `customAttributes`, `savedFilters`)
- **Validation**: Zod schemas in `app/schemas/` — multi-step wizard forms split into per-step schemas (`*StepSetup`, `*StepCredentials`, etc.)
- **i18n**: pt-BR + en via `@nuxtjs/i18n`
- **UI**: `@nuxt/ui` v4 + Tailwind CSS v4 — all UI primitives sourced from Nuxt UI (no custom wrappers unless adding domain behavior)
- **UI contract**: `openspec/changes/standardize-frontend-nuxt-ui/specs/frontend-ui-primitives/spec.md` — authoritative source for component choices (`UChat*` for threads, `UStepper` for wizards, `UTimeline` for events, `useToast` for feedback, `useOverlay` for modals, semantic color utilities only)
- **Reports**: `app/components/reports/` — Nuxt UI components (`UPageGrid`, `UCard`, `UUser`, `UEmpty`) + Unovis charts (`VisArea`, `VisLine`, `VisGroupedBar`, `VisTooltip`, `VisCrosshair`) for overview, conversations, CSAT, SLA reports

## Domain Model

- **Account** → top-level tenant (multi-tenant)
- **Inbox** → central abstraction, `channel_type` + channel-specific record, one per account
- **ContactInbox** → bridges `Contact` to `Inbox` via `source_id` (channel-specific identifier)
- **Conversation** → belongs to `ContactInbox` (not directly to `Contact`), has `display_id` (sequential per account)
- **Message** → belongs to `Conversation`, `sender_type`/`sender_id` (polymorphic: User or Contact)
- **JSONB columns**: `additional_attributes`, `custom_attributes`, `content_attributes`, `provider_config`
- **Status**: Open(0), Resolved(1), Pending(2), Snoozed(3)
- **Message types**: Incoming(0), Outgoing(1), Activity(2), Template(3)

## Critical Gotchas

- **Migrations run on startup** — no manual `migrate`. Fail = fatal exit. Forward-only, no rollback.
- **MinIO ports**: `9010:9000` and `9011:9001` (not default 9000/9001)
- **docker-compose defaults**: user=`wzap`, password=`wzap`, db=`wzap`
- **Backend `.env`**: `JWT_SECRET` (≥32 chars), `BACKEND_KEK` (base64 ≥32 bytes). `openssl rand -base64 32` for KEK.
- **`_ = channelRegistry`** in `router.go` — suppress unused var warning. Do not remove. (`outboundWebhookSvc` is now wired via `OutboundWebhookNotifier`.)
- **Go 1.25** — both `go.mod` and CI are aligned on 1.25.
- **Makefile is in `backend/`**, not at repo root.
- **Home page routing**: `/` redirects authenticated users to `/accounts/{primaryId}` (home dashboard with stats), not conversations. The home page lives at `/accounts/[accountId]/index.vue`.

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
- Event names follow `resource.action`: `message.created`, `message.updated`, `message.deleted`, `conversation.created`, `conversation.updated`, `inbox.status`. Legacy `message.new` / `conversation.new` removed (see `backend/internal/realtime/events.go`).
- Message events are emitted from a **single** point: `service.MessageService` (`Create` → `message.created`, `UpdateStatus` → `message.updated`, `SoftDelete` → `message.deleted`). Channels (WhatsApp, SMS, …) delegate message creation to `MessageService` and never broadcast directly.
- `message.created` / `message.updated` payloads embed a `conversation` summary (`assigneeId`, `teamId`, `unreadCount`, `lastActivityAt`) so clients can reorder lists without an extra fetch, and echo `echoId` (when sent in `POST /messages`) for optimistic reconciliation in the composer.

### Uploads (MinIO)
- Presigned PUT/GET (15m expiry). PUT path must begin with `{accountId}/`.

## Style

Code-style rules are maintained in skills (loaded on demand):
- Go: `go-backend`, `go-errors`, `go-logging`, `go-security`, `go-testing`, `go-migrations`
- Frontend: `nuxt-frontend`

## Referência (`_refs/`)

O diretório `_refs/` contém projetos de estudo úteis para consulta. **Sempre consulte `_refs/` antes de buscar na web** quando tiver dúvidas sobre padrões, fluxos ou decisões de arquitetura.

## OpenSpec Workflow

Changes in `openspec/changes/<name>/`. Commands live in `.claude/commands/` (mirrored in `.opencode/commands/`):
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
| `/api/v1/accounts/:aid/*` | JWT + org scope | Inboxes, contacts, conversations, messages, uploads, labels, teams, canned, attributes, filters, agents, macros, slas, webhooks, audit_logs, notifications, reports (overview, conversations, :entity, csat, sla) |
| `/public/api/v1/inboxes/:identifier/*` | api_token (SHA-256) | Contacts, conversations, messages |
| `/webhooks/*` | none | SMS, Instagram, Facebook, Telegram, Line, Tiktok, Twilio, Twitter |
| `/widget/:token`, `/widget/:token/ws` | CORS + rate limit | SSE widget |
| `/api/v1/widget/*` | widget auth | Sessions, messages, identify, attachments |

## Background jobs

The backend runs three in-process ticker-based jobs alongside the HTTP server:

- **SLA breach detection** — every 60s scans conversations past their `sla_*_due_at`, flags `sla_breached=true`, emits `sla.breached` on the account realtime room, persists a notification for the assignee, and records an audit log.
- **Audit retention** — every 24h deletes `audit_logs` older than 90 days.
- **Twilio content templates sync** — every 24h refreshes `channels_twilio.content_templates` for WhatsApp-medium channels whose cache is older than 24h (via `/v1/Content` pagination).

### Asynq Worker (`cmd/worker/`)

A dedicated worker binary consumes asynq task queues from Redis:

- `webhook:outbound` — outbound webhook delivery via `OutboundProcessor` (5 retries: 1s, 5s, 30s, 2m, 10m)
- `channel:wa:send` — WhatsApp Cloud/360Dialog outbound via `WaSendProcessor`

The worker runs as a separate Docker Compose service (`worker`) with its own air config (`.air.worker.toml`).

### Outbound Webhook Notifier

`OutboundWebhookNotifier` is wired into `MessageService` — when a non-private outgoing message is created in a `Channel::Api` inbox, it dispatches the `message_created` event to the configured webhook URL via asynq.

## Channels: Api, Line, Tiktok, Twilio, Twitter

| Kind | Provisioning | Public webhook | Envs / flags |
|------|--------------|----------------|--------------|
| `Channel::Api` | `POST /api/v1/accounts/:aid/inboxes` (type `api`); `POST /inboxes/:id/rotate_token`; public traffic via `/public/api/v1/inboxes/:identifier/*` (api_token SHA-256) | none (uses `public/api/v1`) | — |
| `Channel::Line` | `POST /api/v1/accounts/:aid/inboxes/line` (channel ID + secret + token persisted via KEK) | `POST /webhooks/line/:line_channel_id` (verifies `X-Line-Signature`) | — |
| `Channel::Tiktok` | OAuth 2.0: `POST /inboxes/tiktok/authorize` → `{url}`; `GET /inboxes/tiktok/oauth/callback` (state nonce in Redis 10m) | `POST /webhooks/tiktok/:business_id` (verifies `Tiktok-Signature: t=…,s=…` HMAC-SHA256 with 5m skew) | `FEATURE_CHANNEL_TIKTOK` (default off), `TIKTOK_CLIENT_KEY`, `TIKTOK_CLIENT_SECRET` |
| `Channel::Twilio` | `POST /api/v1/accounts/:aid/inboxes/twilio` (XOR phone_number/messaging_service_sid; credentials validated via `GET /Accounts/{sid}.json`); `POST /inboxes/:id/twilio_templates` | `POST /webhooks/twilio/:identifier` (message) + `POST /webhooks/twilio/:identifier/status` (verifies `X-Twilio-Signature` HMAC-SHA1) | `FEATURE_CHANNEL_TWILIO_WHATSAPP` (default on), `FEATURE_TWILIO_SMS_MEDIUM` (default off — SMS still provisioned via legacy `channel/sms/twilio/` for existing inboxes) |
| `Channel::Twitter` | OAuth 1.0a: `POST /inboxes/twitter/authorize` → `{url}`; `GET /inboxes/twitter/oauth/callback` | `GET /webhooks/twitter/:profile_id` (CRC, returns `sha256=<hmac>`); `POST /webhooks/twitter/:profile_id` (verifies `x-twitter-webhooks-signature`); only `direct_message_events` are ingested, tweets ignored | `FEATURE_CHANNEL_TWITTER` (default off — registry registration gated), `TWITTER_CONSUMER_KEY`, `TWITTER_CONSUMER_SECRET` |

**WhatsApp providers**: `Channel::Whatsapp` supports Meta Cloud API (`provider=whatsapp_cloud`) and 360Dialog (`provider=default_360dialog`) via `ProviderForType()` factory. Wzap engine integration uses `Channel::Api` — configure `wz_elodesk` on the Wzap side.

**Twilio coexistence**: the legacy `channel/sms/twilio/` sub-provider remains registered in `sms.Registry` so existing `channel_type=Channel::Sms` inboxes (with `provider=twilio`) keep delivering inbound/outbound. Creation of new Twilio SMS inboxes via `POST /inboxes/sms?provider=twilio` returns `400 unsupported_provider` pointing callers at `POST /inboxes/twilio` with `medium="sms"`.
