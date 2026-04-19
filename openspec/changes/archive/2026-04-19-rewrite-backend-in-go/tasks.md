## 1. Remoção do Backend NestJS

- [x] 1.1 Deletar diretório `backend/` NestJS inteiro (src, prisma, node_modules, package.json, tsconfig, nest-cli, .env.example, eslint, jest config)
- [x] 1.2 Remover `backend` de `pnpm-workspace.yaml`
- [x] 1.3 Atualizar `turbo.json` removendo tasks do backend Node ou apontando para `make` do novo backend Go
- [x] 1.4 Remover dependências TS do backend do `package.json` raiz (se houver referências)
- [x] 1.5 Atualizar `.github/workflows/ci.yml` removendo jobs Node do backend (test, lint, build)
- [x] 1.6 Confirmar build limpo do monorepo após remoção (`pnpm install` sem erros)

## 2. Scaffolding Go — Bootstrap

- [x] 2.1 Criar `backend/cmd/backend/main.go` (entrypoint único, invoca config, logger, database, router)
- [x] 2.2 Criar `backend/internal/config/config.go` — carregamento de env via `os.Getenv` + validação de obrigatórias (PORT, DATABASE_URL, REDIS_URL, JWT_SECRET, BACKEND_KEK, MINIO_*, API_URL)
- [x] 2.3 Criar `backend/internal/logger/logger.go` — zerolog singleton com `WithComponent(name)` e função `redactHeaders`
- [x] 2.4 Criar `backend/internal/database/pool.go` — pool pgx v5 com `Connect(ctx, databaseURL)` e `Close()`
- [x] 2.5 Criar `backend/internal/database/migrations.go` — `RunMigrations(ctx, pool)` lendo `migrations/*.sql` via `//go:embed`, tabela `schema_migrations`
- [x] 2.6 Criar `backend/internal/server/router.go` — registro de rotas + DI manual (handler→service→repo wiring)
- [x] 2.7 Criar `backend/internal/server/errors.go` — fiber error handler central + sentinel wrappers (`ErrNotFound`, `ErrConflict`, `ErrForbidden`)
- [x] 2.8 Criar `backend/internal/model/` com structs do domínio (User, Account, AccountUser, RefreshToken, Inbox, ChannelApi, Contact, ContactInbox, Conversation, Message, Attachment)
- [x] 2.9 Criar `backend/internal/dto/` com structs request/response + tags `validate` e `json`
- [x] 2.10 Criar `backend/Dockerfile` (multi-stage, CGO_ENABLED=0, binário estático)
- [x] 2.11 Criar `backend/Makefile` (targets: dev, build, test, lint, docs, tidy, seed)
- [x] 2.12 Criar `backend/.env.example` com todas as variáveis documentadas
- [x] 2.13 Criar `backend/go.mod` com módulo e dependências (fiber, pgx, go-redis, asynq, minio-go, golang-jwt, argon2id, zerolog, gorilla/websocket, swaggo, validator)
- [x] 2.14 Criar `backend/internal/handler/health_handler.go` — `GET /health` retornando `{status, db, redis}`
- [x] 2.15 Criar `backend/migrations/0001_init.sql` — tabelas users, accounts, account_users, refresh_tokens + índices
- [x] 2.16 Configurar Swagger via swaggo: anotações `// @Summary` no health handler + `make docs` gera `docs/swagger.yaml`
- [x] 2.17 Verificar `make build` gera `bin/backend` e `make dev` sobe o servidor com `GET /health` retornando ok

## 3. Banco de Dados — Migrations Complementares

- [x] 3.1 Criar `backend/migrations/0002_inbox_channel_api.sql` — tabelas inboxes, channels_api + índice composto `(account_id, id)`
- [x] 3.2 Criar `backend/migrations/0003_messaging.sql` — tabelas contacts, contact_inboxes, conversations, messages, attachments + unique parcial `(inbox_id, source_id)`
- [x] 3.3 Criar `backend/migrations/0004_audit.sql` — tabela audit_events (futura referência)
- [x] 3.4 Verificar `RunMigrations` aplica todas em ordem e é idempotente (re-run sem erro)

## 4. Camada de Repositório

- [x] 4.1 Criar `backend/internal/repo/user_repo.go` — Create, FindByEmail, FindByID (com accountID quando aplicável)
- [x] 4.2 Criar `backend/internal/repo/account_repo.go` — Create, FindByID, FindBySlug, AddUser
- [x] 4.3 Criar `backend/internal/repo/refresh_token_repo.go` — Create, FindByHash, Revoke, RevokeByFamily, RevokeAllByUserID
- [x] 4.4 Criar `backend/internal/repo/inbox_repo.go` — Create, FindByID, ListByAccount (requer accountID)
- [x] 4.5 Criar `backend/internal/repo/channel_api_repo.go` — Create, FindByInboxID, FindByApiToken (comparação constant-time)
- [x] 4.6 Criar `backend/internal/repo/contact_repo.go` — Create, Upsert, FindByID, Search, Filter, FindConversations (requer accountID)
- [x] 4.7 Criar `backend/internal/repo/conversation_repo.go` — Create, FindByID, ToggleStatus, ListByAccount com filtros (requer accountID)
- [x] 4.8 Criar `backend/internal/repo/message_repo.go` — Create, FindByID, SoftDelete, ListByConversation com paginação reversa (requer accountID, idempotência por source_id)
- [x] 4.9 Criar `backend/internal/repo/attachment_repo.go` — Create, FindByID, FindByMessageID
- [x] 4.10 Implementar padrão `scanXxx(scanner, &m)` + sentinel errors double-wrapped em todos os repos
- [x] 4.11 Garantir que toda query de leitura/escrita em repos tenant-scoped exige `accountID` (testes unitários)

## 5. Autenticação

- [x] 5.1 Criar `backend/internal/service/auth_service.go` — Register (User + Account + AccountUser em tx, Argon2id), Login, Refresh (rotação + anti-replay), Logout (single + all devices)
- [x] 5.2 Criar `backend/internal/handler/auth_handler.go` — `POST /api/v1/auth/register`, `/login`, `/refresh`, `/logout` com `parseAndValidate`
- [x] 5.3 Criar `backend/internal/middleware/jwt_auth.go` — valida `Authorization: Bearer <token>`, popula `c.Locals("user")`, rejeita 401
- [x] 5.4 Implementar geração JWT HS256 (access 15min) e refresh token (48 bytes crypto/rand, SHA-256 hash no DB, family_id, TTL 30d)
- [x] 5.5 Testar registro com email novo (201 + tokens), email duplicado (409), login correto (200), credenciais erradas (401 genérico)
- [x] 5.6 Testar refresh válido (rotação), refresh replay (401 + revogação da família), logout single, logout all devices

## 6. Multi-tenancy

- [x] 6.1 Criar `backend/internal/middleware/org_scope.go` — extrai accountId de path/header, confere Account existência, confere AccountUser membership, popula Locals
- [x] 6.2 Criar `backend/internal/middleware/roles.go` — `RolesRequired(roles ...Role)` lê `c.Locals("role")` e rejeita 403 se não permitida
- [x] 6.3 Criar helpers `CurrentUser(c)` e `CurrentAccount(c)` em `backend/internal/server/helpers.go`
- [x] 6.4 Testar user pertence à account (passa), account inexistente (404), cross-tenant (403)
- [x] 6.5 Testar AGENT tenta operação OWNER-only (403), ADMIN executa (passa)

## 7. Channel::Api — Endpoints Chatwoot-compatíveis

- [x] 7.1 Criar `backend/internal/middleware/api_token.go` — valida `api_access_token` header, descriptografa e compara constant-time, popula `c.Locals("inbox")` e `c.Locals("account")`
- [x] 7.2 Criar `backend/internal/service/inbox_service.go` — provisionamento de Inbox + ChannelApi com identifier, api_token, hmac_token encriptados com KEK
- [x] 7.3 Criar `backend/internal/handler/inbox_handler.go` — `POST /api/v1/accounts/:aid/inboxes` (OWNER/ADMIN), `GET` lista e detalhe (segredos só na criação)
- [x] 7.4 Criar `backend/internal/service/contact_service.go` — create (upsert por identifier), search, filter, update, merge
- [x] 7.5 Criar `backend/internal/handler/contact_handler.go` — endpoints contacts Chatwoot-compatíveis (search, filter, create, update, conversations, merge)
- [x] 7.6 Criar `backend/internal/service/conversation_service.go` — create (com ContactInbox implícito), toggle_status
- [x] 7.7 Criar `backend/internal/handler/conversation_handler.go` — endpoints conversations Chatwoot-compatíveis (create, toggle_status, list paginado)
- [x] 7.8 Criar `backend/internal/service/message_service.go` — create (JSON + multipart, idempotência por source_id), soft delete
- [x] 7.9 Criar `backend/internal/handler/message_handler.go` — `POST messages` (JSON + multipart), `DELETE messages/:mid`
- [x] 7.10 Criar endpoint `POST /public/api/v1/inboxes/:identifier/contact_inboxes/conversations/:cid/update_last_seen` (auth por identifier + HMAC opcional)
- [x] 7.11 Criar `backend/internal/middleware/hmac.go` — validação opcional de `X-Chatwoot-Hmac-Sha256` em requests inbound
- [x] 7.12 Testar criação de inbox retorna credenciais em claro; GET posterior não retorna segredos
- [x] 7.13 Testar idempotência de contact upsert por identifier e message por source_id
- [x] 7.14 Testar multipart upload com arquivo e criação de attachment

## 8. Webhooks Outbound

- [x] 8.1 Criar `backend/internal/webhook/outbound_processor.go` — handler asynq que POSTa JSON assinado (HMAC-SHA256) pro `channel_api.webhook_url`
- [x] 8.2 Criar `backend/internal/service/outbound_webhook_service.go` — enfileira jobs asynq para `message_created`, `message_updated`, `conversation_status_changed`, `conversation_updated`
- [x] 8.3 Implementar backoff exponencial `1s, 5s, 30s, 2m, 10m` (5 tentativas) e dead-letter após esgotar
- [x] 8.4 Garantir `X-Delivery-Id` (UUID v4) estável entre retries do mesmo job
- [x] 8.5 Garantir dispatch não bloqueia request do agente (enfileira e retorna imediatamente)
- [x] 8.6 Testar com servidor HTTP de mentira: mensagem outgoing gera webhook, HMAC bate, retry em 5xx

## 9. Realtime — WebSocket

- [x] 9.1 Criar `backend/internal/realtime/hub.go` — hub gorilla/websocket com `map[string]map[*Client]struct{}` protegido por `sync.RWMutex`, broadcast por room
- [x] 9.2 Criar `backend/internal/realtime/client.go` — struct Client (conn, userID, rooms), readPump/writePump
- [x] 9.3 Criar `backend/internal/handler/realtime_handler.go` — `GET /realtime` com upgrade WS, auth JWT no handshake (header `Sec-WebSocket-Protocol` ou query `?token=`)
- [x] 9.4 Implementar join/leave de rooms (`join.account`, `join.inbox`, `join.conversation`) com validação de membership via AccountUser
- [x] 9.5 Criar `backend/internal/service/realtime_service.go` — broadcast de `message.new`, `message.updated`, `conversation.new`, `conversation.updated`, `inbox.status`
- [x] 9.6 Garantir payloads de broadcast são DTOs (nunca entidade DB crua), sem `password_hash`/`api_token`/`hmac_token`
- [x] 9.7 Testar handshake válido (aceita), sem token (401), join em account alheia (erro), join válido (recebe broadcasts)
- [x] 9.8 Testar isolamento cross-tenant: broadcast em account A não chega em sockets de account B

## 10. Mídia — MinIO

- [x] 10.1 Criar `backend/internal/media/minio_client.go` — inicialização MinIO client, `EnsureBucket` (auto-provision no startup, warn em falha)
- [x] 10.2 Criar `backend/internal/media/upload.go` — stream de multipart para MinIO, limite 256 MB, path `{accountId}/{inboxId}/{messageId}.{ext}`
- [x] 10.3 Criar `backend/internal/handler/upload_handler.go` — `POST /api/v1/accounts/:aid/uploads/signed-url` (presigned PUT, TTL 15min)
- [x] 10.4 Criar endpoint `GET /api/v1/accounts/:aid/attachments/:id/signed-url` (presigned GET, TTL 15min)
- [x] 10.5 Integrar upload de attachment no `message_handler` multipart (stream direto pro MinIO + Attachment row)
- [x] 10.6 Testar upload de imagem (MinIO recebe, attachment row criada), arquivo acima do limite (413), presigned PUT/GET

## 11. Frontend Cutover

- [x] 11.1 Remover `socket.io-client` de `frontend/package.json` e instalar/verificar `@vueuse/core` disponível
- [x] 11.2 Reescrever `frontend/app/composables/useRealtime.ts` — WebSocket nativo via `useWebSocket`, JWT via header/query, re-join automático após reconexão com backoff
- [x] 11.3 Atualizar `frontend/app/stores/inboxes.ts` — remover `channelWhatsapp`, adicionar `channelApi: {identifier, webhookUrl, hmacMandatory}`
- [x] 11.4 Remover `frontend/app/components/SessionQrModal.vue` (QR é responsabilidade do provider)
- [x] 11.5 Adaptar `frontend/app/pages/sessions/index.vue` — exibir criação de Channel::Api com credenciais copiáveis pós-criação
- [x] 11.6 Atualizar `frontend/app/stores/auth.ts` — ajustar endpoints para o novo backend Go (register/login/refresh/logout)
- [x] 11.7 Atualizar stores `conversations.ts` e `messages.ts` — endpoints Chatwoot-compatíveis do novo backend
- [x] 11.8 Smoke test manual: login, criar inbox, listar conversas, enviar mensagem, receber eventos WS

## 12. CI e Documentação

- [x] 12.1 Adicionar jobs `go test -race ./...` e `golangci-lint run ./...` em `.github/workflows/ci.yml`
- [x] 12.2 Remover job `gen-wzap-drift` do CI (não há mais codegen de swagger TS)
- [x] 12.3 Criar `backend/README.md` — instruções de setup, desenvolvimento, build, deploy, variáveis de ambiente
- [x] 12.4 Atualizar `README.md` raiz — refletir stack Go no backend, comandos atualizados
- [x] 12.5 Criar `backend/.golangci.yml` — linters configurados (errcheck, govet, staticcheck, etc.)
- [x] 12.6 Verificar pipeline CI passa (go test, golangci-lint, frontend build)

## 13. Validação Ponta-a-Ponta

- [x] 13.1 Subir stack completa: Postgres + Redis + MinIO (docker-compose) + backend Go + frontend
- [x] 13.2 Registrar user via `POST /api/v1/auth/register` e verificar tokens
- [x] 13.3 Criar inbox via `POST /api/v1/accounts/:aid/inboxes` e copiar credenciais
- [x] 13.4 Apontar wzap com `CHATWOOT_URL=http://localhost:3001` usando credenciais da inbox criada
- [x] 13.5 Wzap posta contact + message via Channel::Api — verificar persistência no backend
- [x] 13.6 Agente responde via frontend — verificar webhook outbound entregue ao wzap (HMAC bate)
- [x] 13.7 Verificar mensagem chega no WA real via wzap
- [x] 13.8 Verificar eventos WS em tempo real no frontend (message.new, conversation.updated)
- [x] 13.9 Verificar healthcheck (`GET /health`) e Swagger (`GET /docs/`) funcionais
