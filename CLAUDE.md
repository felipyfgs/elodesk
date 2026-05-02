# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Idioma

- **Comunicação com o usuário**: pt-BR (Brazilian Portuguese).
- **Código, comentários, mensagens de commit, docs técnicas**: inglês.

Regra firme. O usuário lê em pt-BR; a base de código lê em inglês.

## Repo layout

Monorepo, dois apps + infra Docker. **Não há workspace pnpm/yarn no root** — `backend/` e `frontend/` são instalados/rodados independentemente.

- [backend/](backend/) — Go 1.25, Fiber v2, pgx/v5, asynq. Layered `handler → service → repo`. Makefile vive **aqui**, não no root.
- [frontend/](frontend/) — Nuxt 4 (SPA, `ssr: false`), `@nuxt/ui` v4, Pinia, Tailwind v4, Zod v4.
- [docker-compose.yml](docker-compose.yml) — Postgres 16, Redis 7, MinIO, backend, worker, frontend.
- [openspec/](openspec/) — workflow de propostas para mudanças não-triviais.
- [AGENTS.md](AGENTS.md) — referência detalhada complementar (matriz de canais, fluxos OAuth de cada provedor, etc.). Trate como contexto, não como verdade absoluta — verifique no código quando importar.

## Commands

### Backend ([backend/](backend/))
```bash
make dev              # go run cmd/backend/main.go (migrations rodam no startup)
make build            # CGO_ENABLED=0 → bin/backend
make test             # go test -race ./...
make lint             # golangci-lint run ./...
make docs             # swag init para Swagger
make tidy             # go mod tidy
make install-tools    # instala golangci-lint + swag
```

Worker (binário separado): `go run cmd/worker/main.go` — consome filas asynq do Redis.

### Frontend ([frontend/](frontend/))
```bash
pnpm dev              # nuxt dev (porta 3000)
pnpm build            # build de produção
pnpm lint             # eslint .
pnpm typecheck        # nuxt typecheck (vue-tsc)
pnpm test             # no-op stub
```

### Infra
```bash
docker compose up -d  # postgres (5432), redis (6379), minio (9010 API / 9011 console), backend, worker, frontend
```

## Arquitetura backend

### Entry points
- [backend/cmd/backend/main.go](backend/cmd/backend/main.go) — carrega config, conecta DB, **roda migrations**, inicia Fiber, registra asynqClient, graceful shutdown.
- [backend/cmd/worker/main.go](backend/cmd/worker/main.go) — asynq.Server consumindo filas:
  - `webhook:outbound` — entrega de outbound webhooks (5 tentativas: 1s, 5s, 30s, 2m, 10m)
  - `channel:wa:send` — envio de WhatsApp Cloud / 360Dialog

### Camadas (`backend/internal/`)
- `handler/` — Fiber handlers (HTTP/JSON) — 30+ arquivos
- `service/` — regra de negócio
- `repo/` — acesso a dados via `pgxpool.Pool`. Padrão: cada tipo tem helper `scanX(scanner, *model.X)`. Sem ORM. Inserts idempotentes via `ON CONFLICT … DO UPDATE` quando aplicável.
- `model/` — structs do domínio em [backend/internal/model/](backend/internal/model/)
- `dto/` — payloads de entrada/saída
- `middleware/` — `jwt_auth.go`, `api_token.go`, `org_scope.go`, `roles.go` (Owner=2/Admin=1/Agent=0), `hmac.go`, `widget_cors.go`, `widget_ratelimit.go`
- `crypto/` — KEK AES-256-GCM (`Cipher`) + helpers de hash SHA-256
- `realtime/` — WebSocket hub (raw, **não** socket.io)
- `webhook/` — `OutboundProcessor` para asynq + `OutboundWebhookNotifier` plugado em `MessageService`
- `media/` — cliente MinIO (presigned PUT/GET, proxy upload, download HMAC-signed)
- `audit/`, `filterquery/`, `phone/`, `logger/` (zerolog), `config/`, `database/`

### DI / routes
[backend/internal/server/router.go](backend/internal/server/router.go) é a única fonte de cabeamento — instancia repos, services, handlers e registra todas as rotas. Hub realtime em `router.go:151`. **Nota**: o antigo `_ = channelRegistry` foi removido — `channel/registry.go` foi deletado e cada kind é importado diretamente no router (`appchannel`, `linechan`, `smschan`, `tgchan`, `tiktokchan`, `twiliochan`, `twitterchan`, `whatsappchan`, `webwidget`, …). Não tente reintroduzir o registry.

### Migrations
[backend/internal/database/migrations.go](backend/internal/database/migrations.go) — `go:embed` + `pg_advisory_xact_lock(1)`, **forward-only, sem rollback**, tabela `schema_migrations`. Atualmente 45 arquivos em [backend/migrations/](backend/migrations/) (`NNNN_descricao.sql`). Falha em migration aborta o startup — sempre teste localmente antes de commitar.

### Channels
[backend/internal/channel/](backend/internal/channel/) — interface comum em `channel.go`, dedup em `dedup.go`. Subdiretórios:

| Dir | Notas |
|-----|-------|
| `api/` | `Channel::Api` — REST público em `/public/api/v1/inboxes/:identifier/*`, auth via api_token (SHA-256 at rest) |
| `whatsapp/` | Multi-provider: Meta Cloud API + 360Dialog via factory `ProviderForType()` |
| `sms/` | Sub-registry com `bandwidth/`, `twilio/`, `zenvia/`. Twilio SMS legado mantido para inboxes existentes |
| `email/` | IMAP/SMTP + OAuth |
| `facebook/`, `instagram/` | Meta Graph (compartilham `meta/`) |
| `meta/` | Helpers compartilhados (FB/IG/WhatsApp) |
| `telegram/` | Bot API, secret token por canal |
| `line/`, `tiktok/`, `twitter/`, `twilio/` | OAuth próprios; verifique flags antes de assumir registro |
| `webwidget/` | SSE + JWT de visitante, HMAC para identify |
| `reauth/` | Fluxos de refresh de OAuth |

**Feature flags** (env): `FEATURE_CHANNEL_TIKTOK`, `FEATURE_CHANNEL_TWITTER`, `FEATURE_CHANNEL_TWILIO_WHATSAPP`, `FEATURE_TWILIO_SMS_MEDIUM`. Channels gated por estas flags só são registrados quando `true`.

### Realtime ([backend/internal/realtime/events.go](backend/internal/realtime/events.go))
Único ponto de emissão: `service.MessageService` (`Create`/`UpdateStatus`/`SoftDelete`). Channels delegam — **nunca** broadcast direto.

Eventos canônicos (form `resource.action`):
```
message.created   message.updated   message.deleted
conversation.created   conversation.updated   conversation.deleted
```

Removidos: `message.new`, `conversation.new`, `inbox.status`, `contact.updated`. Não reintroduzir.

WebSocket: `GET /realtime` com JWT em query string ou `Sec-WebSocket-Protocol`. Rooms: `account:N`, `inbox:N`, `conversation:N`. Ping 54s / pong timeout 60s. Backend fecha com 1008 em token inválido/expirado.

### Auth
[backend/internal/service/auth_service.go](backend/internal/service/auth_service.go) — Argon2id (`alexedwards/argon2id`) + JWT v5 (HS256). Access token TTL `JWT_ACCESS_TTL` (default 15m), refresh `JWT_REFRESH_TTL` (default 720h). Refresh token = 48 random bytes, SHA-256 at rest, family ID rastreado para revogar família inteira em replay.

## Arquitetura frontend

### Stack
Nuxt 4 (`ssr: false`), `@nuxt/ui` v4, Pinia, Vue 3, Tailwind v4, TypeScript. Bundler Vite com `optimizeDeps` pré-empacotando: zod, date-fns, pinia, @unovis, tiptap, wavesurfer.js, vue3-emoji-picker, markdown-it, pdfjs-dist, libphonenumber-js.

### Pastas ([frontend/app/](frontend/app/))
- `pages/` — `index.vue`, `login.vue`, `register.vue`, `forgot-password.vue`, `reset-password.vue`, `[...slug].vue`, e seções: `accounts/`, `inboxes/`, `conversations/`, `contacts/`, `notifications/`, `reports/`, `settings/`, `sessions/`
- `stores/` — 18 Pinia stores: `accounts`, `agents`, `audioPlayer`, `auth`, `cannedResponses`, `contacts`, `conversations`, `customAttributes`, `inboxes`, `labels`, `macros`, `messages`, `notes`, `notifications`, `savedFilters`, `sla`, `teams`, `webhooks`
- `composables/` — 12 composables: `useApi`, `useAuth`, `useRealtime`, `useConversationRealtime`, `useAttachmentSrc`, `useContactSearch`, `useConversationFilters`, `useDashboard`, `useDetailsSidebar`, `useErrorHandler`, `useFilterAttributes`, `useResponsive`
- `schemas/` — schemas Zod (validação runtime + tipos)
- `utils/` — helpers (incluindo `chatAdapter.ts` para roles/sides/variants e `attachmentMediaUrl.ts` para resolução estática de URL)
- `components/`, `layouts/`, `middleware/`, `types/`, `assets/`

### Composables-chave
- [useApi.ts](frontend/app/composables/useApi.ts) — `$fetch` com `Authorization: Bearer`, intercepta 401 → `/auth/refresh` (deduplicado) → retry → redirect `/login` em falha. Adapter normaliza `{success, data}` → flat data e snake→camelCase, e converte epoch-segundos → ms.
- [useRealtime.ts](frontend/app/composables/useRealtime.ts) — **WebSocket raw via `useWebSocket` do `@vueuse/core`** (não socket.io, apesar do que README pode sugerir). Singleton por aba. JWT na URL. Decodifica `exp` do token e refresca proativamente ~60s antes da expiração. Close code 1008 dispara refresh.

### UI primitives
Use componentes `@nuxt/ui` v4 (`UCard`, `UButton`, `UStepper`, `UTimeline`, `UChat*`, `UPageGrid`, `UEmpty`, `UUser`, `useToast`, `useOverlay`). **Não criar wrappers customizados** salvo quando adicionando comportamento de domínio. Charts: Unovis (`VisArea`, `VisLine`, `VisGroupedBar`, `VisTooltip`, `VisCrosshair`).

### i18n
`@nuxtjs/i18n` v9, `strategy: 'no_prefix'`, default `pt-BR`, secundário `en`, detecção via cookie. Toda string visível passa por `t('chave')`.

## Domain model ([backend/internal/model/](backend/internal/model/))

- **Account** → tenant top-level. Status: 0=active, 1=suspended.
- **Inbox** → abstração central, possui `channel_type` + record channel-specific.
- **ContactInbox** → bridge `Contact ↔ Inbox` via `source_id` (identificador do canal).
- **Conversation** → pertence a `ContactInbox` (não direto ao Contact). Tem `display_id` sequencial por account, `uuid`, `pubsub_token`. Status: 0=open, 1=resolved, 2=pending, 3=snoozed.
- **Message** → pertence a `Conversation`. `sender_type`/`sender_id` polimórficos (User ou Contact). Tipos: 0=incoming, 1=outgoing, 2=activity, 3=template. Status: 0=sent, 1=delivered, 2=read, 3=failed. ContentType numérico (0=text, 9=image, 10=video, 11=sticker, 12=audio, 13=file, …).
- **Attachment** → fileType: 0=image, 1=audio, 2=video, 3=file, 4=location, 5=fallback.
- **Outras**: Label, Team, CannedResponse, Note, CustomAttributeDefinition, CustomFilter, Macro, SLAPolicy, AuditLog, Notification, OutboundWebhook.
- **JSONB columns** comuns: `additional_attributes`, `custom_attributes`, `content_attributes`, `provider_config`, `meta`.

## Background jobs (in-process tickers no backend)
- **SLA breach detection** — a cada 60s; flag `sla_breached`, emite `sla.breached` no room da account, persiste notificação, registra audit log.
- **Audit retention** — a cada 24h; deleta `audit_logs` com mais de 90 dias.
- **Twilio content templates sync** — a cada 24h; refresca `channels_twilio.content_templates` para canais WhatsApp-medium com cache > 24h.

Worker asynq separado consome `webhook:outbound` e `channel:wa:send` (ver Entry points).

## Uploads (MinIO)
[backend/internal/handler/upload_handler.go](backend/internal/handler/upload_handler.go) — duas formas:
1. **Presigned PUT/GET** com TTL 15min. Path do PUT precisa começar com `{accountId}/`.
2. **Proxy upload** — multipart → backend → MinIO interno.

Download público de attachment: `/api/v1/attachments/:id/file` com token HMAC determinístico (KEK base64-decoded como secret), URL estável por `(accountId, attachmentId)` para cache HTTP.

## Variáveis de ambiente críticas

| Var | Notas |
|-----|-------|
| `JWT_SECRET` | ≥32 chars |
| `BACKEND_KEK` | base64 ≥32 bytes pós-decode. `openssl rand -base64 32` |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | default `15m` / `720h` |
| `DATABASE_URL` / `REDIS_URL` | conexões |
| `MINIO_*` | `MINIO_ENDPOINT`, `MINIO_PORT`, `MINIO_BUCKET`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, mais variantes `MINIO_PUBLIC_*` para URL pública |
| `API_URL` | URL pública do backend (montagem de links de mídia em webhooks) — precisa ser alcançável de containers externos |
| `CORS_ORIGINS` | ex: `*` em dev |
| `META_*` | `META_APP_ID`, `META_APP_SECRET`, `META_GRAPH_VERSION`, `META_ALLOW_UNSIGNED`, `INSTAGRAM_VERIFY_TOKEN`, `FB_VERIFY_TOKEN` |
| Feature flags | `FEATURE_CHANNEL_TIKTOK`, `FEATURE_CHANNEL_TWITTER`, `FEATURE_CHANNEL_TWILIO_WHATSAPP`, `FEATURE_TWILIO_SMS_MEDIUM` |
| Widget | `WIDGET_PUBLIC_BASE_URL`, `WIDGET_JWT_SECRET`, `WIDGET_SESSION_TTL` |

Frontend: `NUXT_PUBLIC_API_URL`, `NUXT_PUBLIC_WS_URL`.

## API routes (resumo)

| Prefix | Auth | Propósito |
|--------|------|-----------|
| `GET /health`, `GET /docs/*` | none | Health, Swagger |
| `POST /api/v1/auth/*` | none | register, login, refresh, logout, forgot, reset, mfa, invitations/:token/accept |
| `GET /realtime` | JWT (query/WS header) | WebSocket |
| `PUT /api/v1/users/:id` | JWT (self) | profile + password |
| `/api/v1/users/:id/notification_preferences` | JWT (self) | GET/PUT preferences |
| `/api/v1/accounts/:aid/*` | JWT + org scope | inboxes, contacts, conversations, messages, uploads, labels, teams, canned, attributes, filters, agents, macros, slas, webhooks, audit_logs, notifications, reports |
| `/public/api/v1/inboxes/:identifier/*` | api_token (SHA-256) | contacts, conversations, messages |
| `/webhooks/*` | provider-specific | sms, instagram, facebook, telegram, line, tiktok, twilio, twitter, email |
| `/widget/:token`, `/widget/:token/ws` | CORS + rate limit | SSE widget |
| `/api/v1/widget/*` | widget auth | sessions, messages, identify, attachments |

## Skills (carregadas sob demanda)

Regras de estilo e domínio vivem em skills, não duplicadas em markdown:
- **Backend**: `go-backend`, `go-errors`, `go-logging`, `go-security`, `go-testing`, `go-migrations`
- **Frontend**: `nuxt-frontend`, `nuxt-ui`

## OpenSpec workflow

Mudanças não-triviais passam por proposta antes da implementação. Comandos slash em [.claude/commands/](.claude/commands/):
- `/opsx:propose` — scaffold de change proposal
- `/opsx:explore` — think mode (sem edits)
- `/opsx:apply` — implementa as tasks
- `/opsx:archive` — move para `openspec/changes/archive/YYYY-MM-DD-<name>/`
- `/go-quality`, `/frontend-quality` — review + auto-fix
- `/full-test`, `/dev-setup` — testes / bootstrap

Mudanças in-flight ficam em [openspec/changes/](openspec/changes/).

## MCP servers

Configurados em [.mcp.json](.mcp.json) e habilitados via [.claude/settings.local.json](.claude/settings.local.json):
- `nuxt-ui`, `nuxt` (HTTP) — docs/componentes oficiais
- `postgres-elodesk`, `postgres-wzap` — query direta nos bancos (read-only espera-se)

Use `nuxt-ui` para descobrir componentes/exemplos antes de criar UI customizada.

## Gotchas críticos (release-blockers se ignorados)

1. **Migrations rodam no startup** com advisory lock. Forward-only. Migration quebrada = backend não sobe.
2. **Realtime emit single-point**: só `MessageService` emite eventos de mensagem. Adicionar broadcast em channel handler é bug.
3. **`_ = channelRegistry` não existe mais** — `registry.go` foi deletado. Imports diretos no router.
4. **MinIO portas não-padrão**: `9010:9000` e `9011:9001`. `MINIO_PUBLIC_PORT=9010` para URLs públicas.
5. **Defaults docker-compose**: usuário/senha/db = `wzap`/`wzap`/`wzap` (legado do nome anterior).
6. **WebSocket é raw, não socket.io** — apesar do que `frontend/README.md` insinua. Source of truth: [useRealtime.ts](frontend/app/composables/useRealtime.ts).
7. **Home `/`** redireciona authenticated users para `/accounts/{primaryId}` (dashboard com stats), não para conversations.
8. **Channel API token / HMAC token / widget hmacToken** são mostrados em **plaintext apenas uma vez** na criação; persistidos como SHA-256 / AES-GCM ciphertext.
9. **Worker asynq é binário separado** ([cmd/worker/main.go](backend/cmd/worker/main.go)) — não esqueça de subir junto em prod.
