## Why

O backend NestJS/TypeScript em `backend/` foi construído como **cliente do wzap** (chama `wzap.createSession`, `wzap.sendText`, assina webhooks específicos, modela `ChannelWhatsapp` com `wzapSessionId`). Acopla vertical — trocar de engine exige reescrever o backend. Ao mesmo tempo, o `wzap/` já é Go/Fiber/pgx e já fala um contrato **Chatwoot-compatível** via `wzap/internal/integrations/chatwoot/`. Queremos um backend **Go idiomático**, desacoplado por contrato (Channel::Api do Chatwoot), usando a mesma stack e convenções do `wzap/` (consistência entre os repos, ops uniformes, ganhos de performance e footprint). Resultado: qualquer engine que fale Chatwoot (wzap, Evolution API, Meta Cloud, Baileys wrapper) pluga sem modificar uma linha do backend.

## What Changes

- **BREAKING**: remover por inteiro `backend/` NestJS atual (services, Prisma schema, Jest, nest-cli, package.json, .env.example TS). Migration Prisma e estrutura TS deixam de existir.
- Criar `backend/` em Go 1.22 com a organização espelhando `wzap/`: `cmd/backend/main.go`, `internal/{config,logger,database,migrations,server,middleware,handler,service,repo,model,dto,realtime,webhook,media,integrations}/`, `Makefile`, `Dockerfile`, `docker-compose.yml` (o da raiz cobre Postgres/Redis/MinIO).
- Stack Go: Fiber v2 (HTTP), pgx v5 (Postgres com connection pool), migrations SQL embutidas via `//go:embed` (mesmo padrão do wzap), `golang-jwt/jwt` (access + refresh com rotação), `zerolog` ou `slog` (estruturado, campo `component=<module>`), `go-redis/redis` + `hibiken/asynq` (filas com retries exponenciais), `minio-go` (storage), `gorilla/websocket` para realtime (gateway com rooms hierárquicos).
- Expor **Channel::Api** Chatwoot-compatível (host): endpoints `/api/v1/accounts/:aid/...` com header `api_access_token` para providers, `/api/v1/auth/*` com JWT para agentes, `/public/api/v1/inboxes/:identifier/...` para read receipts. Outbound webhooks assinados com `X-Chatwoot-Hmac-Sha256`.
- Substituir Socket.IO do NestJS por WebSocket nativo Go (`gorilla/websocket`) com auth JWT no handshake e rooms `account:*`/`inbox:*`/`conversation:*`. Frontend troca `socket.io-client` por cliente WS fino.
- Schema Postgres permanece conceitualmente idêntico (Account/User/AccountUser/Inbox/ChannelApi/Contact/ContactInbox/Conversation/Message/Attachment) mas implementado em SQL puro com migrations numeradas, sem Prisma/ORM. Repos usam `pgx.Row`/`pgx.Rows` com `scanXxx` helpers (padrão do wzap).
- `frontend/` Nuxt 4 permanece, mas: (1) troca cliente realtime para WebSocket nativo; (2) store `inboxes` remove `channelWhatsapp` e ganha `channelApi: {identifier, webhookUrl, hmacMandatory}`; (3) remove `SessionQrModal.vue` (QR é responsabilidade do provider); (4) página `/sessions` vira criação de Channel::Api com credenciais copiáveis pós-criação.
- Paridade de observabilidade: `GET /health` agregado (db+redis), Swagger via `swaggo/swag` (igual ao wzap), logs com redact de `Authorization`/`password`/`token`/`hmacToken`.
- CI: remover job `gen-wzap-drift` (não há mais codegen de swagger). Adicionar `go test -race ./...`, `golangci-lint run ./...`.

## Capabilities

### New Capabilities

- `backend-go-bootstrap`: scaffolding Go (cmd/internal/migrations), config via env com validação, logger pino-equivalente, `/health`, Swagger, Dockerfile, Makefile.
- `backend-go-auth`: JWT access+refresh com rotação e revogação de família, Argon2id para senhas, registro/login/logout, guards middleware para Fiber.
- `backend-go-tenancy`: `OrgScopeMiddleware` validando `X-Account-Id` ou `:accountId`, `RolesMiddleware` com `@Roles(OWNER/ADMIN/AGENT)` via tags/context, decorators equivalentes `CurrentUser`/`CurrentAccount` via Fiber `Locals`.
- `backend-go-channels-api`: Channel::Api Chatwoot-compatível (contacts/conversations/messages/attachments/actions), auth por `api_access_token` header, HMAC inbound opcional, multipart uploads, idempotência por `source_id`.
- `backend-go-outbound-webhooks`: emissão HTTP POST assinada (`X-Chatwoot-Hmac-Sha256`) para `channel_api.webhook_url` em mudanças de estado (`message_created`, `message_updated`, `conversation_status_changed`, `conversation_updated`), retry exponencial via asynq.
- `backend-go-realtime`: WebSocket gateway (gorilla/websocket) com auth JWT no handshake, rooms `account:{id}`/`inbox:{id}`/`conversation:{id}` com validação de membership, DTOs mappers sem campos internos.
- `backend-go-media`: MinIO SDK Go, upload inbound via multipart, presigned PUT/GET URLs (TTL 15min), organização `{accountId}/{inboxId}/{messageId}.{ext}`.

### Modified Capabilities

- `frontend-app`: trocar `socket.io-client` por cliente WebSocket nativo (ou `@vueuse/useWebSocket`); remover `SessionQrModal.vue`; ajustar store `inboxes` para `channelApi`; página `/sessions` exibe credenciais de Channel::Api geradas.

## Não-objetivos

- Não alterar nada em `wzap/` (engine permanece imutável; ele já fala Chatwoot).
- Não migrar dados do backend NestJS (nada em produção; `backend/` é reescrita limpa).
- Não introduzir gRPC, GraphQL ou arquiteturas distribuídas além do que o wzap já usa (single binary + Postgres + Redis + MinIO).
- Não cobrir canais diferentes de WhatsApp no MVP (embora `ChannelApi` seja genérico; Instagram/email/webchat ficam para v2).
- Não implementar billing, 2FA, OAuth social, convites por email.
- Não expor OpenAPI codegen para clientes (frontend lê tipos via dicionário manual por ora).

## Riscos e mitigações

- **Regressão funcional na troca de stack** → manter paridade exata de contrato HTTP/WebSocket com o que o frontend já consome; adicionar testes de integração por rota (`httptest` + `pgx` contra Postgres de teste).
- **Complexidade de migrations SQL puro** → copiar padrão de `wzap/migrations/*.sql` + `//go:embed` (já funciona em produção).
- **Filas com asynq vs BullMQ** → mapear 1:1 as queues (`provider-webhooks`, `media-download`), backoff exponencial equivalente (1s/5s/30s/2m/10m).
- **WebSocket sem fallback long-polling** → aceitar (frontend dev em LAN; pra prod avaliar TLS/ALB sticky sessions). Socket.IO era conforto, não requisito.
- **Perda de tipos auto-gerados** → como não há cliente wzap embutido, não há swagger pra consumir; frontend e backend declaram DTOs explícitos.

## Impact

- **Deletado**: `backend/` inteiro atual (src TS, prisma/, package.json, .env.example, eslint.config.mjs, jest config, tsconfig, nest-cli, ~50 arquivos TS).
- **Criado**: `backend/` Go novo (`cmd/backend/`, `internal/**`, `migrations/*.sql`, `Dockerfile`, `Makefile`, `go.mod`, `go.sum`, `.env.example`, `README.md`).
- **Modificado**: `pnpm-workspace.yaml` (remove `backend`), `turbo.json` (remove tasks backend ou aponta pra `make`), `package.json` raiz (scripts `dev/build` usam Makefile do backend), `.github/workflows/ci.yml` (substitui jobs Node por `go test -race` + `golangci-lint run`).
- **Frontend**: `frontend/package.json` (remove `socket.io-client`, adiciona ou reutiliza `@vueuse/core` para `useWebSocket`); `frontend/app/composables/useRealtime.ts` reescrito pra WS nativo; `frontend/app/stores/inboxes.ts` tipo atualizado; `frontend/app/components/SessionQrModal.vue` removido; `frontend/app/pages/sessions/index.vue` ajustado pra criação de Channel::Api.
- **Dependências novas (Go)**: `github.com/gofiber/fiber/v2`, `github.com/jackc/pgx/v5`, `github.com/redis/go-redis/v9`, `github.com/hibiken/asynq`, `github.com/minio/minio-go/v7`, `github.com/golang-jwt/jwt/v5`, `github.com/alexedwards/argon2id`, `github.com/rs/zerolog`, `github.com/gorilla/websocket`, `github.com/swaggo/swag`, `github.com/go-playground/validator/v10`.
- **Dependências removidas (TS)**: toda a árvore `@nestjs/*`, Prisma, Jest, passport, class-validator, nestjs-pino, openapi-typescript, bullmq, ioredis, socket.io, minio (SDK JS), argon2 (JS).
- **Rollback**: `git revert` da change; `backend/` TS volta pelo histórico.
