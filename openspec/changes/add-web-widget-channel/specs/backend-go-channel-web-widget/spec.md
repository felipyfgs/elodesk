## ADDED Requirements

### Requirement: Tipo `Channel::WebWidget`

O backend SHALL expor o tipo `Channel::WebWidget` com tabela `channels_web_widget` contendo `website_token` (32 bytes random base64url, unique, público), `hmac_token_ciphertext` (AES-GCM via KEK, secreto), `website_url`, `widget_color` (hex), `welcome_title`, `welcome_tagline`, `reply_time` (`in_a_few_minutes|in_a_few_hours|in_a_day`), `feature_flags` JSONB, `created_at`, `updated_at`. Provisão via `POST /api/v1/accounts/:aid/inboxes/web_widget`.

#### Scenario: Criar canal web widget

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/web_widget` com `{name, websiteUrl, widgetColor, welcomeTitle, welcomeTagline, replyTime, featureFlags}`
- **THEN** o backend gera `website_token` (32 bytes random base64url), gera `hmac_token` (32 bytes random base64url), encripta `hmac_token` → `hmac_token_ciphertext` via KEK, cria `channels_web_widget` e `inboxes(channel_type='Channel::WebWidget')`; resposta contém `websiteToken`, `embedScript` (snippet JS pronto pra copiar/colar), `hmacToken` (mostrado UMA vez)

#### Scenario: Segredos nunca retornam após criação

- **WHEN** `GET /api/v1/accounts/:aid/inboxes/:id` para canal web widget
- **THEN** o response contém `websiteToken` (seguro expor), mas NUNCA `hmacToken` (só na criação); campos de aparência são retornados

#### Scenario: Rotação do HMAC token

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/:id/rotate_hmac`
- **THEN** o backend gera novo `hmac_token`, atualiza ciphertext, retorna o novo UMA vez; integrações antigas precisam ser atualizadas no cliente host

### Requirement: Embed script estático

O backend SHALL expor `GET /widget/:websiteToken` retornando HTML/JS mínimo que carrega o bundle JavaScript hospedado em `WIDGET_PUBLIC_BASE_URL`. O script é público (sem CORS restrict), cacheável por 1 hora com `ETag`.

#### Scenario: Embed script carrega bundle

- **WHEN** `GET /widget/:websiteToken` com um `websiteToken` válido
- **THEN** resposta `200` com `Content-Type: text/javascript`, body referenciando `{WIDGET_PUBLIC_BASE_URL}/widget.js` + config JSON inline (websiteToken, cores, textos), header `Cache-Control: public, max-age=3600`

#### Scenario: websiteToken inválido

- **WHEN** `GET /widget/:websiteToken` com token inexistente
- **THEN** resposta `404 Not Found` (sem body revelador); script falha silenciosamente no site host

### Requirement: Sessão de visitante via JWT de 30 dias

O backend SHALL emitir JWT de visitante em `POST /api/v1/widget/sessions` com claims `{sub: contact_identifier, website_token, iat, exp, typ:"visitor"}`, TTL default 30 dias (`WIDGET_SESSION_TTL_DAYS`), assinado com `WIDGET_JWT_SECRET` (separado do JWT de agente). Persistido em cookie `elodesk_widget_session_<website_token>` (`HttpOnly`, `SameSite=None`, `Secure` em produção). Também retornado no body para o JS do widget guardar.

#### Scenario: Primeira sessão cria contact anônimo

- **WHEN** `POST /api/v1/widget/sessions` com `{websiteToken}` sem cookie nem body de identity
- **THEN** o backend cria `contact(identifier="anon_<ulid>", meta={browser, os, city_from_ip}, accountId derivado do websiteToken)`, cria `conversation` nova open, retorna `{contactIdentifier, conversationId, pubsubToken, jwt}` + set-cookie

#### Scenario: Sessão retomada via cookie

- **WHEN** `POST /api/v1/widget/sessions` com cookie contendo JWT válido
- **THEN** o backend decodifica JWT, resolve contact e conversation ativa (última open ou cria nova), retorna tokens

#### Scenario: JWT expirado

- **WHEN** o JWT no cookie tem `exp < now`
- **THEN** o backend trata como "primeira sessão" — novo contact anônimo; cookie antigo é sobrescrito

### Requirement: Identify com HMAC verification

O backend SHALL processar `POST /api/v1/widget/identify` com `{identifier, email?, name?, identifierHash}` validando `hmac_sha256(hmac_token, identifier) == identifierHash` (em hex). Match → upgrade do contact anônimo para identificado (merge se já existe contact com esse identifier). Mismatch → `401 Unauthorized`.

#### Scenario: Identify válido cria/mergeia contact

- **WHEN** `POST /widget/identify` com `identifier="user@acme.com"`, `identifierHash=<hex correto>`, JWT de visitor válido
- **THEN** se existe contact com esse identifier no mesmo account: merge (conversas do anon passam pra ele); senão: update do contact anônimo com os novos campos; resposta retorna novo JWT com `sub=user@acme.com`

#### Scenario: HMAC inválido

- **WHEN** `identifierHash` não bate o HMAC esperado
- **THEN** `401 Unauthorized` código `invalid_identifier_hash`; nada muda; log `warn`

#### Scenario: Identify sem HMAC é aceito com flag

- **WHEN** `identifierHash` ausente no request
- **THEN** contact é atualizado mas com `meta.verified_identity=false`; frontend pode renderizar diferente (bandeira "não verificado")

### Requirement: Mensagem inbound do visitante

O backend SHALL processar `POST /api/v1/widget/messages` com JWT de visitor + `{content, attachmentIds?}`. Persiste `message(conversation_id from JWT, sender_type=Contact, sender_id=contact_id, content, message_type=incoming)`. Atualiza `conversation.last_activity_at`, emite evento realtime `message.created` na fila do agente.

#### Scenario: Envio de texto

- **WHEN** `POST /widget/messages` com `{content:"olá"}`, JWT visitor válido
- **THEN** `message` criada; agente vê na inbox em realtime; resposta `201` com message ID

#### Scenario: Rate limit por IP excedido

- **WHEN** um IP envia mais de 60 mensagens/min no mesmo `websiteToken`
- **THEN** resposta `429 Too Many Requests` com header `Retry-After: <seconds>`; log `warn`

#### Scenario: JWT inválido

- **WHEN** o JWT ausente ou assinatura quebrada
- **THEN** resposta `401 Unauthorized` código `invalid_visitor_token`

### Requirement: Attachments via MinIO presigned

O backend SHALL expor `POST /api/v1/widget/attachments` retornando URL presigned de upload pro MinIO (TTL 10min, path `{accountId}/{inboxId}/{conversationId}/{ulid}/{filename}`). Cliente faz upload direto. Tamanho máximo por arquivo 10MB (override via `feature_flags.attachment_max_mb`). Em `POST /widget/messages`, cliente referencia `attachmentIds` de uploads prévios.

#### Scenario: Upload de imagem

- **WHEN** `POST /widget/attachments` com `{fileName, contentType:"image/png", size:500000}`, JWT visitor
- **THEN** resposta `{uploadUrl, attachmentId}`; cliente faz `PUT` no uploadUrl; backend cria `attachment(file_type=FileTypeImage, file_key=<path>)`

#### Scenario: Arquivo excede limite

- **WHEN** o request traz `size > 10_000_000`
- **THEN** resposta `413 Payload Too Large` código `attachment_too_large`

### Requirement: SSE realtime para outbound do agente

O backend SHALL expor `GET /widget/:websiteToken/ws` (SSE `text/event-stream`) autenticado por `pubsub_token` (query param). Headers: `Cache-Control: no-cache`, `X-Accel-Buffering: no`, `Content-Type: text/event-stream`. Keepalive a cada 30s (`: heartbeat\n\n`). Handler subscreve em `widget:pubsub:<pubsubToken>` no Redis; mensagens recebidas são escritas no stream como eventos `event: message\ndata: <json>\n\n`.

#### Scenario: Conexão SSE válida

- **WHEN** `GET /widget/:websiteToken/ws?pubsubToken=<token>` com token correto
- **THEN** resposta `200` mantida aberta; backend subscreve no Redis; entrega keepalives a cada 30s; entrega mensagens outbound em tempo real

#### Scenario: Agente responde visitante

- **WHEN** agente envia mensagem na conversa pelo frontend
- **THEN** o backend persiste `message(message_type=outgoing)`, publica em `widget:pubsub:<pubsub_token>` com `{type:"message.created", ...}`, o stream SSE entrega ao widget

#### Scenario: Desconexão do cliente

- **WHEN** o widget fecha a conexão (aba fechada, navegação)
- **THEN** o handler detecta via `r.Context().Done()`, faz `UNSUBSCRIBE` do Redis, libera recursos; log `info`

#### Scenario: pubsub_token inválido

- **WHEN** o `pubsubToken` não bate com o do `conversation`
- **THEN** resposta `401 Unauthorized`; conexão fechada imediatamente

### Requirement: Polling fallback para navegadores sem SSE

O backend SHALL expor `GET /api/v1/widget/messages?after=<lastMessageId>&limit=20` autenticado com JWT visitor, retornando mensagens da conversa com `id > after` ordenadas asc. Limite máximo de 100 msg por request.

#### Scenario: Polling retorna mensagens novas

- **WHEN** cliente faz `GET /widget/messages?after=42&limit=20`
- **THEN** backend retorna `[msg43, msg44, ...]` até 20 mensagens; cada mensagem inclui `id`, `content`, `messageType`, `createdAt`, `senderType`, attachments

#### Scenario: Sem novas mensagens

- **WHEN** não há mensagens com `id > after`
- **THEN** resposta `200 OK` com array vazio `[]`

### Requirement: Rate limits por IP e website_token

Todas as rotas públicas `/api/v1/widget/*` e `/widget/*` SHALL aplicar rate limits via Redis:

- `POST /widget/sessions`: 10 req/min/IP, 1000 req/hora/websiteToken.
- `POST /widget/messages`: 60 req/min/IP.
- `POST /widget/identify`: 20 req/min/IP.
- `POST /widget/attachments`: 30 req/min/IP.

Ultrapassar limite → `429 Too Many Requests` com header `Retry-After`.

#### Scenario: Flood de sessions pelo mesmo IP

- **WHEN** um IP envia 11 `POST /widget/sessions` em 1 minuto
- **THEN** a 11ª retorna `429`; `Retry-After: <segundos restantes da janela>`

#### Scenario: Limites independentes por tipo

- **WHEN** um IP envia 15 `POST /widget/sessions` + 50 `POST /widget/messages` em 1 minuto
- **THEN** o limite de sessions é excedido (após 10); o limite de messages ainda não (50 < 60); cada tipo tem seu próprio bucket

### Requirement: CORS aberto para rotas públicas

Todas as rotas públicas do widget SHALL responder com `Access-Control-Allow-Origin: *` e `Access-Control-Allow-Credentials: false`. Cookie de sessão usa `SameSite=None; Secure` em produção (e `SameSite=Lax` em dev quando `APP_ENV=dev`).

#### Scenario: Preflight de browser

- **WHEN** browser envia `OPTIONS /api/v1/widget/sessions` de origem `https://cliente.com.br`
- **THEN** resposta com `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: POST,GET,OPTIONS`, `Access-Control-Allow-Headers: Content-Type,Authorization`

#### Scenario: Cookie cross-site

- **WHEN** widget embutido em `https://cliente.com.br` chama `POST /api/v1/widget/messages` (domínio do elodesk diferente)
- **THEN** cookie `elodesk_widget_session_<token>` é enviado (`SameSite=None; Secure` permite); JWT no header Authorization também é aceito como fallback
