## Why

SMS mantém relevância no Brasil em dois cenários: (a) enterprise com volume (bancos, seguradoras, delivery — geralmente via Zenvia/Vivo/Claro) e (b) fallback global via Twilio ou Bandwidth.com quando empresas operam em múltiplos mercados. Chatwoot tem dois canais SMS históricos: `Channel::Sms` (genérico, shape Bandwidth.com) e `Channel::TwilioSms` (via Twilio SDK, que também serve WhatsApp). Nenhum provedor BR (Zenvia) é nativo.

Esta change adiciona `Channel::Sms` com abstração por provider (Twilio, Bandwidth, Zenvia) por trás de uma interface `SMSProvider`, permitindo escolher por canal. Fica pronta também pra futuras extensões (Vivo, Wavy, Infobip).

## What Changes

- Nova tabela `channels_sms` com coluna `provider` (`twilio|bandwidth|zenvia`) + `provider_config` JSONB para credenciais específicas de cada provider.
- Novo subpacote `internal/channel/sms/` com interface `Provider` e três implementações: `twilio.go`, `bandwidth.go`, `zenvia.go`.
- Suporte a MMS (mídia via URL pública no request) para Twilio e Bandwidth; Zenvia suporta mídia em planos específicos.
- Inbound webhooks: `/webhooks/sms/twilio/:identifier`, `/webhooks/sms/bandwidth/:identifier`, `/webhooks/sms/zenvia/:identifier` com verificação de assinatura por provider.
- Outbound via SDK/REST específico; `source_id` do provider persistido.
- Delivery status callbacks atualizando `message.status` (sent/delivered/failed).
- E.164 phone number normalization via lib `nyaruka/phonenumbers` (padrão BR em Chatwoot).

## Capabilities

### New Capabilities
- `backend-go-channel-sms`: canal SMS multi-provider com interface `Provider` abstraindo Twilio, Bandwidth, Zenvia; parsing de webhook por provider; outbound unificado; delivery status callbacks; suporte BR via Zenvia.

### Modified Capabilities
- Nenhuma.

## Impact

- **Código novo**: `backend/migrations/0011_channels_sms.sql`, `backend/internal/channel/sms/{sms,provider,twilio,bandwidth,zenvia,phone}.go`, `backend/internal/repo/channel_sms_repo.go`, `backend/internal/handler/sms_webhook_handler.go`, DTOs.
- **Dependências Go**: `github.com/nyaruka/phonenumbers` para normalização E.164. Twilio SDK Go (`github.com/twilio/twilio-go`) OPCIONAL — avaliar se REST direto é suficiente; por ora sem dependência.
- **ENV**: nenhum global (credenciais per-canal).
- **Rollback**: `DROP TABLE channels_sms` + `git revert`.

## Riscos e mitigações

- **Webhook signature por provider é heterogêneo** → `X-Twilio-Signature`, Bandwidth usa basic auth, Zenvia usa `X-Zenvia-Signature`. Cada impl encapsula o próprio verify.
- **Phone normalization errada** (ex: "11999999999" vs "+5511999999999") → `phonenumbers.Parse("BR")` como default; normalizar sempre antes de `contact.upsert`.
- **Rate limits dos providers** → usar backoff por default do `reauth.Tracker`; Twilio rate-limit 429 → tratar como retryable.
- **Zenvia API nova (v2)** com token bearer + `X-Zenvia-Signature` — implementar MVP e ajustar se provider refatorar.
- **Multi-country pricing** fora de escopo — não rastreamos custo por mensagem.
