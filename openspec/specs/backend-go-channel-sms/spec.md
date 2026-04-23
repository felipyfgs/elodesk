## ADDED Requirements

### Requirement: Tipo `Channel::Sms` multi-provider

O backend SHALL expor o tipo `Channel::Sms` com tabela `channels_sms` contendo `provider` (`twilio|bandwidth|zenvia`), `phone_number` (E.164, unique por `account_id`), `webhook_identifier` (token opaco), `provider_config_ciphertext` (AES-GCM via KEK) contendo credenciais específicas do provider, `messaging_service_sid` opcional (Twilio), `requires_reauth` bool, timestamps. A provisão é feita via `POST /api/v1/accounts/:aid/inboxes/sms`.

#### Scenario: Criar canal Twilio

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/sms` com `{provider:"twilio", phoneNumber:"+5511988887777", providerConfig:{accountSid, authToken, messagingServiceSid?}}`
- **THEN** o backend valida a credencial chamando `GET https://api.twilio.com/2010-04-01/Accounts/{accountSid}.json` com basic auth; em sucesso, gera `webhook_identifier` (16 bytes random base64url), serializa `providerConfig` → AES-GCM via KEK → `provider_config_ciphertext`, cria linha em `channels_sms` e `inboxes(channel_type='Channel::Sms')`; resposta omite credenciais

#### Scenario: Criar canal Bandwidth

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/sms` com `{provider:"bandwidth", phoneNumber:"+14155551234", providerConfig:{accountId, applicationId, basicAuthUser, basicAuthPass}}`
- **THEN** o backend valida chamando `GET https://messaging.bandwidth.com/api/v2/users/{accountId}/applications/{applicationId}` com Basic Auth; sucesso persiste canal; falha retorna `400 invalid_credentials`

#### Scenario: Criar canal Zenvia

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/sms` com `{provider:"zenvia", phoneNumber:"+5511988887777", providerConfig:{apiToken}}`
- **THEN** o backend valida chamando `GET https://api.zenvia.com/v2/channels` com header `X-API-TOKEN`; sucesso persiste canal

#### Scenario: Credenciais inválidas rejeitam criação

- **WHEN** a validação da credencial retorna `401`/`403`
- **THEN** resposta `400 Bad Request` com código `invalid_credentials`; NADA é persistido

#### Scenario: Telefone duplicado na mesma account

- **WHEN** já existe `channels_sms` com o mesmo `phone_number` no mesmo `account_id`
- **THEN** resposta `409 Conflict` com código `phone_already_registered`

### Requirement: Interface `Provider` e registry

O pacote `internal/channel/sms` SHALL expor uma interface `Provider` com métodos `Name() string`, `VerifyWebhook(r, channel) error`, `ParseInbound(r) (*InboundMessage, error)`, `Send(ctx, channel, out) (sourceID string, err error)`, `ParseDeliveryStatus(r) (sourceID string, status string, err error)`. Um registry em `sms.Registry` SHALL mapear o nome do provider para a instância. O handler SHALL despachar via `Registry.Get(channel.Provider)`.

#### Scenario: Registry resolve provider conhecido

- **WHEN** um inbound chega em `/webhooks/sms/twilio/:identifier` e o canal tem `provider="twilio"`
- **THEN** o handler chama `Registry.Get("twilio")` que retorna a impl `twilio.Provider`, e o processamento segue com essa impl

#### Scenario: Provider desconhecido

- **WHEN** um canal é criado com `provider` fora de `{twilio, bandwidth, zenvia}`
- **THEN** o DTO rejeita com `400 Bad Request` código `unsupported_provider`

### Requirement: Webhooks de inbound por provider

O backend SHALL expor três rotas webhook para recepção de SMS:

- `POST /webhooks/sms/twilio/:identifier` — verifica header `X-Twilio-Signature` (HMAC-SHA1 do auth_token sobre URL + params sorted)
- `POST /webhooks/sms/bandwidth/:identifier` — verifica Basic Auth contra `basic_auth_user`/`basic_auth_pass` do canal
- `POST /webhooks/sms/zenvia/:identifier` — verifica header `X-Zenvia-Signature` (HMAC-SHA256 do `api_secret` sobre body raw)

Todas resolvem o canal por `:identifier`, validam que `channel.provider` bate com o path, e em sucesso chamam `Provider.ParseInbound` + persistência. Falha de signature → `401 Unauthorized` silencioso, log em `warn`.

#### Scenario: Signature Twilio válida

- **WHEN** o header `X-Twilio-Signature` bate o HMAC-SHA1 esperado
- **THEN** o webhook processa a mensagem inbound e retorna `200 OK`

#### Scenario: Signature Twilio inválida

- **WHEN** o header está ausente OU diferente do esperado
- **THEN** resposta `401 Unauthorized`; nada é persistido; log `warn` com `component="sms_webhook"` e `provider="twilio"`

#### Scenario: Provider no path não bate com canal

- **WHEN** request em `/webhooks/sms/twilio/:identifier` mas `channel.provider="bandwidth"`
- **THEN** resposta `404 Not Found`

### Requirement: Persistência de mensagem inbound com E.164 e contact upsert

Para cada inbound parseado, o backend SHALL normalizar o número de origem via `phonenumbers.Parse(raw, "BR")` → `phonenumbers.Format(p, E164)`, fazer upsert de `contacts` por `phone_e164` + `account_id`, resolver/criar `conversation` aberta com esse contact, e criar `message` com `source_id=<provider_message_id>`, `content`, `content_type=ContentTypeText`.

#### Scenario: Número BR sem prefixo

- **WHEN** Twilio entrega `From="11988887777"` (sem `+55`)
- **THEN** o backend normaliza para `+5511988887777`, upserta `contact.phone_e164="+5511988887777"`

#### Scenario: Idempotência por source_id

- **WHEN** o mesmo webhook chega duas vezes com mesmo `MessageSid`
- **THEN** a segunda tentativa detecta `messages.source_id` existente no mesmo inbox e retorna `200 OK` sem duplicar

#### Scenario: Parse de número falha

- **WHEN** `phonenumbers.Parse` retorna erro (formato inválido)
- **THEN** o contact é criado com `phone_e164=NULL`, `phone_raw=<original>`, `phone_invalid=true`; log `warn`; mensagem persistida normalmente

### Requirement: Suporte a MMS (mídia inbound)

Quando o webhook traz mídia (`MediaUrl0..N` Twilio, `message.media[]` Bandwidth, `content[].payload.mediaUrl` Zenvia), o backend SHALL fazer stream de cada mídia para MinIO no path `{accountId}/{inboxId}/{messageId}/{filename}`, criar uma linha em `attachments` com `file_key` persistido, `file_type` derivado do MIME, `extension`. O download usa autenticação do provider quando exigido (Twilio URLs são públicas temporariamente; Bandwidth exige Basic Auth).

#### Scenario: Twilio MMS com imagem

- **WHEN** o inbound traz `NumMedia=1`, `MediaUrl0=https://api.twilio.com/...`, `MediaContentType0=image/jpeg`
- **THEN** o backend faz GET da URL, stream → MinIO path `{accountId}/{inboxId}/{messageId}/image.jpg`, cria `attachment(file_type=FileTypeImage, file_key=<path>, extension="jpg")`

#### Scenario: Múltiplas mídias

- **WHEN** `NumMedia=3` com 3 URLs
- **THEN** 3 attachments são criadas, uma por mídia, todas ligadas à mesma `message`

#### Scenario: Falha ao baixar mídia

- **WHEN** o GET da URL retorna `404`/`5xx`
- **THEN** a mensagem é persistida sem esse attachment; log `warn`; outros attachments seguem

### Requirement: Outbound unificado com dispatch por provider

Quando `Channel::Sms.SendOutbound(msg)` é chamado, o backend SHALL carregar `channel.provider`, chamar `Registry.Get(provider).Send(ctx, channel, out)` e persistir o `sourceID` retornado em `messages.source_id`. O envio inclui texto + lista de URLs de mídia (quando presentes). Dispatch:

- `twilio`: `POST https://api.twilio.com/2010-04-01/Accounts/{sid}/Messages.json` com `From=<phone_number>` ou `MessagingServiceSid=<sid>`, `To=<e164>`, `Body=<text>`, `MediaUrl=<url1>,<url2>`, `StatusCallback=<url>`.
- `bandwidth`: `POST https://messaging.bandwidth.com/api/v2/users/{accountId}/messages` com `applicationId`, `from`, `to`, `text`, `media[]`, `tag` (para callback correlation).
- `zenvia`: `POST https://api.zenvia.com/v2/channels/sms/messages` com `from`, `to`, `contents[]`.

#### Scenario: Envio Twilio texto

- **WHEN** agente responde texto em canal Twilio
- **THEN** o backend POSTa em `Messages.json`, persiste `source_id=<SID retornado>`, `status=sent`

#### Scenario: Envio Bandwidth com mídia

- **WHEN** a mensagem tem attachment em MinIO
- **THEN** o backend gera URL assinada (TTL 24h) pra cada attachment e inclui no array `media[]` do request Bandwidth

#### Scenario: Twilio 429 rate limit

- **WHEN** o send retorna `429 Too Many Requests`
- **THEN** a mensagem é enfileirada em asynq task `channel:sms:send` com backoff `1s, 5s, 30s, 2m, 10m`; `status=pending` até sucesso

#### Scenario: Auth error dispara reauth

- **WHEN** o send retorna `401`/`403` (credencial revogada)
- **THEN** `reauth.Tracker.RecordError(channel:sms:<id>)` é chamado; atingindo threshold, `requires_reauth=true` é gravado e evento realtime `channel.reauth_required` é emitido

### Requirement: Delivery status callbacks

O backend SHALL expor endpoints de status por provider (mesmo `:identifier` do canal):

- `POST /webhooks/sms/twilio/:identifier/status` — parse `MessageSid`, `MessageStatus` (`queued|sent|delivered|failed`).
- `POST /webhooks/sms/bandwidth/:identifier/status` — parse `message.id`, `type` (`message-delivered|message-failed`).
- `POST /webhooks/sms/zenvia/:identifier/status` — parse `messageId`, `messageStatus.code` (`SENT|DELIVERED|NOT_DELIVERED`).

Cada callback atualiza `messages.status` (`sent|delivered|failed`) e emite evento realtime `message.status_updated`.

#### Scenario: Twilio status delivered

- **WHEN** `POST /webhooks/sms/twilio/:identifier/status` com `MessageStatus=delivered`, `MessageSid=<sid>`
- **THEN** `messages.status="delivered"` para o match de `source_id=<sid>`; evento `message.status_updated` publicado no canal realtime da conversa

#### Scenario: Failure com error_code

- **WHEN** Twilio retorna `MessageStatus=failed`, `ErrorCode=30003`
- **THEN** `messages.status="failed"`, `messages.meta->>'error_code'="30003"` persistido

#### Scenario: Status callback sem match

- **WHEN** o `MessageSid` não bate nenhuma `messages.source_id`
- **THEN** resposta `200 OK` (não falhar callback), log `info` com SID

### Requirement: Processamento inbound assíncrono

O handler de webhook SHALL responder `200 OK` imediatamente após verificar assinatura, enfileirando o parse/persistência em task asynq `channel:sms:ingest` com payload `{channelID, provider, rawBody, headers}`. Isso evita timeout de webhook (Twilio exige resposta em 5s).

#### Scenario: Response rápido

- **WHEN** inbound Twilio chega
- **THEN** o backend verifica signature, enfileira task em Redis e retorna `200 OK` em <500ms; o processamento (persist de contact/conversation/message/attachment) roda no worker

#### Scenario: Worker falha

- **WHEN** o worker falha ao persistir (DB down, etc)
- **THEN** asynq re-tenta com backoff padrão; após 10 falhas move para DLQ `channel:sms:ingest:dlq` e loga `error`

### Requirement: Segredos nunca retornam após criação

GET de canal SMS SHALL omitir `provider_config_ciphertext` e qualquer campo sensível (auth_token, api_token, basic_auth_pass). Response contém `provider`, `phoneNumber`, `webhookIdentifier` (opaco, seguro expor), `requiresReauth`, `createdAt`, `updatedAt`, mais campos públicos do provider config (ex: `accountSid` Twilio pode ser exposto; `authToken` nunca).

#### Scenario: GET não vaza credenciais

- **WHEN** `GET /api/v1/accounts/:aid/inboxes/:id` é chamado para um canal SMS
- **THEN** o response contém `provider`, `phoneNumber`, `webhookIdentifier` mas NUNCA `authToken`, `apiToken`, `basicAuthPass`

### Requirement: Normalização E.164 cross-canal

Toda string de telefone entrando pelo `Channel::Sms` (inbound, DTO de criação, agente digitando número destino) SHALL ser normalizada via `nyaruka/phonenumbers` com região default `"BR"` (configurável por env `DEFAULT_PHONE_REGION`). Contacts SHALL ter `phone_e164` unique por `account_id`.

#### Scenario: Unificação contact WhatsApp + SMS

- **WHEN** um contact já existe com `phone_e164="+5511988887777"` (criado via WhatsApp) e chega SMS do mesmo número
- **THEN** a nova mensagem é associada ao MESMO contact (não cria duplicata); a `conversation` é nova (por inbox diferente)

#### Scenario: DEFAULT_PHONE_REGION configurável

- **WHEN** env `DEFAULT_PHONE_REGION="US"` e inbound traz `From="4155551234"` (sem `+1`)
- **THEN** normalização resulta em `+14155551234`
