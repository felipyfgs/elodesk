## ADDED Requirements

### Requirement: WzapHttpClient com tipos gerados do Swagger

O backend SHALL incluir `WzapHttpClient` (@Injectable) que envolve `HttpModule` com `baseURL` e `Authorization` (admin token via env) injetados. Os tipos TS SHALL ser gerados automaticamente de [wzap/docs/swagger.yaml](../../../../wzap/docs/swagger.yaml) via script `pnpm gen:wzap` usando `openapi-typescript`. O client MUST expor métodos tipados para: `createSession`, `connectSession`, `disconnectSession`, `getQr`, `getStatus`, `deleteSession`, `createWebhook`, `sendText`, `sendMedia`, `editMessage`, `deleteMessage`, `reactMessage`, `markRead`, `getMedia`.

#### Scenario: regenerar tipos após update do swagger

- **WHEN** o desenvolvedor roda `pnpm gen:wzap`
- **THEN** o arquivo `src/wzap/wzap.schema.d.ts` é atualizado a partir do swagger mais recente

#### Scenario: CI detecta drift

- **WHEN** `pnpm gen:wzap` roda em CI
- **THEN** se houver diff não comitado no `wzap.schema.d.ts`, o CI falha

#### Scenario: token admin nunca exposto em resposta

- **WHEN** `WzapHttpClient` propaga erro do wzap para o caller
- **THEN** o header `Authorization` nunca aparece em logs nem em respostas de erro

### Requirement: WzapWsClient por sessão com reconexão

O backend SHALL manter um `WzapWsClient` por `ChannelWhatsapp` com status `CONNECTED`, conectando em `{WZAP_WS_URL}/ws/:sessionId?token=...`. Reconexão MUST seguir backoff exponencial (1s, 2s, 4s, …, max 30s). Eventos recebidos SHALL ser publicados via `EventEmitter2` para o `WzapEventService`.

#### Scenario: reconexão após queda

- **WHEN** conexão WS do wzap cai
- **THEN** o client tenta reconectar com backoff, com tentativas em 1s, 2s, 4s, 8s, 16s, 30s, 30s...

#### Scenario: convergência com webhook

- **WHEN** mesmo evento chega via WS e via webhook
- **THEN** apenas um `Message` é persistido (idempotência por `sourceId` unique)

### Requirement: Webhook receiver com validação HMAC

O backend SHALL expor `POST /wzap/webhook/:channelId` (rota pública, sem auth JWT). O handler MUST validar o header `X-Wzap-Signature` contra `HMAC-SHA256(body, channel.webhookSecret)` usando `crypto.timingSafeEqual`. Body válido SHALL ser enfileirado em BullMQ queue `wzap-events` e a rota responde 200 imediatamente (< 50ms).

#### Scenario: assinatura válida

- **WHEN** webhook chega com assinatura correta
- **THEN** o evento é enfileirado e a rota retorna 200

#### Scenario: assinatura inválida

- **WHEN** webhook chega com assinatura incorreta ou ausente
- **THEN** retorna 401 e nada é enfileirado

#### Scenario: channelId inexistente

- **WHEN** `:channelId` não existe na tabela `ChannelWhatsapp`
- **THEN** retorna 404

### Requirement: WzapEventService roteia 47 EventTypes

O worker BullMQ SHALL consumir a queue `wzap-events` e invocar `WzapEventService.handle(channelId, event)` que roteia o `event.type` (dos 47 `EventType`s definidos em [wzap/internal/model/events.go](../../../../wzap/internal/model/events.go)) para handler específico. Eventos não mapeados SHALL ser gravados em log com nível `warn` e tabela `AuditEvent` para triagem, sem falhar o job.

#### Scenario: evento Message é processado

- **WHEN** chega evento `type=Message`
- **THEN** `MessageInboundHandler` é chamado e persiste `Contact`/`ContactInbox`/`Conversation`/`Message`

#### Scenario: evento desconhecido

- **WHEN** chega evento com `type` não mapeado
- **THEN** é logado como warn, gravado em `AuditEvent` e o job é ACKed sem retry infinito

#### Scenario: falha em handler

- **WHEN** handler lança exceção
- **THEN** BullMQ retenta até 5 vezes com backoff exponencial (1s, 5s, 30s, 2m, 10m)

### Requirement: Idempotência por Message.sourceId

A tabela `Message` SHALL ter `sourceId` (string nullable) com unique index parcial `WHERE sourceId IS NOT NULL`. Convenção: `sourceId = "WAID:" + wzapMessageID` para mensagens WA. Handler inbound MUST usar `upsert` por `(inboxId, sourceId)` ao invés de `insert` para evitar duplicatas de webhooks retries.

#### Scenario: webhook duplicado não cria duplicata

- **WHEN** o mesmo evento `Message` chega duas vezes (ex: retry do wzap)
- **THEN** existe apenas uma linha em `Message` com aquele `sourceId`

#### Scenario: mensagens OUTGOING pendentes não violam unique

- **WHEN** múltiplas mensagens OUTGOING estão em status `PENDING` (sourceId ainda null)
- **THEN** o unique index parcial não dispara conflito
