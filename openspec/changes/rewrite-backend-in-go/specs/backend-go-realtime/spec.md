## ADDED Requirements

### Requirement: WebSocket gateway com auth no handshake

O backend SHALL expor `GET /realtime` (upgrade WebSocket) autenticando no handshake:

- Preferencial: header `Sec-WebSocket-Protocol: bearer, <jwt>` — server devolve o mesmo protocol aceito
- Fallback (apenas dev): query `?token=<jwt>`

Token inválido ou ausente → conexão fechada antes de aceitar.

#### Scenario: handshake válido

- **WHEN** cliente abre WS com JWT válido
- **THEN** servidor aceita upgrade e associa `Client{userID}` ao hub

#### Scenario: handshake sem token

- **WHEN** cliente abre WS sem token
- **THEN** servidor retorna 401 no handshake e não faz upgrade

### Requirement: Rooms hierárquicos com validação de membership

Cliente WS SHALL poder entrar em rooms enviando mensagens JSON:

- `{"type":"join.account","payload":{"account_id":"..."}}`
- `{"type":"join.inbox","payload":{"inbox_id":"..."}}`
- `{"type":"join.conversation","payload":{"conversation_id":"..."}}`

Para cada join, o hub MUST validar que o user pertence à account dona do recurso (via `AccountUser`). Falha → responde `{"type":"error","payload":{"message":"access denied"}}` e NÃO adiciona ao room. Sucesso → responde `{"type":"joined","payload":{"scope":"account","id":"..."}}`.

Rooms mapeadas: `account:<id>`, `inbox:<id>`, `conversation:<id>`.

#### Scenario: join em account alheia

- **WHEN** cliente envia `join.account` com id de account em que não pertence
- **THEN** recebe `{"type":"error"}` e socket não é associado ao room

#### Scenario: join válido

- **WHEN** cliente envia `join.account` com id válido
- **THEN** recebe `{"type":"joined"}` e passa a receber broadcasts daquela account

### Requirement: Eventos emitidos pelo servidor

O `RealtimeService` SHALL emitir eventos (JSON `{type, payload}`) pros rooms apropriados:

| Evento | Room | Quando |
|---|---|---|
| `message.new` | `conversation:{id}` + `account:{id}` | Message inserida |
| `message.updated` | `conversation:{id}` | Status/edit/delete de Message |
| `conversation.new` | `account:{id}` | Primeira Message de contato novo |
| `conversation.updated` | `conversation:{id}` + `account:{id}` | Status/assignee mudaram |
| `inbox.status` | `inbox:{id}` + `account:{id}` | Status da inbox mudou |

Payloads MUST ser DTOs (nunca entidade DB crua). Nunca conter `password_hash`, `api_token`, `hmac_token`.

#### Scenario: isolamento cross-tenant em broadcast

- **WHEN** nova mensagem na conversation X da account A
- **THEN** apenas sockets em `conversation:X` ou `account:A` recebem
- **AND** sockets conectados mas sem join relevante não recebem

#### Scenario: payload não vaza segredo

- **WHEN** `message.new` é emitido
- **THEN** o JSON não contém `password_hash`, `api_token` nem `hmac_token`

### Requirement: Reconexão requer re-join

O servidor SHALL documentar (readme/comment) que o cliente é responsável por re-emitir `join.*` após reconexão. Servidor NÃO lembra rooms de conexões anteriores.

#### Scenario: reconexão sem re-join

- **WHEN** cliente desconecta e reconecta sem enviar joins
- **THEN** não recebe eventos até enviar novos `join.*`
