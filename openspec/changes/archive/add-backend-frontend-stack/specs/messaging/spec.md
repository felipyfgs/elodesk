## ADDED Requirements

### Requirement: Modelagem Contact, ContactInbox, Conversation, Message, Attachment

O schema Prisma SHALL definir:
- `Contact` com `id`, `accountId`, `name?`, `phoneNumber?`, `waJid?`, `email?`, `avatarUrl?`, `customAttributes Json`, `additionalAttributes Json`, `createdAt`, `updatedAt`. Índice `(accountId, waJid)`.
- `ContactInbox` com `contactId`, `inboxId`, `sourceId` (ex: waJid completo). Unique `(inboxId, sourceId)`.
- `Conversation` com `id`, `accountId`, `inboxId`, `contactInboxId`, `assigneeId? → User`, `status` enum `OPEN|PENDING|RESOLVED|SNOOZED`, `priority` enum `LOW|MEDIUM|HIGH|URGENT?`, `lastActivityAt`, `firstReplyAt?`, `unreadCount`, `additionalAttributes Json`, `customAttributes Json`. Índice `(accountId, status, lastActivityAt)`.
- `Message` com `id`, `conversationId`, `accountId`, `inboxId`, `content?`, `contentType` (`text|input_select|cards|form|article`), `messageType` (`INCOMING|OUTGOING|ACTIVITY|TEMPLATE`), `senderType` (`CONTACT|USER|SYSTEM`), `senderId?`, `sourceId?`, `private Boolean`, `status` (`PENDING|SENT|DELIVERED|READ|FAILED`), `contentAttributes Json`, `createdAt`. Unique parcial `(inboxId, sourceId) WHERE sourceId IS NOT NULL`. Índice `(conversationId, createdAt)`.
- `Attachment` com `id`, `messageId`, `fileType`, `fileKey` (MinIO), `mimeType`, `fileSize`, `externalUrl?`.

#### Scenario: schema criado via migration

- **WHEN** `pnpm prisma migrate dev`
- **THEN** todas as tabelas acima são criadas com os índices especificados

### Requirement: Pipeline inbound persiste e emite

Ao receber evento `Message` do wzap, o `WzapEventService` SHALL:
1. Fazer `upsert` de `Contact` pelo `(accountId, waJid)`.
2. Fazer `upsert` de `ContactInbox` por `(inboxId, sourceId=waJid)`.
3. Fazer `upsert` de `Conversation` por `(accountId, contactInboxId)` com status `OPEN`, criando-a se não existir.
4. Fazer `upsert` de `Message` por `(inboxId, sourceId="WAID:"+msgId)` com `messageType=INCOMING`, `senderType=CONTACT`, `senderId=contact.id`, `status=SENT`.
5. Se o evento contém mídia, enfileirar job `media-download` para baixar do wzap e re-uploadar no MinIO próprio, criando `Attachment`.
6. Incrementar `Conversation.unreadCount`.
7. Atualizar `Conversation.lastActivityAt=now()`.
8. Emitir via Socket.IO `message.new` para room `account:{id}` e `conversation:{id}` com o DTO completo.

#### Scenario: primeira mensagem de contato novo

- **WHEN** chega `Message` de um JID desconhecido
- **THEN** Contact, ContactInbox, Conversation e Message são criados
- **AND** frontend recebe `conversation.new` + `message.new`

#### Scenario: mensagem em conversa existente

- **WHEN** chega `Message` de um JID que já tem Contact
- **THEN** apenas Message é criada; Contact e Conversation são atualizados
- **AND** frontend recebe só `message.new` + `conversation.updated`

### Requirement: Pipeline outbound otimista

`POST /api/v1/conversations/:id/messages` com `{body}` SHALL:
1. Validar permissão (user é AGENT+ da account dona da Conversation).
2. Inserir `Message` com `messageType=OUTGOING`, `senderType=USER`, `senderId=user.id`, `sourceId=null`, `status=PENDING`.
3. Emitir `message.new` (estado otimista) imediatamente.
4. Chamar `wzap.sendText(sessionId, {phone:contact.waJid, body, replyTo?})`.
5. Ao receber `{messageId}`, atualizar Message com `sourceId="WAID:"+messageId` e `status=SENT`.
6. Emitir `message.updated`.
7. Se `wzap.sendText` falhar, atualizar `status=FAILED` e emitir `message.updated`.

#### Scenario: envio bem-sucedido

- **WHEN** `wzap.sendText` retorna 200 com messageId
- **THEN** a mensagem termina com status `SENT` e tem `sourceId`

#### Scenario: envio falha

- **WHEN** `wzap.sendText` retorna 5xx
- **THEN** a mensagem termina com status `FAILED` e `sourceId` null

#### Scenario: update de Receipt

- **WHEN** webhook `Receipt` chega referenciando a mensagem
- **THEN** `Message.status` evolui para `DELIVERED` ou `READ` e emit `message.updated`

### Requirement: Edição e deleção bidirecional

Inbound:
- Evento `MessageEdit` → localizar Message por `sourceId` → atualizar `content` e `contentAttributes.edited=true` → emit `message.updated`.
- Evento `MessageRevoke` → localizar → atualizar `contentAttributes.deleted=true` (soft delete) → emit `message.updated`.

Outbound:
- `PATCH /api/v1/messages/:id {body}` → `wzap.editMessage()` → atualizar local.
- `DELETE /api/v1/messages/:id` → `wzap.deleteMessage()` → atualizar `contentAttributes.deleted=true`.

#### Scenario: editar mensagem recém-enviada

- **WHEN** agent edita mensagem OUTGOING
- **THEN** wzap é chamado e, ao retornar 200, local é atualizado

#### Scenario: contato deleta mensagem enviada

- **WHEN** webhook `MessageRevoke` chega
- **THEN** Message.contentAttributes.deleted=true e frontend recebe `message.updated`

### Requirement: Mídia via MinIO próprio

Para mídia inbound: o job `media-download` SHALL chamar `wzap.getMedia(sessionId, messageId)`, baixar via presigned URL retornada, re-uploadar no bucket do backend com key `{accountId}/{inboxId}/{messageId}.{ext}`, e criar `Attachment` com `fileKey` apontando para o bucket próprio. URL pública é sempre presigned de 15 minutos.

Para mídia outbound: `POST /api/v1/uploads/signed-url` SHALL retornar presigned PUT para o frontend; o frontend uploada direto no MinIO do backend; depois `POST /api/v1/conversations/:id/messages/media` aciona o envio usando a URL pública do bucket.

#### Scenario: mídia inbound salva no bucket próprio

- **WHEN** recebe Message com anexo
- **THEN** arquivo é baixado do wzap e re-uploadado em `{accountId}/{inboxId}/{messageId}.{ext}` no bucket do backend
- **AND** Attachment.fileKey aponta para o bucket próprio (não para o do wzap)

#### Scenario: presigned URL expira

- **WHEN** frontend recebe presigned URL para download
- **THEN** a URL expira em 15 minutos; frontend deve renovar via endpoint se precisar
