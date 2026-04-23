## ADDED Requirements

### Requirement: CRUD de saved filters user-scoped

O backend SHALL expor endpoints REST pra gerenciar filtros salvos por user+account:

- `GET /api/v1/accounts/{aid}/custom_filters` — listar apenas os filtros do user atual. Query param `?filter_type=conversation|contact`.
- `POST /api/v1/accounts/{aid}/custom_filters` body:
  ```json
  {
    "name": "Urgentes sem atribuição",
    "filter_type": "conversation",
    "query": {
      "operator": "AND",
      "conditions": [
        {"attribute_key":"status","filter_operator":"equal_to","value":"open"},
        {"attribute_key":"priority","filter_operator":"equal_to","value":"urgent"},
        {"attribute_key":"assignee_id","filter_operator":"is_null","value":null}
      ]
    }
  }
  ```
- `PATCH /api/v1/accounts/{aid}/custom_filters/{id}`.
- `DELETE /api/v1/accounts/{aid}/custom_filters/{id}`.

Restrições:
- `filter_type` MUST ser `conversation` ou `contact`.
- `user_id` MUST ser gravado automaticamente a partir do JWT; usuário só vê/edita/deleta os próprios filtros.
- Máximo 1000 filtros por `(user_id, account_id)`; além disso retorna 400 `max_filters_reached`.
- Máximo 20 conditions no `query.conditions`.
- `query.operator` MUST ser `AND` ou `OR` (flat, sem aninhamento).
- Roles: qualquer `Agent+` pode criar seus próprios filtros.

#### Scenario: criar filtro salvo

- **WHEN** agent (user_id=10) envia POST com filter_type="conversation" e 3 conditions
- **THEN** retorna 201 com filtro criado; `user_id=10` gravado automaticamente

#### Scenario: listar retorna só filtros do user atual

- **WHEN** agent A envia GET; filtros de outros users existem
- **THEN** retorna apenas os filtros criados pelo agent A

#### Scenario: rejeitar query com operator aninhado

- **WHEN** agent envia POST com `query.conditions[0]` sendo outro objeto com `operator` e `conditions`
- **THEN** retorna 400 com erro `nested_operators_not_supported`

### Requirement: Aplicar filtro e retornar conversations matching

O backend SHALL expor endpoint `POST /api/v1/accounts/{aid}/conversations/filter` body `{query, page?, per_page?}` que:

- Traduz `query` pra SQL parametrizado usando **whitelist** de `attribute_key` e `filter_operator`.
- Retorna matching conversations paginadas (max 100 per_page, default 25).
- Obedece todo scope de account (sempre inclui `account_id = :aid` no WHERE).
- Timeout de 5s via `context.WithTimeout`; se estourar, retorna 504 com erro `filter_timeout`.

Whitelist de `attribute_key` pra conversations:
- Standard: `status`, `priority`, `assignee_id`, `team_id`, `contact_id`, `inbox_id`, `labels`, `created_at`, `updated_at`, `last_activity_at`.
- Custom: qualquer key de `custom_attribute_definitions` com `attribute_model='conversation'` na account.

Whitelist de `filter_operator`: `equal_to`, `not_equal_to`, `contains`, `starts_with`, `greater_than`, `less_than`, `in` (value é array), `between` (value é `[a, b]`), `is_null`, `is_not_null`.

Role: `Agent+`.

#### Scenario: aplicar filtro AND

- **WHEN** agent POST `/conversations/filter` com body `{query:{operator:"AND", conditions:[{attribute_key:"status",filter_operator:"equal_to",value:"open"},{attribute_key:"labels",filter_operator:"contains",value:"urgente"}]}}`
- **THEN** retorna conversations da account com status=open E que tenham a label "urgente", paginadas

#### Scenario: filtro em custom attribute

- **WHEN** agent aplica filter com `attribute_key="churn_risk"` (custom, definido na account) `filter_operator="greater_than"` `value=0.7`
- **THEN** backend traduz pra `WHERE (additional_attributes->>'churn_risk')::numeric > 0.7` (usando índice GIN)

#### Scenario: rejeitar attribute_key fora da whitelist

- **WHEN** agent aplica filter com `attribute_key="totally_random_key"` que não é standard nem custom
- **THEN** retorna 400 com erro `invalid_attribute_key` e lista as keys válidas

#### Scenario: rejeitar filter_operator não suportado

- **WHEN** agent passa `filter_operator="SQL_INJECTION"` ou similar
- **THEN** retorna 400 com erro `invalid_filter_operator`

### Requirement: Aplicar filtro e retornar contacts matching

O backend SHALL expor endpoint análogo `POST /api/v1/accounts/{aid}/contacts/filter` body `{query, page?, per_page?}`.

Whitelist de `attribute_key` pra contacts:
- Standard: `name`, `email`, `phone_number`, `identifier`, `blocked`, `last_activity_at`, `created_at`, `updated_at`.
- Custom: keys em `custom_attribute_definitions` com `attribute_model='contact'`.

Mesmo comportamento de paginação, timeout e whitelist de operators.

#### Scenario: buscar contacts por custom attribute

- **WHEN** agent POST `/contacts/filter` com `{query:{operator:"AND", conditions:[{attribute_key:"loyalty_tier",filter_operator:"equal_to",value:"gold"}]}}`
- **THEN** retorna contacts com `additional_attributes->>'loyalty_tier' = 'gold'`

### Requirement: Schema de custom_filters

A migration `0004_helpdesk_core.sql` MUST criar:

```sql
CREATE TABLE custom_filters (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  filter_type TEXT NOT NULL CHECK (filter_type IN ('conversation','contact')),
  query JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_custom_filters_user_account ON custom_filters(user_id, account_id);
```

#### Scenario: migration cria tabela

- **WHEN** migration `0004` roda
- **THEN** tabela `custom_filters` existe com CHECK em `filter_type` e index em `(user_id, account_id)`
