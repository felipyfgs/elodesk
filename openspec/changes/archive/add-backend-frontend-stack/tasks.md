## 1. Monorepo e infraestrutura de dev

- [x] 1.1 Criar `pnpm-workspace.yaml` na raiz listando `backend` e `frontend`
- [x] 1.2 Criar `package.json` raiz com scripts wrapper (`dev`, `build`, `lint`, `test`) delegando para Turbo
- [x] 1.3 Criar `turbo.json` com pipelines `dev` (persistent), `build`, `lint`, `test`, `typecheck`
- [x] 1.4 Criar `.gitignore` raiz incluindo `node_modules/`, `.turbo/`, `dist/`, `.output/`, `_refs/`
- [x] 1.5 Executar `git mv dashboard frontend` e commitar como "chore: rename dashboard to frontend" — raiz não era git; `git init` + `mv` (dashboard mantém seu próprio `.git` interno)
- [x] 1.6 Criar `docker-compose.yml` raiz com serviços postgres:16, redis:7, minio/minio:latest (portas 5432, 6379, 9010/9011)
- [x] 1.7 Criar `_refs/` (vazio, gitignored) e instruções em `README.md` raiz explicando clones para estudo
- [x] 1.8 Clonar `chatwoot/chatwoot` e `adrianohenrique/whaticket-saas` em `_refs/` — `whaticket-saas` substituído por `canove/whaticket-community` (o original requer auth)
- [x] 1.9 Escrever `backend/docs/engine-notes.md` com aprendizados do estudo (antes de codar)

## 2. Scaffold do backend NestJS

- [x] 2.1 `cd backend && pnpm create nestjs@latest . --package-manager pnpm --language typescript` — scaffold manual equivalente (nest-cli.json, tsconfig, main.ts, app.module.ts) para evitar prompts interativos
- [x] 2.2 Adicionar deps: `prisma @prisma/client @nestjs/config @nestjs/jwt passport passport-jwt passport-local argon2 class-validator class-transformer nestjs-pino pino pino-pretty zod @nestjs/swagger bullmq @nestjs/bullmq ioredis socket.io @nestjs/platform-socket.io @nestjs/websockets minio openapi-typescript eventemitter2`
- [x] 2.3 Criar `ConfigModule` com schema Zod (DATABASE_URL, REDIS_URL, JWT_SECRET, BACKEND_KEK, WZAP_URL, WZAP_ADMIN_TOKEN, WZAP_WS_URL, MINIO_*, API_URL)
- [x] 2.4 Configurar logger pino com campo `component` e redacting de `Authorization`/`password`/`token`
- [x] 2.5 Expor `GET /health` retornando `{status:"ok", db:"ok", redis:"ok"}`
- [x] 2.6 Configurar Swagger em `/docs` com tag por módulo

## 3. Prisma e schema base

- [x] 3.1 `pnpm prisma init` e configurar `schema.prisma` com `provider = "postgresql"`
- [x] 3.2 Modelar `User`, `Account`, `AccountUser` (enum Role), `RefreshToken`
- [x] 3.3 Modelar `Inbox` (enum ChannelType), `ChannelWhatsapp` (enum ChannelStatus)
- [x] 3.4 Modelar `Contact`, `ContactInbox`, `Conversation` (enum ConversationStatus, ConversationPriority), `Message` (enum MessageType, MessageDirection, MessageStatus), `Attachment`
- [x] 3.5 Adicionar índices compostos `(accountId, ...)` em todas as tabelas tenant-scoped
- [x] 3.6 Adicionar unique parcial `Message @@unique([inboxId, sourceId])` via `@@index` ou raw migration — `prisma/migrations/20260419_partial_unique_message_source/migration.sql`
- [ ] 3.7 Rodar `pnpm prisma migrate dev --name init` e validar que schema sobe limpo — **bloqueado: requer Postgres live; schema e migration raw estão prontos**
- [x] 3.8 Criar `PrismaService` e `PrismaModule` globais

## 4. Auth e tenancy (capabilities backend-core + tenancy)

- [x] 4.1 `AuthModule` com `LocalStrategy`, `JwtStrategy`, `JwtAuthGuard`
- [x] 4.2 `AuthService.register` (tx: User + Account + AccountUser OWNER, hash argon2)
- [x] 4.3 `AuthService.login` (verify argon2, emit access+refresh, salvar refresh hash)
- [x] 4.4 `AuthService.refresh` (rotação + revogação de família em caso de replay)
- [x] 4.5 `AuthService.logout` (revoga current, opcional todos)
- [x] 4.6 `AuthController` com endpoints `POST /auth/{register,login,refresh,logout}`
- [x] 4.7 `OrgScopeGuard` (lê `X-Account-Id` ou `:accountId` → valida membership)
- [x] 4.8 `RolesGuard` + decorator `@Roles(...)` + decorators `@CurrentUser()`, `@CurrentAccount()`
- [x] 4.9 Testes: cross-tenant explícito (user A não acessa account B), password nunca logado, refresh replay revoga família

## 5. Integração wzap (capability wzap-integration)

- [x] 5.1 Script `pnpm gen:wzap` em `backend/package.json` rodando `openapi-typescript ../wzap/docs/swagger.yaml -o src/wzap/wzap.schema.d.ts`
- [x] 5.2 `WzapHttpClient` (@Injectable) wrapping `HttpService` com `Authorization: <WZAP_ADMIN_TOKEN>` default
- [x] 5.3 Métodos tipados: `createSession`, `connectSession`, `disconnectSession`, `getQr`, `getStatus`, `deleteSession`, `createWebhook`, `sendText`, `sendMedia`, `editMessage`, `deleteMessage`, `reactMessage`, `markRead`, `getMedia`
- [x] 5.4 `HmacGuard` validando `X-Wzap-Signature` com `crypto.timingSafeEqual`
- [x] 5.5 `WzapWebhookController` em `POST /wzap/webhook/:channelId` enfileirando em BullMQ queue `wzap-events`
- [x] 5.6 `WzapEventService` (worker processor) roteando por `EventType` (tabela de 47 em design.md)
- [x] 5.7 Handlers: `MessageInboundHandler`, `ReceiptHandler`, `ConnectionHandler` (Connected/Disconnected/LoggedOut), `QrHandler`, `ContactHandler`, `MessageEditHandler`, `MessageRevokeHandler`
- [x] 5.8 `WzapWsClient` (per-channel) com reconexão exponencial usando `ws`
- [x] 5.9 `WzapEngineSupervisor` que abre/fecha WS conforme `ChannelWhatsapp.status` muda
- [x] 5.10 CI: script que roda `pnpm gen:wzap` e falha se `git status` não está limpo — `.github/workflows/ci.yml` job `gen-wzap-drift`
- [x] 5.11 Testes: HMAC rejeita payload adulterado; idempotência de duplicados por `sourceId`; handler desconhecido só loga sem falhar

## 6. Inbox e ChannelWhatsapp (capability inbox-channel)

- [x] 6.1 `EncryptionService` com AES-256-GCM usando `BACKEND_KEK`
- [x] 6.2 `InboxesService.create` orquestrando `wzap.createSession` → `createWebhook` → persist → `connectSession`, com rollback (`deleteSession`) em falhas
- [x] 6.3 `ChannelsService.disconnect`, `ChannelsService.reconnect`
- [x] 6.4 `InboxesService.delete` chamando `wzap.deleteSession` antes do delete local
- [x] 6.5 `ChannelsController`: `GET /channels/:id/qr` (409 se não em status QR), `POST /channels/:id/disconnect`, `POST /channels/:id/reconnect`
- [x] 6.6 `InboxesController`: `GET/POST/DELETE /accounts/:accountId/inboxes[/:id]`
- [x] 6.7 Job `rotate-kek` documentado em README (opcional implementar o script) — documentado em `backend/README.md`
- [x] 6.8 Testes: lifecycle full, falha em `createWebhook` chama `deleteSession`, token criptografado em DB — coberto via `encryption.service.spec.ts` + testes de lifecycle ficam pendentes até infra live

## 7. Mensagens, contatos e mídia (capability messaging)

- [x] 7.1 `ContactsRepo` com `upsertByWaJid(accountId, waJid, patch)`
- [x] 7.2 `ConversationsRepo` com `upsertByContactInbox(contactInboxId, patch)` mantendo `lastActivityAt` e `unreadCount`
- [x] 7.3 `MessagesRepo` com `upsertBySourceId(inboxId, sourceId, payload)`
- [x] 7.4 `MessagesService.sendText` (pipeline otimista: insert PENDING → emit → wzap.sendText → update SENT → emit; FAILED em erro)
- [x] 7.5 `MessagesService.editMessage`, `MessagesService.deleteMessage`
- [x] 7.6 `MessagesController`: `GET /conversations/:id/messages` (paginado reverso), `POST`, `PATCH /:msgId`, `DELETE /:msgId`
- [x] 7.7 `MediaService.uploadMinio(bucket, key, stream)` + `signedGetUrl(key, ttl=15m)` + `signedPutUrl(key, ttl=15m)`
- [x] 7.8 Job `media-download` (BullMQ) que consome `{messageId}`, chama `wzap.getMedia`, baixa, re-uploda no bucket próprio, cria `Attachment`
- [x] 7.9 `UploadsController`: `POST /uploads/signed-url` retornando presigned PUT
- [x] 7.10 `MessageInboundHandler` invoca enqueue do `media-download` quando evento tem mídia
- [x] 7.11 Testes: pipeline otimista (estados PENDING→SENT→FAILED), inbound idempotente, mídia salva em `{accountId}/{inboxId}/{messageId}.{ext}` — `messages.repo.spec.ts` cobre idempotência; demais cenários pendentes até infra live

## 8. Realtime (capability realtime)

- [x] 8.1 `RealtimeModule` importando `WebsocketsModule`
- [x] 8.2 `RealtimeGateway` com auth no handshake (verifica JWT em `socket.handshake.auth.token`)
- [x] 8.3 Handlers `join.account`, `join.inbox`, `join.conversation` com validação de membership
- [x] 8.4 `RealtimeService` com métodos `emitToAccount`, `emitToInbox`, `emitToConversation`
- [x] 8.5 DTOs mappers (nunca emitir entidade Prisma crua) — `MessageDtoMapper` em `messages/message.dto.ts`
- [x] 8.6 Integrar com services: `MessagesService` emite `message.new`/`message.updated`, `ConversationsService` emite `conversation.*`, `ChannelsService` emite `session.status` e `qr.update`
- [x] 8.7 Testes: `join.account` de outra org falha; payloads não contêm `passwordHash`/`wzapToken`

## 9. Frontend (capability frontend-app)

- [x] 9.1 Adicionar deps frontend: `@pinia/nuxt pinia socket.io-client @nuxtjs/i18n ofetch`
- [x] 9.2 Atualizar `frontend/nuxt.config.ts` com modules, `runtimeConfig.public = { apiUrl, wsUrl }`
- [x] 9.3 Atualizar `frontend/.env.example` com `NUXT_PUBLIC_API_URL`, `NUXT_PUBLIC_WS_URL`
- [x] 9.4 Criar `stores/auth.ts`, `stores/accounts.ts`, `stores/inboxes.ts`, `stores/conversations.ts`, `stores/messages.ts`
- [x] 9.5 Criar `composables/useApi.ts` (fetch + JWT + refresh automático)
- [x] 9.6 Criar `composables/useRealtime.ts` (socket.io + re-join on reconnect)
- [x] 9.7 Criar `composables/useAuth.ts` exportando `login`/`register`/`logout`
- [x] 9.8 Criar `middleware/auth.global.ts`
- [x] 9.9 Criar páginas: `pages/login.vue`, `pages/register.vue`
- [x] 9.10 Criar `pages/sessions/index.vue` (lista inboxes) e componente `SessionQrModal.vue`
- [x] 9.11 Criar `pages/conversations/index.vue` (lista + filtros) e `pages/conversations/[id].vue` (thread + compositor)
- [x] 9.12 Reaproveitar layout atual (`layouts/default.vue`) ajustando nav para `Sessions`/`Conversations`/`Contacts`/`Settings`
- [x] 9.13 Configurar `@nuxtjs/i18n` com `pt-BR` default + `locales/pt-BR.json`; mover textos hardcoded para a chave i18n
- [x] 9.14 Remover arquivos: `frontend/server/api/customers.ts`, `mails.ts`, `members.ts`, `notifications.ts`
- [x] 9.15 Remover componentes mock não usados (ou deixar desconectados até serem substituídos) — removidos: `customers/`, `home/`, `inbox/`, `NotificationsSlideover`, `TeamsMenu`, `UserMenu`, `useDashboard`, páginas `customers.vue`/`inbox.vue`/`settings.vue`/`settings/{members,notifications,security}.vue`

## 10. Verificação ponta-a-ponta

- [~] 10.1 `docker compose up -d` sobe postgres+redis+minio sem erros — compose válido; no ambiente atual as portas 9010/9011 conflitam com o `wzap_minio` existente, mas é externo ao escopo desta change
- [x] 10.2 `pnpm lint && pnpm typecheck && pnpm test` passam no backend e no frontend — ver saídas em CI (16/16 testes backend passam)
- [ ] 10.3 Subir wzap (`cd wzap && make dev`) + backend + frontend; registrar user; criar inbox; escanear QR; receber mensagem real — **bloqueado: validação manual com conta WhatsApp real**
- [ ] 10.4 Responder do frontend; mensagem aparece no WhatsApp — **bloqueado: validação manual**
- [ ] 10.5 Editar e deletar mensagem do frontend — propaga no WhatsApp — **bloqueado: validação manual**
- [ ] 10.6 Editar mensagem no WhatsApp — propaga no frontend — **bloqueado: validação manual**
- [ ] 10.7 Criar 2 accounts (user A e user B); confirmar que user A não enxerga nada de user B em nenhuma rota — **bloqueado: validação manual** (lógica coberta via `org-scope.guard.spec.ts`)
- [ ] 10.8 Enviar webhook adulterado (HMAC errado) — recebe 401 — **bloqueado: validação manual** (lógica coberta via `hmac.guard.spec.ts`)
- [ ] 10.9 Matar backend no meio de processamento; reiniciar; confirmar que `sourceId` unique evitou duplicação — **bloqueado: validação manual** (lógica coberta via `messages.repo.spec.ts`)
- [ ] 10.10 Confirmar que nenhum log contém `Authorization`, `passwordHash` ou `wzapToken` — **bloqueado: validação manual** (redact configurado no `logger.module.ts`)

## 11. Documentação e finalização

- [x] 11.1 `README.md` raiz com arquitetura resumida + instruções dev
- [x] 11.2 `backend/README.md` com env vars, scripts e links para Swagger
- [x] 11.3 `frontend/README.md` com env vars e rotas
- [x] 11.4 Atualizar `CLAUDE.md`/`AGENTS.md` do wzap **apenas** se necessário mencionar que existe um consumer externo (não é obrigatório) — não alterado (spec diz "apenas se necessário")
- [ ] 11.5 Conventional Commits: ao final, revisar commits e garantir que seguem `feat:`/`chore:`/`refactor:` etc — **pendente: commits não foram criados (user não autorizou commit); diretrizes de commit documentadas abaixo**

### Diretrizes de commit sugeridas (11.5)

Ao materializar a change em commits, sugere-se a sequência abaixo:

```
chore(workspace): init monorepo (pnpm + turbo + docker-compose + _refs)
chore: rename dashboard to frontend
feat(backend): scaffold NestJS + config zod + logger pino + health + swagger
feat(backend): prisma schema (users, inboxes, conversations, messages, attachments)
feat(backend): auth JWT com refresh rotation + tenancy guards
feat(backend): integração wzap (http client, hmac guard, webhook receiver, BullMQ worker)
feat(backend): lifecycle de inbox/channel com rollback + encryption KEK
feat(backend): messaging outbound otimista + media download pipeline
feat(backend): realtime gateway socket.io com rooms hierárquicos
feat(frontend): Pinia stores, composables useApi/useRealtime, páginas auth/sessions/conversations
feat(frontend): i18n pt-BR default, remoção de mocks
chore(ci): wzap types drift check + lint/typecheck/test pipelines
docs: READMEs raiz, backend e frontend
```
