## ADDED Requirements

### Requirement: Tipo `Channel::Telegram`

O backend SHALL expor o tipo `Channel::Telegram` com tabela `channels_telegram` armazenando `bot_token_ciphertext` (AES-GCM via KEK), `bot_name` (auto-fetched), `webhook_identifier` (token opaco para URL do webhook), `secret_token_ciphertext` (AES-GCM) usado para validação do header `X-Telegram-Bot-Api-Secret-Token`.

#### Scenario: Criar canal Telegram

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/telegram` com `{botToken}`
- **THEN** o backend chama `getMe` com o botToken (extrai `bot_name`), gera `secret_token` (32 bytes random base64url), gera `webhook_identifier` (16 bytes random base64url), faz `setWebhook` apontando pra `/webhooks/telegram/:identifier` com `secret_token`, grava ciphertexts via KEK, cria `inboxes(channel_type='Channel::Telegram')` e retorna a inbox (sem tokens)

#### Scenario: Falha de getMe rejeita criação

- **WHEN** `getMe` retorna erro (bot_token inválido)
- **THEN** resposta `400 Bad Request` com mensagem `invalid_bot_token`; NADA é persistido

### Requirement: Webhook Telegram validado por secret_token

`POST /webhooks/telegram/:identifier` SHALL verificar o header `X-Telegram-Bot-Api-Secret-Token` contra o `secret_token` armazenado (decriptado). Falha → `401 Unauthorized`. Sucesso → parse `TelegramUpdate` e roteamento.

#### Scenario: Secret válido

- **WHEN** o header bate o secret_token do canal `:identifier`
- **THEN** o webhook é processado (messaging, edited, callback) e retorna `200 OK`

#### Scenario: Secret ausente ou inválido

- **WHEN** o header está ausente ou diferente
- **THEN** resposta `401 Unauthorized`; nada é processado; evento logado em `warn`

### Requirement: Suporte a tipos de message

Para cada `Update.message` recebido com `chat.type == "private"`, o backend SHALL suportar: `text`, `photo`, `video`, `audio`, `voice`, `document`, `sticker`, `location`, `contact`, `video_note`, `animation`. Tipos não suportados são logados em `debug` e a mensagem é gravada com `content_type=ContentTypeText` e `content="[unsupported:<type>]"`.

#### Scenario: Mensagem texto

- **WHEN** o Update traz `message.text = "olá"`
- **THEN** uma `message` é criada com `content="olá"`, `source_id=<message_id>`, `content_type=ContentTypeText`

#### Scenario: Foto inbound

- **WHEN** o Update traz `message.photo = [{file_id, ...}, ...]` (array de sizes)
- **THEN** uma `attachment` row é criada com `file_type=FileTypeImage` e `content_attributes.file_id` = maior resolução disponível; download é lazy (não baixa no momento)

#### Scenario: Grupo é ignorado

- **WHEN** `message.chat.type in ("group","supergroup","channel")`
- **THEN** o webhook retorna `200 OK` SEM criar nada; evento logado em `info`

### Requirement: Resolução lazy de media

Quando frontend/agente solicita `GET /api/v1/accounts/:aid/attachments/:id/signed-url` para uma attachment Telegram sem `file_key` persistido, o backend SHALL: chamar `getFile` com o `file_id`, baixar o conteúdo do CDN (URL `api.telegram.org/file/bot<TOKEN>/<file_path>`), fazer upload pra MinIO no path `{accountId}/{inboxId}/{messageId}/{filename}`, gravar `file_key` na attachment e retornar URL assinado.

#### Scenario: Primeira visualização de foto

- **WHEN** `GET .../attachments/:id/signed-url` é chamado e `attachment.file_key IS NULL`
- **THEN** o backend resolve via `getFile` + download + upload para MinIO, atualiza `file_key`, retorna URL assinado (TTL 15min)

#### Scenario: Visualização subsequente usa cache

- **WHEN** `attachment.file_key IS NOT NULL`
- **THEN** o backend só gera URL assinada apontando pro path já armazenado, sem chamada Telegram

### Requirement: Outbound com Markdown→HTML e reply threading

`Channel::Telegram.SendOutbound` SHALL converter Markdown em subset HTML Telegram (`<b>`, `<i>`, `<u>`, `<s>`, `<code>`, `<pre>`, `<a>`), setar `parse_mode=HTML`, suportar `reply_to_message_id` (quando `content_attributes.in_reply_to_source_id` presente), suportar `reply_markup.inline_keyboard` (quando `content_attributes.buttons` presente).

#### Scenario: Envio texto com negrito e link

- **WHEN** o conteúdo é `**Olá** [doc](https://x.com)`
- **THEN** o POST para `sendMessage` usa `text="<b>Olá</b> <a href=\"https://x.com\">doc</a>"` + `parse_mode=HTML`

#### Scenario: Envio com reply_to

- **WHEN** a mensagem tem `content_attributes.in_reply_to_source_id = 123` e source_id bate com uma mensagem anterior da mesma conversa
- **THEN** o POST inclui `reply_to_message_id=123`

#### Scenario: Envio com inline keyboard

- **WHEN** `content_attributes.buttons = [[{text:"Sim", callback_data:"yes"}]]`
- **THEN** o POST inclui `reply_markup={"inline_keyboard":[[{"text":"Sim","callback_data":"yes"}]]}` e a resposta do botão vira `callback_query` tratada como nova mensagem

#### Scenario: Tag HTML não suportada

- **WHEN** o Markdown gera uma tag fora do subset (ex: `<script>`)
- **THEN** a tag é strippada via `bluemonday.Policy` antes do envio

### Requirement: Delete webhook ao deletar canal

Quando um canal Telegram é deletado (soft delete do inbox ou hard delete), o backend SHALL chamar `deleteWebhook` no Telegram Bot API para não receber entregas órfãs.

#### Scenario: Delete canal Telegram

- **WHEN** inbox Telegram é deletada
- **THEN** o backend chama `POST api.telegram.org/bot<token>/deleteWebhook` antes (ou como parte) da remoção; falha no Telegram NÃO impede o delete local (log `warn`)

### Requirement: Segredos nunca retornam após criação

GET de canal Telegram SHALL omitir `bot_token_ciphertext` e `secret_token_ciphertext`. Response contém `botName`, `webhookIdentifier` (opaco, seguro expor), `createdAt`, `updatedAt`.
