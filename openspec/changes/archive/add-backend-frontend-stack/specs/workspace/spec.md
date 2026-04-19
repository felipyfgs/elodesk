## ADDED Requirements

### Requirement: Monorepo com pnpm workspace + Turbo

O repositório raiz (`/home/obsidian/dev/project/`) SHALL ser configurado como monorepo pnpm unificando apenas `backend/` e `frontend/`. O diretório `wzap/` e o diretório `_refs/` MUST permanecer fora do workspace.

#### Scenario: workspace inclui backend e frontend

- **WHEN** o desenvolvedor roda `pnpm install` na raiz
- **THEN** o comando resolve dependências de `backend/package.json` e `frontend/package.json`
- **AND** não tenta resolver nada dentro de `wzap/` ou `_refs/`

#### Scenario: Turbo coordena pipelines

- **WHEN** o desenvolvedor roda `pnpm turbo run dev`
- **THEN** Turbo inicia em paralelo o dev server do backend e do frontend
- **AND** hot reload de ambos funciona independentemente

#### Scenario: `_refs/` é ignorado pelo git

- **WHEN** o desenvolvedor clona Chatwoot ou Whaticket SaaS em `_refs/`
- **THEN** `git status` não lista esses diretórios como untracked
- **AND** `_refs/` aparece no `.gitignore` raiz

### Requirement: Dashboard renomeado para frontend preservando histórico

O diretório `dashboard/` SHALL ser renomeado para `frontend/` usando `git mv` numa única transação. Histórico de commits MUST ser preservado.

#### Scenario: renomeação preserva histórico

- **WHEN** o desenvolvedor executa `git log --follow frontend/app/app.vue`
- **THEN** o log mostra commits anteriores de `dashboard/app/app.vue`

#### Scenario: imports antigos são atualizados

- **WHEN** a renomeação for commitada
- **THEN** nenhum arquivo do monorepo referencia o caminho `dashboard/` (com exceção de `_refs/`)

### Requirement: Docker Compose de desenvolvimento

A raiz SHALL expor um `docker-compose.yml` que levanta apenas a infraestrutura de dados necessária ao backend (Postgres 16, Redis 7, MinIO). O backend e o frontend MUST rodar fora do compose (diretamente via `pnpm dev`).

#### Scenario: subir infra de dev

- **WHEN** o desenvolvedor executa `docker compose up -d` na raiz
- **THEN** Postgres em `localhost:5432`, Redis em `localhost:6379` e MinIO em `localhost:9010` ficam disponíveis
- **AND** nenhum container do backend ou do frontend é criado pelo compose

#### Scenario: volumes persistem dados

- **WHEN** o desenvolvedor executa `docker compose down` e `docker compose up -d`
- **THEN** bancos e buckets persistem entre restarts (exceto quando for usado `-v`)
