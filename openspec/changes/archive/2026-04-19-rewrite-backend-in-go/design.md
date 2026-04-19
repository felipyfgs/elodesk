## Context

O `backend/` atual é NestJS/TS e foi implementado como cliente do `wzap/`: há um `WzapHttpClient`, um `WzapWsClient`, um supervisor WS, e o domínio inclui `ChannelWhatsapp` com `wzapSessionId`/`wzapToken`. Isso viola o princípio explicitado pelo usuário ("total desacoplamento") e duplica responsabilidade com o `wzap/` (que já oferece integração Chatwoot pronta em `wzap/internal/integrations/chatwoot/`). Queremos reescrever o backend em Go, idiomaticamente, com contrato Channel::Api (Chatwoot) — o wzap conecta sem mudar.

O `wzap/` é a referência de estilo: Go 1.22 + Fiber v2 + pgx v5 + migrations SQL via `//go:embed` + NATS JetStream + MinIO + gorilla/websocket + zerolog. Copiaremos a maioria dessas escolhas com duas exceções justificadas: (1) sem NATS — Redis + asynq bastam pro MVP de filas (wzap usa NATS por fan-out de eventos; nosso backend não tem esse requisito ainda); (2) Fiber é o HTTP router já escolhido no wzap, manter simplifica PRs cruzados.

## Goals / Non-Goals

**Goals:**
- Backend Go idiomático, pacote único, binário único, footprint mínimo.
- Contrato externo 100% compatível com Chatwoot (subset que o wzap já consome).
- Zero regressão funcional vs. backend TS atual no que diz respeito ao frontend.
- Consistência de estilo com `wzap/` (padrão handler→service→repo, logger com `component=`, repos com `scanXxx` + sentinel errors, validação via `go-playground/validator/v10`).
- Paridade de observabilidade: `/health`, Swagger em `/docs`, logs estruturados com redact.

**Non-Goals:**
- Substituir o wzap, reescrever sua lógica, ou adicionar conectividade direta ao `whatsmeow`.
- Migrar dados do backend NestJS (não há dados em produção).
- Suportar canais além do "api" (Chatwoot-compatível) no MVP. `channelType` fica como rótulo descritivo (whatsapp/sms/email/...), mas o protocolo do canal é sempre API.
- Reimplementar Socket.IO em Go. Usamos WebSocket nativo (`gorilla/websocket`) — sem fallback long-polling.
- Criar cliente TS auto-gerado do OpenAPI pro frontend. Tipos declarados manualmente.

## Decisions

### D1 — Fiber v2 em vez de chi ou net/http puro

**Escolhido:** Fiber v2.

**Porquê:** o wzap já usa Fiber (`wzap/internal/server/router.go`); reuso de idiomas (middleware chain, `fiber.Ctx`, `fiber.Map`), compatibilidade de utilitários (`dto.SuccessResp`/`ErrorResp` podem ser copiados com ajuste). Alternativas: chi (lib padrão, mais enxuto, mas teríamos que reinventar utilitários já maduros no wzap), net/http puro (inviável pra velocidade de dev).

### D2 — pgx v5 direto, sem ORM

**Escolhido:** pgx v5 + SQL puro + scanners.

**Porquê:** padrão do wzap. Prisma ia acoplar a um runtime Node embutido (não existe em Go) e sqlc/ent/gorm introduzem camadas que o time não usa no wzap. Repos seguem o padrão `wzap/internal/repo/*_repo.go`: colunas em constantes de pacote, `scanXxx(scanner, &m)` com interface local `xxxScanner` pra reusar em `pgx.Row` e `pgx.Rows`, erros sentinel double-wrapped (`fmt.Errorf("%w: %w", ErrNotFound, err)`).

**Alternativas:** sqlc (gera typed queries — ótima DX, mas adiciona um codegen step que o wzap evita), ent (ORM pesado), gorm (mágico demais). Rejeitadas para consistência com o wzap.

### D3 — Migrations SQL embutidas via //go:embed

**Escolhido:** `migrations/*.sql` + `//go:embed` + função `RunMigrations(pool)` executando em ordem alfabética.

**Porquê:** idêntico ao wzap (`wzap/migrations/`). Zero dependência externa (goose/migrate). Versionamento via número sequencial no prefixo (`0001_init.sql`, `0002_channel_api.sql`, ...). Track via tabela `schema_migrations(version text primary key, applied_at timestamp)`.

**Alternativas:** goose (popular mas adiciona binário/CLI), golang-migrate (similar), Atlas (overkill). Rejeitadas.

### D4 — asynq para filas em Redis

**Escolhido:** `hibiken/asynq`.

**Porquê:** queue library Go mature com retries exponenciais, priorities, scheduling, UI opcional. Mapeia 1:1 os jobs do BullMQ (`provider-webhooks`, `media-download`). Sintaxe familiar ao BullMQ.

**Alternativas:** River (PostgreSQL-based — obrigaria outra dep), queue própria em Redis (reinventar roda), Machinery (menos mantida). Rejeitadas.

### D5 — JWT com golang-jwt + refresh rotation em Postgres

**Escolhido:** `golang-jwt/jwt/v5` para access token (HS256, 15min), refresh token como `crypto/rand` 48 bytes base64url salvo como SHA-256 hash na tabela `refresh_tokens` (30 dias TTL, `family_id` pra revogar cadeia em replay).

**Porquê:** mesmo modelo do backend TS atual — migração conceitual limpa. Argon2id via `alexedwards/argon2id` para hash de senha.

**Alternativas:** Paseto (mais moderno, menos ecossistema), sessões server-side cookie (precisaria CSRF + storage), OIDC próprio (over-engineering pro MVP).

### D6 — WebSocket gorilla/websocket em vez de Socket.IO

**Escolhido:** `gorilla/websocket` puro. Gateway `/realtime` com JWT no `Sec-WebSocket-Protocol` ou query (`?token=...`). Mensagens JSON `{type: "join.account"|"join.inbox"|"join.conversation"|"event", payload}`. Rooms gerenciadas por `map[string]map[*Client]struct{}` protegidos por `sync.RWMutex` (padrão do hub em `wzap/internal/websocket/`).

**Porquê:** Socket.IO adiciona fallback long-polling + protocolo próprio + heartbeats — conforto que não justifica o custo server-side em Go. Frontend troca `socket.io-client` por `useWebSocket` do VueUse.

**Alternativas:** nhooyr/websocket (mais moderna, API mais simples, licença boa — poderíamos usar também; gorilla é incumbent no wzap então mantém consistência), Centrifugo (processo separado — over-engineering), SSE (uni-direcional, força HTTP pra ack).

### D7 — Logger zerolog com redact manual

**Escolhido:** `rs/zerolog` com helper `logger.WithComponent("<module>")` que retorna um logger com campo `component` já preenchido. Redact manual de headers sensíveis antes de logar (função `redactHeaders(h)`).

**Porquê:** padrão do wzap (zerolog singleton, `component=` em toda linha). pino é comparável em features mas não é Go. Alternativas: slog (stdlib, bom o bastante; poderíamos usar também, mas zerolog mantém consistência com wzap).

### D8 — Validação de request: go-playground/validator + helper

**Escolhido:** struct tags `validate:"required,email"` + helper `parseAndValidate(c *fiber.Ctx, req any) error` (copia `wzap/internal/integrations/chatwoot/handler.go:validateReq`).

**Porquê:** padrão do wzap. DTOs ficam em `internal/dto/` com tags de validação.

### D9 — Swagger via swaggo/swag

**Escolhido:** anotações `// @Summary` nos handlers + `swag init` gera `docs/swagger.yaml` + `fiber-swagger` serve em `/docs`.

**Porquê:** wzap usa isso. Codegen simples, anotações inline.

### D10 — Multipart uploads para attachments

**Escolhido:** Fiber `c.FormFile()` + stream direto pro MinIO via `client.PutObject(ctx, bucket, key, reader, size, opts)`. Nunca carregar o payload inteiro em memória.

**Porquê:** o wzap sobe mídia para o Chatwoot via multipart; nosso backend precisa receber multipart do wzap. Limite de 256 MB (match com `maxMediaBytes` do wzap).

### D11 — Contrato outbound webhook

**Escolhido:** POST JSON em `channel_api.webhook_url` com headers:

```
Content-Type: application/json
X-Chatwoot-Hmac-Sha256: <hex(HMAC-SHA256(body, channel_api.hmac_token))>
```

Payload:

```json
{
  "event_type": "message_created",
  "message": { "id": 123, "content": "...", "message_type": "outgoing", "source_id": null, "content_attributes": {...} },
  "conversation": { "id": 456, "contact_inbox": { "source_id": "5511999..." } },
  "account": { "id": "uuid", "name": "Acme" }
}
```

Retry via asynq com backoff exponencial `1s, 5s, 30s, 2m, 10m` (igual wzap `wzap/internal/webhook/dispatcher.go`).

### D12 — Organização de pacotes

```
backend/
  cmd/backend/main.go              # entrypoint único
  internal/
    config/                         # env loading + validação
    logger/                         # zerolog singleton + WithComponent
    database/                       # pgx pool + RunMigrations
    migrations/                     # *.sql embutidos via //go:embed
    server/
      router.go                     # registro de rotas + DI manual (como wzap)
      errors.go                     # sentinel wrappers, fiber error handler
    middleware/
      jwt_auth.go                   # valida Bearer
      api_token.go                  # valida api_access_token (providers)
      hmac.go                       # valida X-Chatwoot-Hmac-Sha256 inbound (opcional)
      org_scope.go                  # preenche accountId no Locals
      roles.go                      # OWNER/ADMIN/AGENT
    handler/
      auth_handler.go               # register/login/refresh/logout
      inbox_handler.go              # provisionamento interno (JWT)
      contact_handler.go            # Channel::Api: contacts
      conversation_handler.go       # Channel::Api: conversations
      message_handler.go            # Channel::Api: messages (JSON + multipart)
      upload_handler.go             # presigned URLs pro frontend
      realtime_handler.go           # WS upgrade + join/leave
      health_handler.go
    service/
      auth_service.go
      inbox_service.go
      contact_service.go
      conversation_service.go
      message_service.go
      outbound_webhook_service.go   # enfileira asynq task
      realtime_service.go           # broadcast pros rooms
    repo/
      user_repo.go
      account_repo.go
      refresh_token_repo.go
      inbox_repo.go
      channel_api_repo.go
      contact_repo.go
      conversation_repo.go
      message_repo.go
      attachment_repo.go
    model/                          # structs do domínio (sem validator tags)
    dto/                            # request/response (com validator tags)
    realtime/                       # hub gorilla/websocket
    webhook/
      outbound_processor.go         # asynq handler
    media/
      minio_client.go
      upload.go
    integrations/                   # futuro: adaptadores específicos (não usados no MVP)
  migrations/
    0001_init.sql                   # users, accounts, account_users, refresh_tokens
    0002_inbox_channel_api.sql      # inboxes, channels_api
    0003_messaging.sql              # contacts, contact_inboxes, conversations, messages, attachments
    0004_audit.sql                  # audit_events, partial unique em messages(inbox_id, source_id)
  docs/                             # swaggo output
  Dockerfile
  Makefile                          # dev/build/docs/tidy/test
  go.mod
  go.sum
  .env.example
  README.md
```

## Risks / Trade-offs

- **[Risk] Regressão no frontend ao trocar Socket.IO** → **Mitigation**: escrever o cliente WS nativo antes de cortar Socket.IO; rodar `/conversations` dev em paralelo durante transição; smoke test manual.
- **[Risk] `scanXxx` repetitivo em 9 repos** → **Mitigation**: aceitar boilerplate; wzap convive bem com ele, DX linear.
- **[Risk] asynq perde jobs em crash de Redis** → **Mitigation**: asynq persiste em Redis lists com ACK. Pra MVP é aceitável; pra prod futura, avaliar persistência Redis (AOF + replica).
- **[Risk] Multipart uploads estourando memória** → **Mitigation**: `c.FormFile` + stream direto pro MinIO (nunca buffer em memória); hard limit de 256 MB por upload.
- **[Risk] JWT leak em log do Fiber** → **Mitigation**: `redactHeaders` aplicado em middleware `logger.Request()`; proibido log direto de `c.Get("Authorization")`.
- **[Risk] Rollback difícil se o Go ficar inviável** → **Mitigation**: `backend/` NestJS permanece no histórico git; `git revert` da change recupera tudo em um commit.
- **[Trade-off] Consistência com wzap > modernidade isolada**: escolhemos zerolog/Fiber/pgx/gorilla em vez de slog/chi/sqlc/nhooyr. Razão: PRs cruzados, ops uniforme, onboarding único.
- **[Trade-off] WebSocket sem fallback** — aceito no MVP; proxy/LB modernos suportam WS nativo.

## Migration Plan

1. **Fase 0 (desta change):** deletar `backend/` TS inteiro em um único commit `feat(backend): rewrite in Go — remove NestJS`. Impacto: nenhum cliente em produção usa o backend ainda.
2. **Fase 1 — scaffolding Go:** `cmd/backend/main.go` + `internal/config` + `internal/logger` + `/health` + Swagger stub + Dockerfile + Makefile + `.env.example` + migrations vazias. Rodar `go run cmd/backend/main.go` e ver `GET /health` retornando ok.
3. **Fase 2 — DB + auth:** migrations `0001`/`0002`/`0003`/`0004`, `repo/user_repo.go`, `repo/account_repo.go`, `repo/refresh_token_repo.go`, `auth_service`, `auth_handler`. Registrar/login/refresh/logout funcionando com Postman.
4. **Fase 3 — Tenancy:** middleware `org_scope`, `roles`, helpers `CurrentUser`/`CurrentAccount` via `fiber.Locals`. Testes unitários por middleware.
5. **Fase 4 — Channel::Api inbound:** handlers `contact_handler`, `conversation_handler`, `message_handler` (JSON + multipart), middleware `api_token`. Teste de integração contra Postgres de teste.
6. **Fase 5 — Outbound webhooks:** `asynq` client + worker, `outbound_webhook_service` emitindo `message_created`/`message_updated`/`conversation_status_changed`. Testes com servidor HTTP de mentira.
7. **Fase 6 — Realtime:** hub `realtime/hub.go`, handler `realtime_handler.go` upgrade WS, service `realtime_service` broadcast.
8. **Fase 7 — Media:** `minio_client.go`, `upload_handler.go`, integração com `message_handler` multipart.
9. **Fase 8 — Frontend cutover:** reescrever `useRealtime.ts` com `@vueuse/useWebSocket`; renomear/limpar store `inboxes`; adaptar `/sessions` para Channel::Api. Remover `SessionQrModal.vue` e `socket.io-client`.
10. **Fase 9 — CI + docs:** workflow `go test -race ./...` + `golangci-lint run ./...`; `backend/README.md`; atualizar `README.md` raiz.
11. **Fase 10 — Validação ponta-a-ponta:** subir wzap com `CHATWOOT_URL=http://localhost:3001` apontando pro backend Go; registrar user; criar inbox; wzap posta contact+message; agente responde; ver payload do webhook outbound no log do wzap; conferir que a mensagem chega no WA real.

**Rollback:** `git revert` da change aborta tudo; `backend/` TS volta intacto.

## Open Questions

- Nome do binário: `backend` ou algo mais específico (`wzap-hub`/`wz-backend`)? Proponho `backend` por simplicidade — mesma convenção do wzap (`cmd/wzap/main.go` → `bin/wzap`).
- WebSocket auth via query `?token=JWT` (visível em logs/proxies) vs header `Sec-WebSocket-Protocol: bearer,<token>` (padrão emergente). Proponho header; fallback query só pra dev.
- Swagger na raiz `/docs` ou sob `/api/v1/docs`? Proponho `/docs` (como wzap).
- CORS: permitir qualquer origem em dev, whitelist em prod via env `CORS_ORIGINS`? Proponho sim, lista separada por vírgula.
- Primeiro admin: criar automaticamente no `make seed` ou só via `/auth/register` (self-service)? Proponho self-service — MVP sem admin global.
