## Why

Hoje só existe o `wzap/` (gateway Go/whatsmeow) e um template Nuxt (`dashboard/`) sem qualquer integração. Não há camada de produto — não existe multi-tenancy, autenticação de agentes, persistência de conversas ou UI de atendimento. Queremos um produto SaaS de atendimento sobre o wzap sem acoplar lógica de produto ao engine: o wzap deve ficar como provedor WhatsApp (igual o Chatwoot o trata hoje via [wzap/internal/integrations/chatwoot/](../../../wzap/internal/integrations/chatwoot/)) e ganhamos um **backend NestJS próprio** que consome a API do wzap e serve um **frontend Nuxt** dedicado.

## What Changes

- Novo projeto **`backend/`** (NestJS + Prisma + Postgres + Redis/BullMQ + Socket.IO) que consome o wzap via REST + webhooks HMAC — mesmo padrão da integração Chatwoot em [wzap/internal/integrations/chatwoot/handler.go](../../../wzap/internal/integrations/chatwoot/handler.go) — e expõe uma API multi-tenant para o frontend.
- Renomear `dashboard/` para **`frontend/`** e transformá-lo em cliente exclusivo do novo backend: Pinia, socket.io-client, composables `useApi`/`useRealtime`, páginas `login`/`sessions`/`conversations`, i18n PT-BR, remoção dos mocks em `server/api/*`.
- Monorepo raiz em `/home/obsidian/dev/project/` com `pnpm-workspace.yaml` + Turbo unindo `backend/` + `frontend/`. `wzap/` e `_refs/` ficam fora do workspace; `_refs/` (gitignored) recebe clones de Chatwoot e Whaticket SaaS para estudo.
- Geração automática de tipos TS a partir de [wzap/docs/swagger.yaml](../../../wzap/docs/swagger.yaml) via `openapi-typescript` (script `gen:wzap` no backend).
- Schema Prisma inspirado em Chatwoot: `Account → AccountUser → Inbox → ChannelWhatsapp → Conversation → Message/Attachment`, com `Message.sourceId = "WAID:<msgID>"` para idempotência e linkagem bidirecional.
- MinIO próprio do backend (bucket isolado do wzap): inbound baixa via presigned do wzap e re-uploda; outbound frontend sobe via presigned e backend passa URL pública ao wzap.
- **Não altera o wzap** — toda integração é cliente do contrato público (REST/webhook/WebSocket).

## Capabilities

### New Capabilities

- `backend-core`: Bootstrap NestJS (Config/Logger/Health/Swagger), auth JWT email+senha, models `User`/`Account`/`AccountUser`/`RefreshToken`, registro/login/refresh/logout.
- `tenancy`: Isolamento multi-tenant por `accountId` em todo repo, `OrgScopeGuard`, `RolesGuard` (OWNER/ADMIN/AGENT), decorators `@CurrentAccount`/`@CurrentUser`.
- `wzap-integration`: HTTP client tipado gerado do swagger, WS client por sessão com reconexão, webhook receiver com HMAC, fila BullMQ, `WzapEventService` roteando os 47 `EventType`s definidos em [wzap/internal/model/events.go](../../../wzap/internal/model/events.go).
- `inbox-channel`: Modelagem `Inbox` + `ChannelWhatsapp`, CRUD de sessão WA (create/connect/disconnect/QR/status), criação automática de webhook no wzap com secret próprio.
- `messaging`: `Contact`/`ContactInbox`/`Conversation`/`Message`/`Attachment`; pipeline inbound (upsert + emit), outbound otimista (PENDING → SENT), edição/deleção bidirecional, mídia via MinIO próprio.
- `realtime`: Socket.IO gateway com rooms `account:{id}` / `inbox:{id}` / `conversation:{id}`, auth JWT no handshake, emits em cada ponto do pipeline.
- `frontend-app`: Renomear `dashboard/` → `frontend/`, Pinia stores (`auth`/`accounts`/`inboxes`/`conversations`/`messages`), composables `useApi`/`useRealtime`/`useAuth`, middleware `auth.global`, páginas novas, i18n PT-BR.
- `workspace`: Monorepo pnpm workspace + Turbo, `docker-compose.yml` raiz (postgres+redis+minio), `.gitignore` raiz incluindo `_refs/`, `git mv dashboard frontend`.

### Modified Capabilities

<!-- Nenhuma. A proposta não altera specs existentes. -->

## Não-objetivos

- Não modificar o wzap (rotas, DTOs, schema). Toda mudança aqui é externa ao repositório Go.
- Não implementar billing/planos, verificação de email, 2FA ou convites por email no MVP.
- Não suportar outros canais (email, Instagram, webchat) — `Inbox.channelType` fica aberto mas só `whatsapp` é implementado.
- Não importar dados históricos do Chatwoot/Whaticket; começamos com base vazia.
- Não portar código 1:1 — `_refs/` é só estudo para inspirar arquitetura, não para copy-paste.

## Riscos e mitigações

- **Drift entre DTOs wzap e backend** → mitigado pelo `gen:wzap` (regenera tipos a cada alteração do swagger do wzap); CI roda o script e falha se houver drift.
- **Perda de eventos do wzap** (webhook 5xx ou WS caído) → webhook HTTP é a fonte de verdade; idempotência por `Message.sourceId` unique; BullMQ retenta com backoff; WS por sessão é só otimização de latência.
- **Vazamento cross-tenant** → `OrgScopeGuard` obrigatório em todo controller; índices compostos `(accountId, ...)` em todas as tabelas; teste explícito por rota impedindo user de account A ler dados de account B.
- **Tokens do wzap** → salvos criptografados com KEK de env (`BACKEND_KEK`); HMAC comparado com `crypto.timingSafeEqual`.
- **Renomeação `dashboard/` → `frontend/`** → via `git mv` preserva histórico; feito numa única transação.

## Impact

- **Criado**: `backend/` (NestJS completo), `pnpm-workspace.yaml`, `turbo.json`, `docker-compose.yml` raiz, `.gitignore` raiz, `_refs/` (gitignored).
- **Renomeado**: `dashboard/` → `frontend/`.
- **Não tocado**: nada no diretório `wzap/` (engine permanece read-only para esta mudança).
- **Dependências novas**: NestJS, Prisma, BullMQ, Socket.IO, pino, @nestjs/jwt, Passport, class-validator, openapi-typescript, MinIO SDK JS; Pinia, socket.io-client, @nuxtjs/i18n, @pinia/nuxt no frontend.
- **Infra dev**: docker-compose levanta Postgres 16 + Redis 7 + MinIO; wzap roda em processo separado (`cd wzap && make dev`).
