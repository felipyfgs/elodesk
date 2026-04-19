## ADDED Requirements

### Requirement: Provisionamento de Channel::Api

O backend SHALL expor `POST /api/v1/accounts/:aid/inboxes` (autenticado JWT + role OWNER/ADMIN) aceitando `{name, channel_type?, channel: {webhook_url?, hmac_mandatory?}}` que cria:

- `Inbox` (accountId, name, channelType default "api")
- `ChannelApi` com `identifier` (token random de 32 bytes base64url), `api_token` (random 48 bytes, encriptado no DB com KEK), `hmac_token` (random 32 bytes, encriptado), `webhook_url` (opcional), `hmac_mandatory` (default false)

A resposta 201 MUST retornar `api_token` e `hmac_token` em claro UMA ÚNICA VEZ (cliente precisa copiar). Requests subsequentes não retornam segredos em claro.

#### Scenario: criação retorna credenciais

- **WHEN** OWNER faz `POST /api/v1/accounts/:aid/inboxes {name:"Suporte"}`
- **THEN** retorna 201 com `{inbox, channel:{identifier, api_token, hmac_token, webhook_url, hmac_mandatory}}`
- **AND** `GET /api/v1/accounts/:aid/inboxes/:id` posteriormente NÃO retorna `api_token` nem `hmac_token` em claro

### Requirement: Auth por api_access_token

O backend SHALL fornecer middleware `ApiTokenAuth` que lê header `api_access_token: <token>` e:

1. Descriptografa e compara (constant-time) contra cada `ChannelApi.api_token` da account (via lookup por hash ou por tentativa).
2. Popula `c.Locals("inbox", inbox)` e `c.Locals("account", account)` e permite o request.
3. Ausente ou inválido → 401 sem detalhar.

Rotas `/api/v1/accounts/:aid/contacts`, `.../conversations`, `.../conversations/:cid/messages`, `.../actions/contact_merge` aceitam tanto JWT (agente) quanto `api_access_token` (provider); `OrgScope` roda depois e usa o account já populado.

#### Scenario: provider com token válido

- **WHEN** wzap faz `POST /api/v1/accounts/:aid/contacts` com `api_access_token: <válido>`
- **THEN** request passa e `c.Locals("inbox")` é populado

#### Scenario: token inválido

- **WHEN** header `api_access_token` não bate com nenhuma inbox
- **THEN** retorna 401 sem distinguir causa

### Requirement: Contacts endpoints (Chatwoot-compatível)

O backend SHALL expor:

- `POST /api/v1/accounts/:aid/contacts/search?q=<query>` — busca por nome ou phone_number
- `POST /api/v1/accounts/:aid/contacts/filter` com body `{payload:[{attribute_key, filter_operator, values}]}` — filtro multi-attribute
- `POST /api/v1/accounts/:aid/contacts` aceitando `{inbox_id, name, identifier, phone_number, avatar_url, email, additional_attributes, custom_attributes}` — cria Contact + ContactInbox (source_id = `identifier`)
- `PATCH /api/v1/accounts/:aid/contacts/:id` — atualiza campos parciais
- `GET /api/v1/accounts/:aid/contacts/:id/conversations` — lista conversas do contato (reverso)
- `POST /api/v1/accounts/:aid/actions/contact_merge` aceitando `{base_contact_id, mergee_contact_id}` — merge idempotente

Shape de request/response = idêntico ao Chatwoot (ver `wzap/internal/integrations/chatwoot/client.go`).

#### Scenario: upsert por identifier

- **WHEN** `POST /contacts` com `inbox_id=1, identifier="5511988776655"` e já existe ContactInbox com esse `source_id` na mesma inbox
- **THEN** retorna o contact existente (idempotente) em vez de criar duplicata

#### Scenario: filter por phone

- **WHEN** `POST /contacts/filter` com `{payload:[{attribute_key:"phone_number", filter_operator:"equal_to", values:["+5511988776655"]}]}`
- **THEN** retorna array `{payload:[...]}` com contatos que batem

### Requirement: Conversations endpoints

O backend SHALL expor:

- `POST /api/v1/accounts/:aid/conversations` com `{inbox_id, contact_id, source_id, status?}` — cria Conversation (e ContactInbox se não existir)
- `POST /api/v1/accounts/:aid/conversations/:cid/toggle_status` com `{status}` — muda status (`open|resolved|pending|snoozed`)
- `GET /api/v1/accounts/:aid/conversations` — lista paginada por `account_id`, filtros `status`, `assignee_id`

#### Scenario: toggle fecha conversa

- **WHEN** `POST /conversations/:cid/toggle_status {"status":"resolved"}`
- **THEN** Conversation.status = RESOLVED e evento `conversation_status_changed` é enfileirado pro webhook outbound

### Requirement: Messages endpoints (JSON + multipart)

O backend SHALL expor `POST /api/v1/accounts/:aid/conversations/:cid/messages` aceitando dois formatos:

**JSON** (`Content-Type: application/json`):
```json
{
  "content": "...",
  "message_type": "incoming" | "outgoing" | "template",
  "source_id": "WAID:abc",
  "content_attributes": {...},
  "echo_id": "optional"
}
```

**Multipart** (`multipart/form-data`):
- fields `content`, `message_type`, `source_id`, `content_attributes` (JSON string)
- files `attachments[]` (múltiplos); cada arquivo é streamado pro MinIO em `{accountId}/{inboxId}/{messageId}.{ext}`; `Attachment` row criada com `file_key`

Idempotência: `(inbox_id, source_id) WHERE source_id IS NOT NULL` é UNIQUE parcial; retry com mesmo `source_id` retorna a mensagem existente.

`DELETE /api/v1/accounts/:aid/conversations/:cid/messages/:mid` — soft delete marcando `content_attributes.deleted=true`.

#### Scenario: inbound message idempotente

- **WHEN** `POST messages` com `source_id="WAID:abc"` é feito duas vezes
- **THEN** apenas uma linha existe em `messages` com esse source_id na inbox

#### Scenario: multipart upload

- **WHEN** `POST messages` multipart com um arquivo `image/jpeg`
- **THEN** `messages` row criada + `attachments` row apontando pro MinIO + evento `message_created` emitido

### Requirement: Read receipts via public API

O backend SHALL expor `POST /public/api/v1/inboxes/:identifier/contact_inboxes/conversations/:cid/update_last_seen` aceitando `{source_id, last_seen}` e atualizando `Conversation.last_seen_at`. Auth por `identifier` + `identifier_hash` (HMAC opcional se `hmac_mandatory=true`).

#### Scenario: update last seen

- **WHEN** provider faz POST com `identifier` e hash válidos
- **THEN** `last_seen_at` é atualizado e retorna 200
