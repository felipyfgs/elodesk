# backend-go-channel-meta-shared Specification

## Purpose

Shared utilities for Meta-family channels (Instagram, Facebook Page, Threads).
Factors HTTP client, signature verification, webhook types, and handshake
handling into `internal/channel/meta/` so each concrete channel consumes the
same primitives.

## Requirements

### Requirement: Client Graph API compartilhado

O backend SHALL expor `internal/channel/meta.Client` com baseURL `https://graph.facebook.com/{version}`, onde `version` é carregado de env `META_GRAPH_VERSION` (default `v22.0`). O client MUST injetar Bearer token via header `Authorization`, aplicar timeout de 10s por request, retornar erros tipados (`ErrMetaAuthFailed`, `ErrMetaRateLimit`, `ErrMetaPermanent`).

#### Scenario: GET Graph retorna 200

- **WHEN** `client.Get(ctx, "/{id}/messages", token)` é chamado
- **THEN** o request é feito contra `https://graph.facebook.com/v22.0/{id}/messages` com `Authorization: Bearer <token>` e retorna o payload decodificado

#### Scenario: GET Graph retorna 401/403

- **WHEN** o Graph API retorna erro `OAuthException` código 190/200
- **THEN** o client MUST retornar `ErrMetaAuthFailed` para que o caller acione `reauth.Tracker.RecordError`

### Requirement: Verificação de assinatura HMAC-SHA256

O backend SHALL fornecer `meta.VerifySignature(body []byte, header string, appSecret string) bool` que valida header `X-Hub-Signature-256: sha256=<hex>` via `hmac.Equal`. Em produção (sem env `META_ALLOW_UNSIGNED=true`), requests sem header OU com signature inválida MUST ser rejeitados com 401.

#### Scenario: Signature válida

- **WHEN** o body `raw` tem HMAC-SHA256 = `<x>` com `META_APP_SECRET` e o header vem `sha256=<x>`
- **THEN** `VerifySignature` retorna `true`

#### Scenario: Signature inválida

- **WHEN** o header `X-Hub-Signature-256` é ausente, vazio, ou com hash errado
- **THEN** `VerifySignature` retorna `false` em produção (com `META_ALLOW_UNSIGNED=true`, retorna `true` e loga warning)

### Requirement: Handshake `hub.challenge`

O backend SHALL fornecer `meta.HandleVerifyChallenge(c *fiber.Ctx, expectedToken string) error` que inspeciona query params `hub.mode=subscribe` + `hub.verify_token` + `hub.challenge`; se `verify_token` bate, responde `200 OK` com o `hub.challenge` como `text/plain`, senão `401 Unauthorized`.

#### Scenario: Handshake Meta Instagram passa

- **WHEN** Meta envia GET `/webhooks/instagram/:identifier?hub.mode=subscribe&hub.verify_token=<match>&hub.challenge=<x>`
- **THEN** o backend responde `200 OK` com body `<x>` e `Content-Type: text/plain`

#### Scenario: Handshake com token errado

- **WHEN** `hub.verify_token` não bate o esperado
- **THEN** o backend responde `401 Unauthorized`

### Requirement: Tipos normalizados Meta

O backend SHALL expor structs em `internal/channel/meta/types.go`: `Entry` (`ID, Time, Messaging []MessagingEntry, Standby []MessagingEntry, Changes []Change`), `MessagingEntry` (`Sender.ID, Recipient.ID, Timestamp, Message.Mid, Message.Text, Message.Attachments[], Message.IsEcho, Message.ReplyTo, Delivery, Read`), `Change` (`Field, Value json.RawMessage`). Todos os canais Meta MUST consumir esses tipos ao parsear webhooks.

#### Scenario: Parse de webhook Meta

- **WHEN** um webhook é recebido com payload `{object, entry: [{id, time, messaging: [...]}]}`
- **THEN** o parser desserializa para `[]Entry` e cada `MessagingEntry` carrega campos acessíveis por nome (sem chaves mágicas)
