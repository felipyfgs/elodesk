## ADDED Requirements

### Requirement: Socket.IO gateway com auth no handshake

O backend SHALL expor `RealtimeGateway` em `ws://.../socket.io`. Auth MUST ser feita no handshake lendo `auth.token` (JWT) — conexões sem token ou com token inválido são recusadas no `connect`.

#### Scenario: handshake com JWT válido

- **WHEN** client conecta com `auth: { token: "..." }` válido
- **THEN** a conexão é aceita e `socket.data.userId` é populado

#### Scenario: handshake sem token

- **WHEN** client conecta sem `auth.token`
- **THEN** o gateway chama `socket.disconnect()` imediatamente

#### Scenario: handshake com token expirado

- **WHEN** JWT expirou
- **THEN** a conexão é recusada com mensagem `"token expired"`

### Requirement: Rooms hierárquicos com validação de membership

Socket SHALL poder entrar em 3 tipos de room:
- `account:{accountId}` — broadcast geral da org.
- `inbox:{inboxId}` — eventos de uma sessão WA específica.
- `conversation:{conversationId}` — thread aberta.

Para cada `join`, o gateway MUST validar que o user pertence à account proprietária do recurso; caso contrário, emite erro `"access denied"` e não adiciona o socket ao room.

#### Scenario: user entra em room de sua account

- **WHEN** user envia `join.account` com `accountId` de uma account em que tem membership
- **THEN** socket é adicionado ao room `account:{id}` e gateway emite `joined` de volta

#### Scenario: user tenta entrar em room de outra account

- **WHEN** user envia `join.account` com `accountId` de outra account
- **THEN** gateway emite `error` com mensagem `"access denied"` e socket NÃO é adicionado ao room

### Requirement: Eventos emitidos pelo backend

O `RealtimeService` SHALL emitir os seguintes eventos para o room apropriado:

| Evento | Room | Quando |
|---|---|---|
| `message.new` | `conversation:{id}` + `account:{id}` | Nova mensagem (inbound ou outbound otimista) |
| `message.updated` | `conversation:{id}` | Status/edit/delete de mensagem existente |
| `conversation.new` | `account:{id}` | Primeira mensagem de um contato novo |
| `conversation.updated` | `conversation:{id}` + `account:{id}` | Mudança de status, atribuição, unread, labels |
| `session.status` | `inbox:{id}` + `account:{id}` | Mudança em `ChannelWhatsapp.status` |
| `qr.update` | `inbox:{id}` | Novo QR recebido |
| `presence` | `conversation:{id}` | Digitação/online (efêmero) |

Os payloads MUST ser DTOs JSON (não entidades Prisma cruas) para evitar vazamento de campos internos.

#### Scenario: message.new chega só para quem está em room correta

- **WHEN** nova mensagem é criada na conversation X da account A
- **THEN** sockets em `conversation:X` ou em `account:A` recebem o evento
- **AND** sockets de account B não recebem

#### Scenario: payload não vaza campos internos

- **WHEN** um evento `message.new` é emitido
- **THEN** payload não inclui `wzapToken`, `passwordHash` ou campos internos

### Requirement: Reconexão preserva subscrições

O backend SHALL documentar (via OpenAPI do Socket.IO ou README) que o client precisa re-emitir os `join.*` após reconexão. O backend MUST rejeitar gracefully se o socket tentar agir em room em que não entrou.

#### Scenario: reconexão sem re-join

- **WHEN** client reconecta e recebe eventos de um room em que estava antes
- **THEN** eventos não chegam (socket.io não re-entra automaticamente)
- **AND** client precisa chamar `join.*` novamente
