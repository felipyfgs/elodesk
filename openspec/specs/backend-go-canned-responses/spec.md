## ADDED Requirements

### Requirement: CRUD de canned responses

O backend SHALL expor endpoints REST pra gerenciar respostas rápidas (canned responses) por account:

- `GET /api/v1/accounts/{aid}/canned_responses` — listar todas as respostas da account, ordenadas por `short_code` ASC.
- `POST /api/v1/accounts/{aid}/canned_responses` body `{short_code: string, content: string}`.
- `PATCH /api/v1/accounts/{aid}/canned_responses/{id}`.
- `DELETE /api/v1/accounts/{aid}/canned_responses/{id}`.

Restrições:
- `short_code` MUST ser unique por account, lowercase, sem espaços, validator regex `^[a-z0-9][a-z0-9-_]{0,31}$`.
- `content` MUST ter no máximo 10.000 caracteres; pode conter markdown básico, mas o service NÃO processa; é passado literal pro composer.
- CRUD MUST exigir role `Admin+`. Leitura (GET) permitida pra `Agent+`.

#### Scenario: criar canned response

- **WHEN** admin envia `POST /api/v1/accounts/1/canned_responses` com `{"short_code":"greet-new","content":"Olá! Como posso ajudar hoje?"}`
- **THEN** retorna 201 com `{id, short_code:"greet-new", content, account_id:1, created_at, updated_at}`

#### Scenario: rejeitar short_code inválido

- **WHEN** admin envia short_code com espaço ou caractere especial (`Olá!`)
- **THEN** retorna 400 com erro de validação

#### Scenario: rejeitar short_code duplicado

- **WHEN** admin cria canned com `short_code` já existente na account
- **THEN** retorna 409 com erro `canned_short_code_taken`

### Requirement: Busca e listagem para o picker

O endpoint `GET /api/v1/accounts/{aid}/canned_responses` SHALL aceitar query param `?search=<term>` pra filtrar:

- Prefix match em `short_code` (priority 1)
- Substring match em `short_code` (priority 2)
- Substring match em `content` (priority 3)

Resultados ordenados por priority, depois por `short_code` ASC. Limite default 50, max 100 via `?limit=`.

Agent MUST conseguir listar e buscar (não só admin).

#### Scenario: buscar por prefix

- **WHEN** agent envia `GET /api/v1/accounts/1/canned_responses?search=gre`
- **THEN** retorna respostas com short_code começando em "gre" primeiro (ex.: "greet-new", "greet-ret"), depois respostas que contêm "gre" em outras posições

### Requirement: Schema de canned_responses

A migration `0004_helpdesk_core.sql` MUST criar:

```sql
CREATE TABLE canned_responses (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  short_code TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (account_id, short_code)
);
CREATE INDEX idx_canned_responses_account ON canned_responses(account_id);
```

#### Scenario: migration cria tabela

- **WHEN** migration `0004` é aplicada
- **THEN** tabela `canned_responses` existe com unique `(account_id, short_code)` e index em `account_id`
