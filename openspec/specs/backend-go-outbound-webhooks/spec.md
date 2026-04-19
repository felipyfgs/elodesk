# backend-go-outbound-webhooks Specification

## Purpose
TBD - created by archiving change rewrite-backend-in-go. Update Purpose after archive.
## Requirements
### Requirement: Emissão de webhooks outbound assinados

O backend SHALL emitir webhooks HTTP POST no `channel_api.webhook_url` em toda mudança de estado relevante, com headers:

```
Content-Type: application/json
User-Agent: backend-go/<version>
X-Chatwoot-Hmac-Sha256: <hex(HMAC-SHA256(body, channel_api.hmac_token))>
X-Delivery-Id: <uuid>
```

Eventos obrigatórios:

- `message_created` — Message nova (outgoing ou incoming relevante)
- `message_updated` — edit ou soft delete (content_attributes.edited/deleted)
- `conversation_status_changed` — status muda
- `conversation_updated` — meta muda (assignee, labels, custom_attributes)

Payload shape = idêntico ao Chatwoot (ver `wzap/internal/integrations/chatwoot/webhook_outbound.go`).

#### Scenario: mensagem outgoing gera webhook

- **WHEN** agente envia `POST /api/v1/accounts/:aid/conversations/:cid/messages` com JWT
- **THEN** Message é criada com `message_type=outgoing, status=PENDING`
- **AND** `message_created` é enfileirado pro webhook do inbox e posteriormente POSTado assinado

#### Scenario: provider valida HMAC

- **WHEN** provider recebe webhook e computa `HMAC-SHA256(body, hmac_token)`
- **THEN** bate com header `X-Chatwoot-Hmac-Sha256`

### Requirement: Retry exponencial com backoff

A fila asynq `provider-webhooks` SHALL retentar jobs falhos com backoff `1s, 5s, 30s, 2m, 10m` (5 tentativas). Após esgotar, mover pro dead-letter e emitir log `level=error` com `delivery_id` + `inbox_id`.

#### Scenario: provider offline

- **WHEN** POST pro `webhook_url` retorna 5xx
- **THEN** asynq retenta com backoff exponencial até 5×

#### Scenario: sucesso no retry

- **WHEN** provider volta após falha transiente e aceita o POST
- **THEN** job marca sucesso e não é mais retentado

### Requirement: Idempotência de entregas

Cada dispatch SHALL ter `X-Delivery-Id` (UUID v4) estável entre retries. Provider MAY usar pra dedupe.

#### Scenario: retry usa mesmo delivery_id

- **WHEN** um job retenta 3 vezes
- **THEN** todas as requests têm o mesmo `X-Delivery-Id`

### Requirement: Dispatch não bloqueia request do agente

`message_service.Send` SHALL retornar ao caller (frontend) imediatamente após persistir com `status=PENDING` e enfileirar o job; não aguarda resposta do provider.

#### Scenario: response rápido no POST

- **WHEN** agente envia mensagem
- **THEN** HTTP response retorna em < 100ms mesmo se provider está lento/offline

