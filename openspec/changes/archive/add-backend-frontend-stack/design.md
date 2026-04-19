## Context

O repositório `/home/obsidian/dev/project/` contém hoje dois projetos não integrados: `wzap/` (gateway Go/whatsmeow, 104 rotas REST + webhook HMAC + WS por sessão, totalmente funcional) e `dashboard/` (template Nuxt 4 Nuxt UI com endpoints mockados em `server/api/*`, sem auth, sem cliente HTTP real).

A proposta introduz uma terceira peça (`backend/` NestJS) cujo papel é exatamente o que o Chatwoot hoje cumpre via [wzap/internal/integrations/chatwoot/](../../../wzap/internal/integrations/chatwoot/): consumir o wzap via REST e webhook, manter estado próprio de conversas/contatos/mensagens, e servir uma experiência multi-tenant. A diferença é que controlamos as duas pontas (backend + frontend), então podemos desenhar o produto sem restrições de compatibilidade com o domínio Chatwoot.

**Restrições:**
- Wzap é tratado como dependência externa imutável. Todo contrato vem de [wzap/docs/swagger.yaml](../../../wzap/docs/swagger.yaml), [wzap/internal/model/events.go](../../../wzap/internal/model/events.go) e [wzap/internal/webhook/dispatcher.go](../../../wzap/internal/webhook/dispatcher.go).
- Isolamento multi-tenant é hard-requirement — `OrgScopeGuard` + índices compostos `(accountId, ...)` são obrigatórios.
- Idempotência é crítica: wzap pode reenviar webhooks em retry e eventos via WS e HTTP podem chegar duplicados. Chave: `Message.sourceId = "WAID:<wzapMessageID>"` unique.

## Goals / Non-Goals

**Goals:**
- Backend NestJS consumindo wzap 100% via contrato público (sem acoplar ao código Go).
- Schema Prisma multi-tenant com nomenclatura Chatwoot (facilita migração futura se decidirmos adotar Chatwoot como frontend).
- Pipeline inbound resiliente: webhook HTTP é fonte de verdade, WS é otimização; ambos convergem via idempotência.
- Tipos TS do wzap sempre sincronizados (via codegen do Swagger), zero drift silencioso.
- Frontend 100% desacoplado — nunca fala direto com wzap, só com o backend (HTTP + Socket.IO).
- Monorepo minimalista: pnpm workspace + Turbo só onde faz sentido (backend + frontend TS); wzap fica como repo separado.

**Non-Goals:**
- Migrar dados históricos de Chatwoot/Whaticket.
- Implementar features além do MVP (billing, verificação de email, convites, 2FA).
- Suportar canais além de WhatsApp no MVP (embora o schema deixe `Inbox.channelType` aberto).
- Reescrever o frontend do zero — aproveitar o template `dashboard/` e estendê-lo.
- Portar código 1:1 do Chatwoot/Whaticket.

## Decisions

### D1 — Ingest do wzap: webhook HTTP como fonte de verdade + WS como otimização

**Escolhido:** Webhook HTTP recebe `POST /wzap/webhook/:channelId` com `X-Wzap-Signature` HMAC-SHA256, enfileira em BullMQ, retorna 200 rápido; worker processa e grava. Em paralelo, `WzapWsClient` mantém uma conexão WS por `ChannelWhatsapp` ativa, gerando eventos via `EventEmitter2` que também caem no mesmo service handler. Idempotência por `Message.sourceId` unique garante convergência.

**Alternativas:** (a) Só webhook — perde latência em bursts. (b) Só WS — perde eventos se o WS cair (wzap não faz catch-up de eventos em WS). (c) NATS JetStream — requer expor NATS externamente, complica deploy.

**Porquê:** o wzap tenta 3× em 5xx e tem backoff exponencial (confirmado em [wzap/internal/webhook/dispatcher.go](../../../wzap/internal/webhook/dispatcher.go)), então webhook é garantido. WS é pura latência.

### D2 — Tipos wzap gerados do Swagger, não escritos à mão

**Escolhido:** `openapi-typescript ../wzap/docs/swagger.yaml -o src/wzap/wzap.schema.d.ts` executado via `pnpm gen:wzap`. `WzapHttpClient` é wrapper fino tipado. CI roda o script e falha se `git diff` não estiver limpo.

**Alternativas:** (a) DTOs à mão — drift inevitável. (b) Zod + inferência — mesma manutenção manual. (c) gRPC — wzap não expõe gRPC.

**Porquê:** Swagger é a fonte de verdade do wzap (regenerado via `make docs`). Codegen elimina classe inteira de bugs.

### D3 — Schema Prisma inspirado em Chatwoot, não em Whaticket

**Escolhido:** Modelagem `Account → AccountUser → Inbox → ChannelWhatsapp → Conversation → Message/Attachment`, com `ContactInbox` como tabela de junção carregando `sourceId` (padrão do Chatwoot).

**Alternativas:** (a) Modelagem Whaticket (Ticket/Contact/User) — mais simples mas menos escalável pra features futuras. (b) Modelagem própria do zero — risco de reinventar errado.

**Porquê:** Chatwoot tem modelo validado em produção por anos, aceita múltiplos canais e tem extensões (labels, automation, CSAT) que queremos na v3. `sourceId` da convenção Chatwoot (`"WAID:..."`) preserva compatibilidade se um dia decidirmos plugar Chatwoot no backend como alternativa de UI.

### D4 — MinIO próprio do backend, bucket separado do wzap

**Escolhido:** Backend tem bucket `wzap-media` próprio. Inbound: ao receber evento `Message` com mídia, baixar presigned URL do wzap → re-uploadar no bucket próprio → registrar `Attachment` com `fileKey` local. Outbound: frontend pega presigned URL do backend via `POST /uploads/signed-url` → uploda direto → backend passa URL pública ao wzap.

**Alternativas:** (a) Reusar o MinIO do wzap — acopla storage entre projetos, troca de engine perde mídia histórica. (b) Cache híbrido — complexidade sem ganho claro para MVP.

**Porquê:** isolamento de falhas e reversibilidade. Custo: 2× de storage para itens quentes.

### D5 — Auth JWT próprio, sem provedor externo

**Escolhido:** `@nestjs/jwt` + Passport com `LocalStrategy` (email+senha) e `JwtStrategy` (Bearer). Access token 15min, refresh token 30d rotacionado armazenado como hash na tabela `RefreshToken`. Sem email verification, sem 2FA, sem OAuth social no MVP.

**Alternativas:** (a) BetterAuth — over-engineering pro MVP. (b) Keycloak — infraestrutura extra.

**Porquê:** MVP. Fácil trocar depois porque `AuthService` é isolado.

### D6 — Realtime via Socket.IO com rooms hierárquicos

**Escolhido:** `@nestjs/platform-socket.io` + `RealtimeGateway`. Auth no handshake via `auth.token` (JWT). Rooms: `account:{id}` (broadcast geral da org), `inbox:{id}` (eventos de uma sessão WA), `conversation:{id}` (thread aberta). Guards garantem que o socket só entra em rooms de accounts que o user pertence.

**Alternativas:** (a) WS nativo (`ws`) — mais trabalho pra rooms/reconexão. (b) SSE — unidirecional só server→client; precisa HTTP pro ack de leitura.

**Porquê:** Socket.IO tem reconexão, rooms, auth, fallback de long-polling prontos. Ecossistema Nest oferece integração direta.

### D7 — Monorepo pnpm workspace incluindo só backend + frontend

**Escolhido:** `pnpm-workspace.yaml` raiz lista `backend` e `frontend`. Turbo coordena pipelines (`dev`, `build`, `lint`, `test`). `wzap/` fica fora (é repo Go) e `_refs/` fica fora (gitignored, só estudo).

**Alternativas:** (a) 3 repos separados — perde sharing de tipos entre backend e frontend. (b) Nx — mais features mas mais complexidade.

**Porquê:** backend e frontend compartilham tipos (ex: `MessageDto` pode vir de um pacote `shared/`), Turbo acelera CI local.

### D8 — Schema nomenclatura em inglês, UI em PT-BR

**Escolhido:** Tabelas/models/APIs em inglês (`Account`, `Message`, `conversations.controller.ts`). Strings de UI via `@nuxtjs/i18n` com PT-BR como default; EN como adicional futuro.

**Porquê:** facilita contribuição, lookup em docs e migração Chatwoot; alinhado com o wzap (já em EN).

## Fluxos principais

### F1 — Criar sessão WhatsApp (onboarding)

```
UI                  backend                      wzap
 │  POST /accounts/:id/inboxes {name}            │
 ├────────────────────►│                          │
 │                      │  POST /sessions         │
 │                      ├─────────────────────────►│
 │                      │◄─── {sessionId, token}───│
 │                      │  POST /sessions/:id/webhooks
 │                      │      {url, secret, events:["All"]}
 │                      ├─────────────────────────►│
 │                      │  POST /sessions/:id/connect
 │                      ├─────────────────────────►│
 │                      │  (persiste Inbox + ChannelWhatsapp)
 │◄── 201 {inbox, channel} ──│                     │
 │                                                │
 │  [Socket.IO room inbox:{id}]                   │
 │                      │◄── POST webhook (QR)─────│
 │◄── emit qr.update ───│                          │
 │  [scan QR]                                     │
 │                      │◄── POST webhook (PairSuccess)
 │◄── emit session.status=CONNECTED ──│           │
```

### F2 — Mensagem inbound (WA → frontend)

```
WhatsApp → wzap → POST /wzap/webhook/:channelId (HMAC) → HmacGuard
  → BullMQ queue "wzap-events" → WzapEventService.handle(event)
    → case "Message": upsert Contact → upsert ContactInbox → upsert Conversation
       → insert Message { sourceId: "WAID:"+msgId, direction: INCOMING, status: SENT }
       → se tem mídia: enqueue media-download job
       → realtime.emit(account, "message.new", dto)
    → ACK 200 (rápido, <50ms — processamento assíncrono)
```

### F3 — Mensagem outbound (frontend → WA)

```
UI → POST /api/v1/conversations/:id/messages {body}
  → MessagesService
    1. insert Message { direction: OUTGOING, status: PENDING } → emit message.new (otimista)
    2. wzap.sendText(sessionId, { phone, body })
    3. update Message { sourceId: "WAID:"+msgId, status: SENT } → emit message.updated
    4. [assincronamente] webhook Receipt atualiza status → DELIVERED/READ → emit message.updated
```

### F4 — Edição / deleção (bidirecional)

- **Inbound**: wzap event `MessageEdit` ou `MessageRevoke` → localizar Message por `sourceId` → update `content` ou `contentAttributes.deleted=true` → emit.
- **Outbound**: `PATCH /messages/:id` → `wzap.editMessage()`; `DELETE /messages/:id` → `wzap.deleteMessage()`. Backend atualiza registro local assim que wzap responde 200.

## Migração e deploy

1. **Fase 0** (esta change implementa): setup monorepo + scaffolding + stacks de dev. Sem impacto em produção (nada existe ainda).
2. **Deploy dev**: `docker-compose up` (postgres+redis+minio) + `pnpm dev` (backend em 3001 + frontend em 3000) + wzap rodando separadamente em 8080.
3. **Deploy prod** (fora de escopo desta change): Dockerfile por projeto, orquestrado via Compose ou Kubernetes. wzap fica no mesmo cluster/rede.
4. **Rollback**: como é feature nova e isolada, rollback é `git revert` da change e parar os containers novos. Wzap não é tocado.

## Open questions

- Nomenclatura: preferimos `phoneNumber` ou `phone` em `Contact`? Chatwoot usa `phone_number`.
- Tamanho máximo de payload no POST /messages: reutilizar o 512 KB do wzap ou subir?
- Precisa de uma rota `/health/wzap` que pinge o wzap via HTTP e expõe status agregado? (útil para o frontend mostrar "engine offline").
- Guardar `wzap.url`, `wzap.adminToken` em tabela `SystemConfig` (editável via UI de super-admin) ou só em env vars? MVP: env.

## Risks / Trade-offs

- **[Risk]** Wzap muda o shape do webhook sem bump → tipos locais ficam errados. **Mitigation:** codegen do Swagger rodando em CI; event payloads validados via Zod no `WzapEventService` antes de qualquer upsert.
- **[Risk]** HMAC secret vazado → atacante injeta mensagens falsas. **Mitigation:** secret por canal (não global), rotacionável via endpoint admin, armazenado criptografado.
- **[Risk]** BullMQ cresce descontrolado se wzap disparar flood. **Mitigation:** rate limit por `channelId` no worker; alarme se queue > 10k.
- **[Risk]** MinIO do backend lota com mídia de grupos grandes. **Mitigation:** TTL configurável por accountId (default 90d) com job de limpeza.
- **[Risk]** `Message.sourceId` nulável para mensagens OUTGOING em status PENDING — pode falhar o unique antes do wzap responder. **Mitigation:** unique é parcial `WHERE sourceId IS NOT NULL` (Postgres partial index).
- **[Risk]** Frontend expõe MinIO URL pública com presigned longo. **Mitigation:** presigned de 15min, renovação on-demand via endpoint do backend.
- **[Trade-off]** MinIO duplicado dobra custo de storage para mídia recente mas ganha independência do wzap — aceitável dado MVP.
- **[Trade-off]** Socket.IO tem overhead vs WS nativo (~10KB de payload de handshake) — aceitável pela DX.
