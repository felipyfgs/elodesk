## ADDED Requirements

### Requirement: CRUD de definições de custom attribute

O backend SHALL expor endpoints REST pra definir atributos custom por account:

- `GET /api/v1/accounts/{aid}/custom_attribute_definitions` — listar. Query params: `?attribute_model=contact|conversation`.
- `POST /api/v1/accounts/{aid}/custom_attribute_definitions` body:
  ```json
  {
    "attribute_key": "loyalty_tier",
    "attribute_display_name": "Tier de Lealdade",
    "attribute_display_type": "list",
    "attribute_model": "contact",
    "attribute_values": ["bronze","silver","gold"],
    "attribute_description": "Classificação do cliente",
    "regex_pattern": null,
    "default_value": null
  }
  ```
- `PATCH /api/v1/accounts/{aid}/custom_attribute_definitions/{id}`.
- `DELETE /api/v1/accounts/{aid}/custom_attribute_definitions/{id}`.

Restrições:
- `attribute_key` MUST ser unique por `(account_id, attribute_model)`, lowercase, regex `^[a-z][a-z0-9_]{0,62}$`.
- `attribute_display_type` MUST ser um de: `text`, `number`, `currency`, `percent`, `link`, `date`, `list`, `checkbox`.
- `attribute_model` MUST ser `contact` ou `conversation`.
- `attribute_values` (jsonb array de strings) obrigatório APENAS quando `attribute_display_type == 'list'`.
- `regex_pattern` opcional, aplicado a valores do tipo `text`.
- `attribute_key` MUST NÃO colidir com campos standard (lista interna no service):
  - Contact: `id, name, email, phone_number, identifier, created_at, updated_at, blocked, last_activity_at, additional_attributes, account_id`.
  - Conversation: `id, status, assignee_id, team_id, contact_id, contact_inboxes_id, inbox_id, display_id, uuid, created_at, updated_at, last_activity_at, priority, additional_attributes, account_id`.
- CRUD MUST exigir role `Admin+`.

#### Scenario: criar custom attribute tipo list

- **WHEN** admin envia `POST /custom_attribute_definitions` com tipo list e 3 valores
- **THEN** retorna 201 com a definition criada; `attribute_values` preservado como array

#### Scenario: rejeitar key reservado

- **WHEN** admin tenta criar definition com `attribute_key="email"` e `attribute_model="contact"`
- **THEN** retorna 400 com erro `attribute_key_reserved`

#### Scenario: rejeitar list sem valores

- **WHEN** admin cria definition com `attribute_display_type="list"` e `attribute_values` vazio ou null
- **THEN** retorna 400 com erro `list_values_required`

### Requirement: Setar e remover valores em contact/conversation

O backend SHALL expor endpoints pra manipular valores em contacts e conversations:

- `POST /api/v1/accounts/{aid}/contacts/{id}/custom_attributes` body `{[key]: value, ...}` — merge JSONB.
- `DELETE /api/v1/accounts/{aid}/contacts/{id}/custom_attributes` body `{keys: [string]}` — remove chaves.
- `POST /api/v1/accounts/{aid}/conversations/{id}/custom_attributes` body `{[key]: value, ...}`.
- `DELETE /api/v1/accounts/{aid}/conversations/{id}/custom_attributes` body `{keys}`.

Valores são persistidos em `contacts.additional_attributes` (jsonb) e `conversations.additional_attributes` (jsonb) — colunas já existem desde o baseline.

Validação no service antes do UPDATE:
- Cada key do body MUST ter uma `custom_attribute_definition` ativa pro `attribute_model` correto; keys desconhecidas retornam 400.
- Tipo do valor MUST corresponder a `attribute_display_type`:
  - `text`: string; respeita `regex_pattern` se presente.
  - `number`, `currency`, `percent`: number.
  - `link`: string com URL válida.
  - `date`: string ISO-8601 ou unix timestamp (documentar).
  - `list`: string presente em `attribute_values`.
  - `checkbox`: boolean.

Setar valores: role `Agent+`.

#### Scenario: setar valor de list válido

- **WHEN** agent envia `POST /contacts/7/custom_attributes` com `{"loyalty_tier":"gold"}` e a definition existe com values `["bronze","silver","gold"]`
- **THEN** retorna 200; `contacts.additional_attributes` passa a conter `{"loyalty_tier":"gold"}` (merge preservando outras keys)

#### Scenario: rejeitar valor fora do enum

- **WHEN** agent envia `{"loyalty_tier":"platinum"}` mas valores permitidos são `["bronze","silver","gold"]`
- **THEN** retorna 400 com erro `value_not_in_list`

#### Scenario: rejeitar key sem definition

- **WHEN** agent envia `{"random_key":"xyz"}` sem definition correspondente
- **THEN** retorna 400 com erro `unknown_attribute_key` e lista as keys conhecidas

#### Scenario: remover chave específica

- **WHEN** agent envia `DELETE /contacts/7/custom_attributes` com `{"keys":["loyalty_tier"]}`
- **THEN** `additional_attributes` é atualizado removendo essa key mas preservando outras

### Requirement: Schema de custom_attribute_definitions

A migration `0004_helpdesk_core.sql` MUST criar:

```sql
CREATE TABLE custom_attribute_definitions (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  attribute_key TEXT NOT NULL,
  attribute_display_name TEXT NOT NULL,
  attribute_display_type TEXT NOT NULL CHECK (attribute_display_type IN ('text','number','currency','percent','link','date','list','checkbox')),
  attribute_model TEXT NOT NULL CHECK (attribute_model IN ('contact','conversation')),
  attribute_values JSONB,
  attribute_description TEXT,
  regex_pattern TEXT,
  default_value TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (account_id, attribute_model, attribute_key)
);
CREATE INDEX idx_custom_attr_defs_account ON custom_attribute_definitions(account_id);
```

Valores ficam em `contacts.additional_attributes` e `conversations.additional_attributes` (colunas já existem). Migration MAY adicionar index GIN em `additional_attributes` pra performance de saved filters:

```sql
CREATE INDEX idx_contacts_additional_attrs_gin ON contacts USING GIN (additional_attributes);
CREATE INDEX idx_conversations_additional_attrs_gin ON conversations USING GIN (additional_attributes);
```

#### Scenario: migration cria tabela e índices

- **WHEN** migration `0004` roda
- **THEN** tabela `custom_attribute_definitions` existe; CHECK constraints em `attribute_display_type` e `attribute_model` estão ativas; índices GIN em `additional_attributes` existem
