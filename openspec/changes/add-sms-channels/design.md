## Context

SMS é um canal maduro mas fragmentado: cada provedor tem REST/webhook/signature próprios. Chatwoot resolve isso com dois tipos distintos (`Channel::Sms` genérico no shape Bandwidth + `Channel::TwilioSms`), o que gera duplicação. No Brasil, Zenvia domina o enterprise (Itaú, Magalu) e não é nativo em nenhuma das duas.

Queremos um único `Channel::Sms` com abstração por provider atrás da interface `SMSProvider`, cobrindo MVP com Twilio, Bandwidth e Zenvia. A seleção é por canal (coluna `provider`), credenciais ficam em `provider_config` JSONB encriptado. Reuso total do `channel.Channel` + `reauth.Tracker` + MinIO introduzidos na Fase 1 / WhatsApp.

Restrições principais:

- **Encryption ao rest**: `crypto/kek` para `provider_config` (que inclui `auth_token`, `api_secret`, etc).
- **Idempotência**: `messages.source_id = <provider_message_id>` (SID no Twilio, messageId no Bandwidth, id no Zenvia).
- **E.164**: normalização via `github.com/nyaruka/phonenumbers`, default region BR.
- **Webhook signature heterogêneo**: cada provider valida de jeito diferente — encapsular por impl.

## Goals / Non-Goals

**Goals:**

- `Channel::Sms` com três providers plugáveis (Twilio, Bandwidth, Zenvia) via interface `Provider`.
- Inbound webhooks por provider com verificação de assinatura isolada por impl.
- Outbound unificado com dispatch por `provider`; source_id do provider persistido.
- Suporte MMS (imagem, áudio, vídeo, documento) via URL pública no request — Twilio e Bandwidth nativos; Zenvia limitado ao plano.
- Delivery status callbacks mapeando pra `message.status` (`sent|delivered|failed`).
- E.164 phone normalization (default BR) com upsert de `contact` por `phone_e164`.

**Non-Goals:**

- Curto-número / short code provisioning (delegado ao operador manual).
- Rastreio de custo por mensagem (fora do MVP).
- SMS verification / OTP flows (não é caso de uso de atendimento).
- Outros providers brasileiros (Vivo/Claro direto, Wavy, Infobip, TotalVoice) — seguem o mesmo shape `Provider` e entram em follow-up.
- RCS (Rich Communication Services) — outro canal, não SMS.
- Group SMS — não existe no Brasil; Twilio suporta mas fica fora.

## Decisions

### D1 — Um tipo `Channel::Sms` com discriminador `provider`

**Escolhido:** Tabela única `channels_sms` com coluna `provider` (`twilio|bandwidth|zenvia`), `phone_number` (E.164, unique por account/inbox), `provider_config` JSONB ciphertext via KEK (estrutura varia por provider), `webhook_identifier` opaco para roteamento de webhook.

**Porquê:** Chatwoot separou `Channel::Sms` e `Channel::TwilioSms` por razões históricas (o Twilio SDK também serve WhatsApp). Em Go, não temos esse acoplamento — interface `Provider` resolve. Tabela única evita explosão de tipos. Alternativa com `channels_sms_twilio`/`_bandwidth`/`_zenvia` triplicaria a camada repo sem ganho.

### D2 — Interface `Provider` + registry

**Escolhido:** Pacote `internal/channel/sms/` expõe:

```go
type Provider interface {
    Name() string // "twilio" | "bandwidth" | "zenvia"
    VerifyWebhook(r *http.Request, channel *ChannelSMS) error
    ParseInbound(r *http.Request) (*InboundMessage, error)
    Send(ctx context.Context, channel *ChannelSMS, out *OutboundMessage) (sourceID string, err error)
    ParseDeliveryStatus(r *http.Request) (sourceID string, status string, err error)
}
```

Registry em `sms.Registry` mapeia `"twilio" → twilio.New(...)`, etc. O handler do webhook faz:

```go
p := sms.Registry.Get(channel.Provider)
if err := p.VerifyWebhook(r, channel); err != nil { /* 401 */ }
msg, err := p.ParseInbound(r)
```

**Porquê:** Isola quirks por provider sem `switch` espalhado. Segue o padrão de `channel.Registry` já existente para WhatsApp/Telegram. Testes com fake provider ficam simples.

### D3 — `provider_config` JSONB com shape discriminado

**Escolhido:** Estrutura validada por DTO na criação:

- `twilio`: `{account_sid, auth_token, messaging_service_sid?}`
- `bandwidth`: `{account_id, application_id, basic_auth_user, basic_auth_pass}`
- `zenvia`: `{api_token}` (bearer único)

Inteiro `provider_config` é serializado → AES-GCM via KEK → `provider_config_ciphertext`. Decriptado em memória só nas chamadas outbound/verify.

**Porquê:** Mesmo padrão do email (`provider_config`). Dá espaço para adicionar provider novo sem migration de schema. Alternativa com colunas dedicadas por provider explodiria a tabela; alternativa 100% env var força reconfig em deploy (anti-multi-tenant).

### D4 — Rotas webhook por provider, um handler

**Escolhido:** Rotas separadas para facilitar matching de signature (cada provider envia campos distintos):

- `POST /webhooks/sms/twilio/:identifier`
- `POST /webhooks/sms/bandwidth/:identifier`
- `POST /webhooks/sms/zenvia/:identifier`

Handler `sms_webhook_handler.go` carrega o canal por `:identifier`, verifica que `channel.provider` bate com o path, chama `Provider.VerifyWebhook`, depois `ParseInbound` ou `ParseDeliveryStatus`.

**Porquê:** URL distinta facilita debug e log. Providers mandam callbacks em paths diferentes de qualquer jeito (Twilio manda status em outra URL setada no send, Bandwidth idem). Alternativa com rota única + roteamento por body seria mais frágil.

### D5 — Verificação de assinatura encapsulada por impl

**Escolhido:**

- **Twilio**: `X-Twilio-Signature` = HMAC-SHA1(auth_token, URL completa + concat ordenado de params). Lib oficial `twilio/twilio-go` expõe helper; implementamos direto (dependência pesada pra só isso).
- **Bandwidth**: HTTP Basic Auth (usuário + senha configurados no provisioning). Simples.
- **Zenvia**: `X-Zenvia-Signature` = HMAC-SHA256(api_secret, body). Cabeçalho documentado na API v2.

Toda verify acontece antes de parse; falha → `401` silencioso com log `warn`.

**Porquê:** Cada provider tem seu ritual. Não há abstração útil em cima — tentar unificar viraria ifs-sobre-ifs. Isolamento em `twilio.go`/`bandwidth.go`/`zenvia.go` mantém teste local por provider.

### D6 — E.164 normalization obrigatória via `nyaruka/phonenumbers`

**Escolhido:** Toda string de telefone (inbound, outbound, contact lookup) passa por `phonenumbers.Parse(raw, "BR")` → `phonenumbers.Format(p, E164)` → `+5511999999999`. Fallback: se parse falha, persistir bruto com flag `phone_invalid=true` no contact e logar warning.

**Porquê:** Contact matching cross-canal (WhatsApp, SMS) exige shape canônico. `phonenumbers` é port do libphonenumber do Google, padrão de facto em Go. Chatwoot usa `phonelib` (Ruby wrapper do libphonenumber) com default BR — espelhamos a convenção.

### D7 — MMS via URL pública (não pre-download)

**Escolhido:** Inbound com mídia: Twilio envia `MediaUrl0..N` no body; Bandwidth `message.media[]`; Zenvia `content[].payload.mediaUrl`. Fluxo:

1. Stream da URL para MinIO no path `{accountId}/{inboxId}/{messageId}/{filename}` (sem presign, chamada interna).
2. Criar `attachment` row com `file_key` final.
3. Deletar a URL temporária só do ponto de vista da gente (Twilio retém 12h; Bandwidth 3 dias; Zenvia TTL varia).

Outbound com anexo: o agente sobe arquivo em MinIO (fluxo presigned existente), backend gera URL assinada de 24h, passa no request do provider.

**Porquê:** Espelha o path já usado em WhatsApp (`internal/channel/whatsapp/media.go`). Evita dependência de CDN externa ao responder. Alternativa lazy (como Telegram) não se aplica: a maior parte das SMS tem no máximo 1 mídia e o download é rápido.

### D8 — Delivery status via callback opcional

**Escolhido:** No outbound, setar `StatusCallback` (Twilio), `callbackUrl` (Bandwidth), `notificationUrl` (Zenvia) apontando pra `/webhooks/sms/<provider>/:identifier/status`. Handler extrai `sourceID` + `status`, atualiza `messages.status` para `sent|delivered|failed` e emite evento realtime `message.status_updated`.

Se o callback não chega em 5min, marcar `status=sent` (assumido entregue com TTL); não é erro.

**Porquê:** Dá UX de conversa "entregue/lida" tipo WhatsApp sem custo extra. Callback opcional: provedores que não entregam não quebram nada. TTL em 5min evita ficar permanente em `pending`.

### D9 — Outbound síncrono + fallback asynq

**Escolhido:** Mesma decisão de WhatsApp/Telegram. Envio tenta sync com timeout 10s. Se 5xx/429/timeout → enfileira `channel:sms:send` task no asynq com backoff padrão (1s, 5s, 30s, 2m, 10m). `reauth.Tracker` incrementa em auth errors (`401`/`403`) para disparar `requires_reauth`.

**Porquê:** Consistência total com outros canais. Sync dá feedback imediato no 90% caso feliz; async absorve picos. Tracker evita marcar reauth por erro transitório.

## Risks / Trade-offs

- **[Risco]** Auth token vaza em logs → nunca logar body do webhook inteiro; usar `crypto/subtle.ConstantTimeCompare` no verify; `provider_config_ciphertext` nunca retorna em GETs.
- **[Risco]** Replay de webhook (provider re-envia) → idempotência por `source_id` no `messages` resolve; delivery status upserts idempotentes por `source_id + status`.
- **[Risco]** Zenvia API v2 pode mudar (provider relativamente novo em contrato estável) → encapsular toda chamada em `zenvia.go`, testes de contrato com fixture real, fácil adaptar.
- **[Risco]** Telefone BR sem `+55` quebra matching contact cross-canal → normalização obrigatória no entry point, contact `phone_e164` é unique.
- **[Trade-off]** Ordem dos params no Twilio signature exige canonicalização exata (sorted) → helper `twilio.canonicalParams` com teste dedicado.
- **[Trade-off]** Bandwidth usa Basic Auth como "signature" — menos seguro que HMAC mas é o que o provider expõe; docs alertam para gerar senha forte.
- **[Trade-off]** MMS streaming pode estourar timeout de webhook (5s Twilio) → processamento inbound é fire-and-forget (enfileira task `channel:sms:ingest`), handler responde `200` cedo.

## Migration Plan

1. Deploy migration `0011_channels_sms.sql`.
2. Admin provisiona canal via `POST /api/v1/accounts/:aid/inboxes/sms` com `{provider, phoneNumber, providerConfig}`. Backend valida credencial fazendo chamada trivial (Twilio `accounts.get`, Bandwidth `GET /applications/:id`, Zenvia `GET /v2/channels`).
3. Admin configura webhook no painel do provider apontando pra URL do elodesk (Twilio e Bandwidth não tem setWebhook API pra todas as contas; documentar no README).
4. Teste inbound: enviar SMS pro número configurado e ver conversa aparecer.
5. Teste outbound: agente responde, mensagem chega no celular.
6. Validar MMS: receber imagem; conferir attachment em MinIO; agente responde com imagem.
7. **Rollback**: `DROP TABLE channels_sms` + `git revert`; webhooks ficam pendurados no painel do provider (cliente desativa manualmente).

## Open Questions

- **Phone number sharing**: dois clientes usam o mesmo número `+55...` em accounts diferentes? Rejeitar (unique por `phone_e164` cross-account) ou permitir (unique por `account_id + phone_e164`)? Proponho **unique por account** — permite sandbox/prod no mesmo número sem colisão.
- **Twilio Messaging Service**: opcional — se setado no `provider_config`, override `phone_number` no envio (loadbalance entre números). Proponho suportar mas sem provisionar pelo elodesk.
- **Zenvia multi-canal**: Zenvia v2 unifica SMS + WhatsApp Business via mesma API (`channel=sms|whatsapp`). Aproveitar? Proponho **não** no MVP — WhatsApp já tem canal próprio (wzap + futuramente BSP); Zenvia SMS só.
- **Opt-out automático (STOP/PARAR)**: Twilio bloqueia automaticamente após `STOP`; Bandwidth/Zenvia requerem implementação. Proponho logar no MVP, bloqueio explícito em follow-up com coluna `opted_out_at` no contact.
