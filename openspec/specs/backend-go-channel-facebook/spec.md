# backend-go-channel-facebook Specification

## Purpose

Facebook Messenger channel (`Channel::FacebookPage`) built on the Meta Graph
API. Supports page access tokens, webhook subscriptions including
`standby`/`messaging_handovers`, delivery watermark updates, and Instagram
Business Account linked-inbox dedup.

## Requirements

### Requirement: Tipo `Channel::FacebookPage`

O backend SHALL expor o tipo `Channel::FacebookPage` com tabela `channels_facebook_page` armazenando `page_id` (unique por account), `page_access_token_ciphertext`, `user_access_token_ciphertext` (opcional), `instagram_id` (opcional, linkando Instagram Business Account à Page).

#### Scenario: Criar canal Facebook Page

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/facebook_page` com `{pageId, pageAccessToken, userAccessToken?, instagramId?}`
- **THEN** o sistema grava tokens via KEK e cria `inboxes(channel_type='Channel::FacebookPage')`

### Requirement: Webhook Facebook com handshake + signature

Idêntico ao Instagram: `GET /webhooks/facebook/:identifier` para handshake com `FB_VERIFY_TOKEN`, `POST /webhooks/facebook/:identifier` com `meta.VerifySignature`. Subscribed fields no painel Meta DEVEM incluir: `messages`, `messaging_postbacks`, `message_deliveries`, `message_reads`, `message_echoes`, `standby`, `messaging_handovers`.

#### Scenario: Handshake

- **WHEN** Meta faz GET com `verify_token = FB_VERIFY_TOKEN`
- **THEN** retorna `200` + `hub.challenge`

#### Scenario: Delivery update

- **WHEN** POST traz `messaging[0].delivery.watermark = <ts>`
- **THEN** o sistema atualiza `messages.status = delivered` para todas as mensagens da conversa com timestamp ≤ watermark

### Requirement: Standby handover

Quando o payload contém `entry.standby[]` (ao invés de `messaging[]`), o backend SHALL processar as mensagens com mesma lógica de `messaging`, mas marcando `content_attributes.source = "standby"` para rastreabilidade.

#### Scenario: Mensagem em standby

- **WHEN** outro app tem controle da conversa e um webhook vem com `standby[{sender, recipient, message}]`
- **THEN** a mensagem é gravada com `content_attributes.source='standby'` e aparece na conversa

### Requirement: Dedup para Instagram linked

Quando `channels_facebook_page.instagram_id IS NOT NULL` e um webhook Instagram chega com esse `instagram_id`, o backend MUST usar o `DedupLock` global por `message.mid` para evitar que o mesmo DM crie duas mensagens (uma via `/webhooks/instagram`, outra via `/webhooks/facebook`).

#### Scenario: DM Instagram duplicado cross-canal

- **WHEN** o mesmo `message.mid` chega em `/webhooks/instagram/:id1` E em `/webhooks/facebook/:id2` (onde id2 tem `instagram_id` linkado ao id1)
- **THEN** `DedupLock.Acquire("elodesk:meta:<mid>")` passa na primeira, falha na segunda; mensagem é criada apenas uma vez

### Requirement: Outbound via page_access_token

`Channel::FacebookPage.SendOutbound` SHALL POSTar em `graph.facebook.com/{page_id}/messages` com `Authorization: Bearer <page_access_token>`. Suporta `message.text`, `message.attachment` (image/video/audio/file + payload.url), `quick_replies[]`, `messaging_tag` (para mensagens fora da 24h window).

#### Scenario: Envio texto com quick replies

- **WHEN** a mensagem tem `quick_replies: [{content_type:'text', title:'Sim', payload:'YES'}]`
- **THEN** o POST inclui `message.quick_replies` no body; provider responde `messages[0].id` como `source_id`

#### Scenario: Envio fora da janela 24h sem tag

- **WHEN** o agente envia mensagem ao contato mais de 24h após o último DM e `messaging_tag` não é fornecido
- **THEN** Graph API rejeita; o sistema propaga erro + `message.status=failed` e loga o erro Graph

### Requirement: Segredos nunca retornam após criação

GET de canal Facebook MUST omitir tokens. Response público contém `pageId`, `instagramId?`, `requiresReauth`, `createdAt`, `updatedAt`.
