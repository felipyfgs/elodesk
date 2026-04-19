## ADDED Requirements

### Requirement: Scaffolding Go idiomático em backend/

O diretório `backend/` SHALL conter um projeto Go 1.22 com layout espelhando o do `wzap/`:

- `cmd/backend/main.go` — entrypoint único
- `internal/config/` — carregamento e validação de env
- `internal/logger/` — zerolog singleton com helper `WithComponent(name)`
- `internal/database/` — pool pgx v5 + `RunMigrations(ctx, pool)` executando `migrations/*.sql` embutidos via `//go:embed`
- `internal/server/` — `router.go` com registro de rotas + DI manual; `errors.go` com fiber error handler central
- `internal/middleware/` — `jwt_auth.go`, `api_token.go`, `hmac.go`, `org_scope.go`, `roles.go`
- `internal/handler/`, `internal/service/`, `internal/repo/`, `internal/model/`, `internal/dto/`
- `internal/realtime/`, `internal/webhook/`, `internal/media/`
- `migrations/*.sql`
- `Dockerfile`, `Makefile`, `go.mod`, `go.sum`, `.env.example`, `README.md`

#### Scenario: build produz binário único

- **WHEN** o desenvolvedor executa `cd backend && make build`
- **THEN** é gerado `bin/backend` (binário estático, CGO_ENABLED=0)
- **AND** `./bin/backend` sobe o servidor na porta configurada

#### Scenario: estrutura espelha wzap

- **WHEN** inspeção de `backend/` e `wzap/`
- **THEN** a organização de pacotes em `internal/` é equivalente (handler/service/repo/middleware/dto/model)

### Requirement: Config via env com validação

O backend SHALL carregar variáveis de ambiente (`.env` opcional em dev) e validar no startup que TODAS as obrigatórias estão presentes e bem formadas:

- `PORT` (int, default 3001)
- `NODE_ENV` / `GO_ENV` (`development`|`test`|`production`, default `development`)
- `LOG_LEVEL` (`trace`|`debug`|`info`|`warn`|`error`, default `info`)
- `DATABASE_URL` (PostgreSQL URI)
- `REDIS_URL` (Redis URI)
- `JWT_SECRET` (≥ 32 caracteres)
- `JWT_ACCESS_TTL` (default `15m`), `JWT_REFRESH_TTL` (default `30d`)
- `BACKEND_KEK` (base64 de 32 bytes)
- `MINIO_ENDPOINT`, `MINIO_PORT`, `MINIO_USE_SSL`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`
- `API_URL` (URL pública do próprio backend)
- `CORS_ORIGINS` (opcional, lista separada por vírgula)

#### Scenario: falta env obrigatória

- **WHEN** o backend é iniciado sem `JWT_SECRET`
- **THEN** o processo aborta no startup com mensagem listando as variáveis faltando
- **AND** o servidor HTTP não chega a abrir

#### Scenario: env inválida

- **WHEN** `BACKEND_KEK` decodifica para menos de 32 bytes
- **THEN** o processo aborta com mensagem específica

### Requirement: Logger estruturado com componente

Todo log emitido pelo backend SHALL:

- Ser JSON estruturado via `zerolog`
- Começar com campo `component=<nome do módulo>` (ex: `component=auth`, `component=outbound-webhook`)
- Redatar antes de emitir: `Authorization`, `Cookie`, `X-Chatwoot-Hmac-Sha256`, `password`, `passwordHash`, `token`, `hmacToken`, `apiToken`, `refreshToken`, `accessToken`

#### Scenario: log de request não vaza token

- **WHEN** uma request chega com header `Authorization: Bearer abc123`
- **THEN** o log da request mostra `authorization: [REDACTED]`

#### Scenario: log de service usa component

- **WHEN** `AuthService.Login` emite log de erro
- **THEN** o log tem `component=auth` e nível `warn` ou `error`

### Requirement: Healthcheck agregado

O backend SHALL expor `GET /health` respondendo JSON `{status, db, redis}` indicando o estado de cada dependência.

#### Scenario: infra up

- **WHEN** Postgres e Redis estão acessíveis
- **THEN** `GET /health` retorna 200 com `{"status":"ok","db":"ok","redis":"ok"}`

#### Scenario: dependência fora

- **WHEN** Redis está inacessível
- **THEN** `GET /health` retorna 200 com `{"status":"degraded","db":"ok","redis":"error"}`

### Requirement: Swagger via swaggo em /docs

O backend SHALL expor Swagger UI em `/docs` com anotações `swag` nos handlers. `make docs` regenera `docs/swagger.yaml`.

#### Scenario: docs disponível

- **WHEN** o backend está no ar
- **THEN** `GET /docs/` renderiza a UI Swagger listando todas as rotas
