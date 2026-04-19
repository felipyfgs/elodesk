## ADDED Requirements

### Requirement: CRUD de notes por contato

O backend SHALL expor endpoints REST pra gerenciar notas internas por contact:

- `GET /api/v1/accounts/{aid}/contacts/{cid}/notes` — listar notas do contato, ordenadas por `created_at DESC`.
- `POST /api/v1/accounts/{aid}/contacts/{cid}/notes` body `{content: string}`.
- `PATCH /api/v1/accounts/{aid}/contacts/{cid}/notes/{nid}` body `{content}`.
- `DELETE /api/v1/accounts/{aid}/contacts/{cid}/notes/{nid}`.

Restrições:
- `content` MUST ter no máximo 50.000 caracteres.
- `user_id` do criador MUST ser gravado automaticamente a partir do JWT.
- Criação e leitura: role `Agent+`.
- Edição e delete: somente o autor (`user_id == current_user`) OU role `Admin+`.
- Notas NUNCA devem ser entregues ao cliente/canal externo (são internas).

#### Scenario: agent cria nota no contato

- **WHEN** agent logado (user_id=10) envia `POST /api/v1/accounts/1/contacts/7/notes` com `{"content":"Cliente pediu desconto"}`
- **THEN** retorna 201 com `{id, contact_id:7, user_id:10, content, created_at, updated_at}`

#### Scenario: outro agent não pode editar nota alheia

- **WHEN** agent B (user_id=11) envia `PATCH` em nota criada pelo agent A (user_id=10)
- **THEN** retorna 403 com erro `not_note_owner`

#### Scenario: admin pode editar qualquer nota

- **WHEN** admin (user_id=99, role=Admin) envia `PATCH` em nota criada por outro agent
- **THEN** retorna 200 e nota é atualizada

### Requirement: Notes visíveis em qualquer conversation do contato

Quando frontend exibe detalhe de contact OU conversation thread, o client SHALL conseguir buscar notes desse contact via `GET /contacts/{cid}/notes`. Não existe conceito de "nota por conversation" — notas são sempre no nível do contact.

Após criação de nota, o hub de realtime MUST emitir `{type: "note.created", payload: {note_id, contact_id, user_id, account_id}}` na room da account (clientes com esse contact aberto podem refetch).

#### Scenario: broadcast de nova nota

- **WHEN** agent cria nota em contact 7 da account 1
- **THEN** todos os clientes conectados na room `account.1` recebem `{type:"note.created", payload:{note_id, contact_id:7, user_id, account_id:1}}`

### Requirement: Schema de notes

A migration `0004_helpdesk_core.sql` MUST criar:

```sql
CREATE TABLE notes (
  id BIGSERIAL PRIMARY KEY,
  account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  contact_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notes_contact ON notes(contact_id, created_at DESC);
CREATE INDEX idx_notes_account ON notes(account_id);
```

`user_id` usa `ON DELETE RESTRICT` (não deletar user que tem notas; se user for removido, transferir notas pra placeholder antes).

#### Scenario: migration cria tabela

- **WHEN** migration `0004` roda
- **THEN** tabela `notes` existe com index composto em `(contact_id, created_at DESC)` pra listagem ordenada rápida
