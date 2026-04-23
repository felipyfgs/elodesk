## ADDED Requirements

### Requirement: CRUD de teams account-scoped

O backend SHALL expor endpoints REST pra gerenciar teams por account:

- `GET /api/v1/accounts/{aid}/teams` — listar.
- `POST /api/v1/accounts/{aid}/teams` body `{name, description?, allow_auto_assign?}`.
- `PATCH /api/v1/accounts/{aid}/teams/{id}`.
- `DELETE /api/v1/accounts/{aid}/teams/{id}`.

Restrições:
- `name` MUST ser unique por account (case-insensitive).
- `allow_auto_assign` default `false` (usado pela Onda 2 em assignment policies).
- CRUD MUST exigir role `Admin` ou `Owner`.

#### Scenario: criar team

- **WHEN** admin envia `POST /api/v1/accounts/1/teams` com `{"name":"Suporte N1","description":"Primeiro atendimento"}`
- **THEN** retorna 201 com `{id, name:"suporte n1", description, allow_auto_assign:false, account_id:1, created_at, updated_at}`

#### Scenario: rejeitar nome duplicado

- **WHEN** admin cria team com `name` já existente na account
- **THEN** retorna 409 com erro `team_name_taken`

### Requirement: Gestão de membros do team

O backend SHALL permitir adicionar/remover agentes em teams:

- `GET /api/v1/accounts/{aid}/teams/{id}/team_members` — listar.
- `POST /api/v1/accounts/{aid}/teams/{id}/team_members` body `{user_ids: [number]}` — bulk add.
- `DELETE /api/v1/accounts/{aid}/teams/{id}/team_members` body `{user_ids: [number]}` — bulk remove.

Usuário adicionado MUST ter `account_users` row ativo na mesma account (integridade via service). Associação (`team_id, user_id`) MUST ser única.

#### Scenario: adicionar múltiplos membros em um request

- **WHEN** admin envia `POST /teams/5/team_members` com `{"user_ids":[10,11,12]}`
- **THEN** 3 rows em `team_members` são criadas; retorna 201 com a lista atualizada

#### Scenario: rejeitar user fora da account

- **WHEN** admin tenta adicionar user cujo `account_users` não existe ou está inativo na account
- **THEN** retorna 400 com erro `user_not_in_account` e nenhuma inserção é feita

### Requirement: Atribuir team a conversation

A tabela `conversations` MUST ganhar coluna `team_id BIGINT NULL` com FK pra `teams(id)` `ON DELETE SET NULL`.

Endpoint `POST /api/v1/accounts/{aid}/conversations/{id}/assignments` (definido em `backend-go-channels-api`) aceita `team_id` no body. Mudança MUST emitir realtime event `conversation.assignment_changed`.

Listagem de conversations MUST aceitar filtro `?team_id=<id>` e `?team_id=null` (desatribuídas).

#### Scenario: filtrar conversations por team

- **WHEN** agent envia `GET /api/v1/accounts/1/conversations?team_id=5`
- **THEN** retorna apenas conversations com `team_id=5` da account 1

#### Scenario: deletar team desatribui conversations

- **WHEN** admin envia `DELETE /api/v1/accounts/1/teams/5`
- **THEN** team é removido; conversations com `team_id=5` ficam com `team_id=NULL` (via `ON DELETE SET NULL`); clientes recebem broadcast de realtime

### Requirement: Schema de teams e team_members

A migration `0004_helpdesk_core.sql` MUST criar:

```sql
CREATE TABLE teams (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  allow_auto_assign BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (account_id, lower(name))
);

CREATE TABLE team_members (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (team_id, user_id)
);

ALTER TABLE conversations ADD COLUMN team_id BIGINT NULL REFERENCES teams(id) ON DELETE SET NULL;
CREATE INDEX idx_conversations_team ON conversations(team_id) WHERE team_id IS NOT NULL;
```

#### Scenario: migration aplicada com sucesso

- **WHEN** migration `0004_helpdesk_core.sql` roda
- **THEN** tabelas `teams`, `team_members` e coluna `conversations.team_id` existem; constraints e índices estão em vigor
