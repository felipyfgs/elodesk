## 1. Migração e modelo

- [x] 1.1 Criar `backend/migrations/0010_channels_sms.sql` (id, inbox_id FK, provider enum, phone_number E.164, webhook_identifier unique, provider_config_ciphertext, messaging_service_sid, requires_reauth, created_at, updated_at, unique por account_id+phone_number)
- [x] 1.2 Adicionar struct `ChannelSMS` em `backend/internal/model/models.go` (ciphertext com `json:"-"`)
- [x] 1.3 Criar `backend/internal/repo/channel_sms_repo.go` com CRUD + `FindByWebhookIdentifier(id, provider)` + scopes por accountID
- [x] 1.4 Adicionar coluna `phone_e164` + índice unique (`account_id`, `phone_e164`) em `contacts` (incluído em `0010_channels_sms.sql`)

## 2. Packages base

- [x] 2.1 Adicionar dependências Go: `github.com/nyaruka/phonenumbers`
- [x] 2.2 Criar `backend/internal/channel/sms/types.go` com `InboundMessage`, `OutboundMessage`, `Provider` interface, `ChannelSMS` ref
- [x] 2.3 Criar `backend/internal/channel/sms/registry.go` — `Registry.Register(name, provider)`, `Registry.Get(name)`
- [x] 2.4 Criar `backend/internal/channel/sms/phone.go` — `NormalizeE164(raw, defaultRegion)` via `nyaruka/phonenumbers`; fallback `phone_invalid=true`
- [x] 2.5 Criar `backend/internal/channel/sms/config.go` — serialização/deserialização de `provider_config` (struct tipada por provider) + encrypt/decrypt via KEK

## 3. Provider Twilio

- [x] 3.1 Criar `backend/internal/channel/sms/twilio/twilio.go` — struct `Provider`, impl `Name()="twilio"`
- [x] 3.2 Implementar `VerifyWebhook` — HMAC-SHA1(auth_token, URL + params sorted); comparação `crypto/subtle.ConstantTimeCompare`
- [x] 3.3 Implementar `ParseInbound` — extrai `MessageSid`, `From`, `To`, `Body`, `NumMedia`, `MediaUrl0..N`, `MediaContentType0..N`
- [x] 3.4 Implementar `Send` — POST `Messages.json` com basic auth; suporta `MessagingServiceSid` se presente
- [x] 3.5 Implementar `ParseDeliveryStatus` — extrai `MessageSid`, `MessageStatus`, `ErrorCode`
- [x] 3.6 Implementar `ValidateCredentials(config)` — `GET /Accounts/{sid}.json`

## 4. Provider Bandwidth

- [x] 4.1 Criar `backend/internal/channel/sms/bandwidth/bandwidth.go`
- [x] 4.2 Implementar `VerifyWebhook` — HTTP Basic Auth contra `basic_auth_user`/`basic_auth_pass`
- [x] 4.3 Implementar `ParseInbound` — array JSON; extrai `message.id`, `message.from`, `message.to[0]`, `message.text`, `message.media[]`
- [x] 4.4 Implementar `Send` — POST `messages` com applicationId; mídias em `media[]` URL assinada MinIO
- [x] 4.5 Implementar `ParseDeliveryStatus` — parse array com `type=message-delivered|message-failed`
- [x] 4.6 Implementar `ValidateCredentials(config)` — `GET /users/{accountId}/applications/{appId}`

## 5. Provider Zenvia

- [x] 5.1 Criar `backend/internal/channel/sms/zenvia/zenvia.go`
- [x] 5.2 Implementar `VerifyWebhook` — HMAC-SHA256(api_secret, raw body) contra header `X-Zenvia-Signature`
- [x] 5.3 Implementar `ParseInbound` — extrai `id`, `from`, `to`, `contents[].text` ou `contents[].payload.mediaUrl`
- [x] 5.4 Implementar `Send` — POST `v2/channels/sms/messages` com header `X-API-TOKEN`
- [x] 5.5 Implementar `ParseDeliveryStatus` — parse `messageStatus.code`
- [x] 5.6 Implementar `ValidateCredentials(config)` — `GET /v2/channels` com token

## 6. Ingestão e persistência

- [x] 6.1 Criar `backend/internal/channel/sms/ingest.go` — função `IngestInbound(ctx, channel, inbound)` que: normaliza phone, upsert contact, ensure conversation, cria message + attachments
- [x] 6.2 Criar `backend/internal/channel/sms/media.go` — stream URL (com auth opcional) → MinIO path `{accountId}/{inboxId}/{messageId}/{filename}` + criar `attachment` row
- [x] 6.3 Criar task asynq `channel:sms:ingest` em `backend/internal/channel/sms/tasks.go`
- [x] 6.4 Criar task asynq `channel:sms:send` em `backend/internal/channel/sms/tasks.go` (fallback de 5xx/429)

## 7. Canal (`channel.Channel`)

- [x] 7.1 Criar `backend/internal/channel/sms/sms.go` implementando `channel.Channel`: `Type()="Channel::Sms"`, `SendOutbound` dispatcha via `Registry.Get(channel.Provider).Send`
- [x] 7.2 Integrar `reauth.Tracker` em `Send`: `401`/`403` → `RecordError(channel:sms:<id>)`

## 8. Handler webhook

- [x] 8.1 Criar `backend/internal/handler/sms_webhook_handler.go` com:
  - `POST /webhooks/sms/twilio/:identifier`
  - `POST /webhooks/sms/twilio/:identifier/status`
  - `POST /webhooks/sms/bandwidth/:identifier`
  - `POST /webhooks/sms/bandwidth/:identifier/status`
  - `POST /webhooks/sms/zenvia/:identifier`
  - `POST /webhooks/sms/zenvia/:identifier/status`
- [x] 8.2 Handler carrega canal por `:identifier`, valida que `channel.provider` bate com path (`404` se não), chama `Provider.VerifyWebhook`, enfileira task `channel:sms:ingest` e responde `200 OK` cedo
- [x] 8.3 Status handlers sincronizam `messages.status` e emitem `message.status_updated` no realtime

## 9. Provisioning

- [x] 9.1 Criar `backend/internal/handler/sms_inbox_handler.go` com `POST /api/v1/accounts/:aid/inboxes/sms`
- [x] 9.2 DTO `CreateSMSInboxReq` em `backend/internal/dto/sms.go` — discriminado por `provider`
- [x] 9.3 Handler chama `Registry.Get(provider).ValidateCredentials(config)` antes de persistir; falha → `400 invalid_credentials`
- [x] 9.4 Gerar `webhook_identifier` (16 bytes random base64url); persistir canal; retornar `{inboxId, webhookUrls:{primary, status}, phoneNumber}` para o admin configurar no painel do provider

## 10. Wiring

- [x] 10.1 Registrar `sms.Channel` no `channel.Registry` em `backend/internal/server/router.go`
- [x] 10.2 Inicializar `sms.Registry` no startup registrando `twilio.New`, `bandwidth.New`, `zenvia.New`
- [x] 10.3 Registrar rotas webhook + provisioning
- [x] 10.4 Registrar tasks asynq no worker

## 11. Testes

- [x] 11.1 `backend/internal/channel/sms/phone_test.go` — normalização BR (com/sem `+55`), US, formatos inválidos
- [x] 11.2 `backend/internal/channel/sms/twilio/twilio_test.go` — verify signature (válido/inválido/sorted params), ParseInbound (texto, MMS), Send (com/sem messaging_service)
- [x] 11.3 `backend/internal/channel/sms/bandwidth/bandwidth_test.go` — basic auth verify, parse array, send com mídia
- [x] 11.4 `backend/internal/channel/sms/zenvia/zenvia_test.go` — HMAC-SHA256 verify, parse v2 payload, send
- [x] 11.5 `backend/internal/channel/sms/ingest_test.go` — dedup por source_id, contact upsert cross-canal, conversation reuse
- [x] 11.6 Integration test — spin up Postgres + Redis + mock HTTP do provider; POST webhook → ver message gravada → send → ver chamada mock

## 12. Documentação

- [x] 12.1 `backend/README.md` seção SMS — como provisionar por provider (Twilio Console/Bandwidth Dashboard/Zenvia Portal), URLs de webhook, variáveis ENV (`DEFAULT_PHONE_REGION`)
- [x] 12.2 `README.md` raiz adiciona SMS à lista de canais suportados
- [x] 12.3 Swagger annotations em handlers novos; `make docs`

## 13. Validação ponta-a-ponta

- [ ] 13.1 **Twilio**: criar número trial, provisionar canal, enviar SMS pro número; ver conversa aparecer no frontend; agente responde; verificar no celular
- [ ] 13.2 **Twilio MMS**: enviar foto pro número; conferir attachment em MinIO e visualização no frontend
- [ ] 13.3 **Twilio status**: verificar callback muda `status` para `delivered` após o celular confirmar
- [ ] 13.4 **Bandwidth**: provisionar número sandbox, configurar Basic Auth no callback, ciclo completo
- [ ] 13.5 **Zenvia**: provisionar em ambiente sandbox Zenvia, testar inbound/outbound
- [ ] 13.6 **Reauth**: revogar auth_token Twilio no painel, enviar; conferir `requires_reauth=true` e evento realtime
- [ ] 13.7 **Cross-canal**: contact com mesmo `phone_e164` de WhatsApp recebe SMS e o frontend agrega; confirma que contact é o mesmo
