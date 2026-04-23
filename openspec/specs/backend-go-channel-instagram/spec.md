# backend-go-channel-instagram Specification

## Purpose

Instagram DM channel (`Channel::Instagram`) built on the Meta Graph API v22+.
Supports long-lived token with proactive refresh, webhook verification with
`X-Hub-Signature-256`, message dedup via `message.mid`, echo delay, and
outbound delivery via `graph.instagram.com/{instagram_id}/messages`.

## Requirements

### Requirement: Tipo `Channel::Instagram`

O backend SHALL expor o tipo `Channel::Instagram` com tabela dedicada `channels_instagram` armazenando `instagram_id` (unique por account), `access_token_ciphertext` (AES-GCM via KEK), `expires_at`, `updated_at`. Não há `refresh_token` separado (Instagram usa in-place refresh).

#### Scenario: Criar canal Instagram

- **WHEN** `POST /api/v1/accounts/:aid/inboxes/instagram` com `{instagramId, accessToken}`
- **THEN** o sistema grava `access_token_ciphertext` e `expires_at = now + 60d`, cria `inboxes(channel_type='Channel::Instagram')` e retorna a inbox (sem token)

### Requirement: Webhook Instagram com handshake + signature + dedup

O backend SHALL aceitar:

- `GET /webhooks/instagram/:identifier` → `meta.HandleVerifyChallenge` com `INSTAGRAM_VERIFY_TOKEN`
- `POST /webhooks/instagram/:identifier` → `meta.VerifySignature`, parseia `Entry`, dedup por `message.mid` via `DedupLock`, processa mensagens

#### Scenario: Handshake válido

- **WHEN** Meta faz GET com `verify_token = INSTAGRAM_VERIFY_TOKEN`
- **THEN** retorna `200` + `hub.challenge`

#### Scenario: Mensagem nova dedupa

- **WHEN** POST com `messaging[0].message.mid = "m_XYZ"`
- **THEN** `DedupLock.Acquire("elodesk:meta:m_XYZ")` com TTL 24h; primeira chamada processa, duplicatas são silenciosamente ignoradas

#### Scenario: Mensagem echo atrasa 2s

- **WHEN** `messaging[0].message.is_echo = true`
- **THEN** o processamento é enfileirado em asynq com `ProcessIn(2s)` para aguardar a resposta da API outbound gerar `source_id`

### Requirement: Refresh proativo de access_token

Antes de qualquer chamada outbound ou scheduled action, se `expires_at - now < 10 days`, o backend SHALL chamar `GET graph.instagram.com/refresh_access_token?grant_type=ig_refresh_token&access_token=<current>`. Sucesso atualiza `access_token_ciphertext` + `expires_at`; falha incrementa `reauth.Tracker`.

#### Scenario: Token perto de expirar refresca

- **WHEN** `expires_at = now + 5 days` e um envio é solicitado
- **THEN** refresh é tentado antes do envio; novo token substitui o antigo; envio prossegue

#### Scenario: Refresh falha dispara reauth

- **WHEN** Graph retorna `OAuthException` em `/refresh_access_token`
- **THEN** `reauth.Tracker.RecordError("channel:instagram:<id>")` é chamado; threshold 1 imediatamente marca `requires_reauth=true` e emite evento `channel.reauth_required`

### Requirement: Outbound via Graph API v22

`Channel::Instagram.SendOutbound` SHALL POSTar em `graph.instagram.com/{instagram_id}/messages` com body `{"recipient":{"id":<contact_source_id>},"message":{...}}`. Texto usa `{"text":<content>}`; mídia usa `{"attachment":{"type":"image|video|audio|file","payload":{"url":<public_url>}}}`. `Authorization: Bearer <access_token>`.

#### Scenario: Envio texto

- **WHEN** agente envia mensagem texto
- **THEN** o sistema POSTa `{recipient:{id:<from>}, message:{text:"oi"}}`, extrai `messages[0].id` como `source_id`

#### Scenario: Envio mídia

- **WHEN** mensagem tem attachment com `file_key`
- **THEN** o sistema gera URL assinado MinIO (TTL 15min), POSTa `attachment.payload.url`, persiste `source_id`

### Requirement: Segredos nunca retornam após criação

GET de canal Instagram MUST omitir `access_token_ciphertext` e qualquer token. Response público contém só `instagramId`, `expiresAt`, `requiresReauth`, `createdAt`, `updatedAt`.
