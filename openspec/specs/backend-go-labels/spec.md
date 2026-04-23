## ADDED Requirements

### Requirement: CRUD de labels account-scoped

O backend SHALL expor endpoints REST pra gerenciar labels por account:

- `GET /api/v1/accounts/{aid}/labels` — listar todas as labels da account.
- `POST /api/v1/accounts/{aid}/labels` — criar label. Body: `{title: string, color: string, description?: string, show_on_sidebar: boolean}`.
- `PATCH /api/v1/accounts/{aid}/labels/{id}` — atualizar.
- `DELETE /api/v1/accounts/{aid}/labels/{id}` — remover label e todas as suas taggings.

Restrições:
- `title` MUST ser unique por account (case-insensitive, lowercase normalizado).
- `color` MUST ser hex válido (`^#[0-9A-Fa-f]{6}$`); default `#1f93ff` se omitido.
- CRUD MUST exigir role `Admin` ou `Owner` via `RolesRequired` middleware.

#### Scenario: criar label nova

- **WHEN** admin envia `POST /api/v1/accounts/1/labels` com `{"title":"Urgente","color":"#ff0000","show_on_sidebar":true}`
- **THEN** retorna 201 com `{id, title:"urgente", color:"#ff0000", show_on_sidebar:true, account_id:1, created_at, updated_at}`

#### Scenario: rejeitar título duplicado

- **WHEN** admin tenta criar label com `title` já existente na account (case-insensitive)
- **THEN** retorna 409 com erro `label_title_taken`

#### Scenario: agent não pode criar label

- **WHEN** user com role Agent envia POST /labels
- **THEN** retorna 403

### Requirement: Aplicar e remover label em conversation

O backend SHALL permitir associar uma label a uma conversation:

- `POST /api/v1/accounts/{aid}/conversations/{id}/labels` body `{label_id: number}`.
- `DELETE /api/v1/accounts/{aid}/conversations/{id}/labels/{label_id}`.
- `GET /api/v1/accounts/{aid}/conversations/{id}/labels` — listar labels aplicadas.

Tagging MUST ser única por `(label_id, conversation_id)` (idempotente). Aplicação requer role `Agent+`.

Após apply/remove, o hub de realtime MUST emitir evento pra room da conversation:

- `{type: "label.added", payload: {conversation_id, label_id, account_id}}`
- `{type: "label.removed", payload: {conversation_id, label_id, account_id}}`

#### Scenario: aplicar label a conversation

- **WHEN** agent envia `POST /api/v1/accounts/1/conversations/42/labels` com `{"label_id": 3}`
- **THEN** retorna 201, label fica associada, e clientes conectados na room `conversation.42` recebem `{type:"label.added", payload:{conversation_id:42, label_id:3}}`

#### Scenario: re-aplicar label já existente (idempotente)

- **WHEN** agent envia a mesma associação duas vezes
- **THEN** retorna 200 (ou 201 na primeira, 200/204 na segunda), sem criar duplicata; query no banco retorna 1 row

### Requirement: Aplicar e remover label em contact

O backend SHALL permitir associar label a contact com endpoints análogos:

- `POST /api/v1/accounts/{aid}/contacts/{id}/labels` body `{label_id}`.
- `DELETE /api/v1/accounts/{aid}/contacts/{id}/labels/{label_id}`.
- `GET /api/v1/accounts/{aid}/contacts/{id}/labels`.

Mesma regra de idempotência e broadcast análogo na room do contact (se houver; caso não, apenas na account room).

#### Scenario: listar labels de contact

- **WHEN** agent envia `GET /api/v1/accounts/1/contacts/7/labels`
- **THEN** retorna 200 com array de labels aplicadas ao contact

### Requirement: Delete de label em cascata

Quando uma label é deletada, todas as `label_taggings` que a referenciam MUST ser removidas via FK `ON DELETE CASCADE`. Clientes realtime MUST receber `label.removed` pra cada associação afetada? Não — emitir apenas um evento `label.deleted` na account room.

#### Scenario: deletar label remove todas as taggings

- **WHEN** admin envia `DELETE /api/v1/accounts/1/labels/3`
- **THEN** a label é removida, todas as `label_taggings` com `label_id=3` são removidas em cascata, e todos os clientes da account recebem `{type:"label.deleted", payload:{label_id:3}}`

### Requirement: Tabela de taggings polimórfica

O schema MUST conter tabela `label_taggings` com colunas `(id, account_id, label_id, taggable_type, taggable_id, created_at)`:

- `taggable_type` MUST ser CHECK-constrained em `('conversation','contact')`.
- Index composto em `(taggable_type, taggable_id)` pra lookup de labels por objeto.
- Unique constraint em `(label_id, taggable_type, taggable_id)`.
- FK `label_id` com `ON DELETE CASCADE` pra `labels(id)`.

#### Scenario: migration cria tabela com constraints

- **WHEN** migration `0004_helpdesk_core.sql` é aplicada
- **THEN** tabela `label_taggings` existe com CHECK em `taggable_type`, unique composto, e index em `(taggable_type, taggable_id)`
