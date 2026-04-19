## Context

Web widget é o único canal onde o cliente do elodesk embute código nosso em produção dele. Isso muda o shape de segurança: em vez de um agente autenticado com JWT, o visitante é anônimo e o endpoint público. Chatwoot resolve isso com:

- `website_token` opaco por canal (exposto no snippet) — identifica qual widget.
- `pub_sub_token` por conversa (exposto só após `create session`) — autoriza SSE.
- `hmac_token` por canal (secreto) — usado pelo site host pra assinar identity dos usuários logados dele, evitando impersonation.
- JWT de sessão de visitante com TTL longo persistido em cookie.

Queremos paridade funcional: embed script serve um bundle único que abre UI; visitante digita/anexa; agente vê na inbox; agente responde; visitante recebe via SSE. Reuso total de `channel.Channel`, `contact`, `conversation`, `message`, MinIO.

Restrições:

- **Public CORS**: rotas do widget precisam ser CORS-open, mas identificadas só por `website_token`.
- **Anti-abuse**: rate limit agressivo por IP + website_token (Redis).
- **Bundle size**: widget JS precisa ser leve (<50KB gzip) — hoste fora do Nuxt frontend.
- **SSE em proxies**: headers certos + keepalive; fallback polling.

## Goals / Non-Goals

**Goals:**

- `Channel::WebWidget` provisionável com config de aparência e textos.
- Embed script `GET /widget/:websiteToken` servindo bundle JS estático.
- API pública do widget autenticada por JWT de visitante (emitido via `POST /widget/sessions`).
- SSE realtime `GET /widget/:websiteToken/ws` entregando outbound do agente.
- Contact anônimo com upgrade opcional via `POST /widget/identify` (HMAC-verified).
- Fallback polling `GET /widget/messages?after=` quando SSE indisponível.
- Attachments via MinIO presigned (reuso do `media/upload.go`).
- Cookie persistente (`SameSite=None; Secure; HttpOnly`) retoma sessão após reload.

**Non-Goals:**

- Editor rico no widget (só texto, emoji, anexo).
- Typing indicators / read receipts (fica em follow-up com WebSocket).
- Customização profunda de CSS (só `widget_color` + textos; CSS fixo).
- Bot automation / canned replies disparados do widget (fluxo via integração externa).
- Proactive messages (agente puxar conversa sem visitante iniciar) — follow-up.
- Offline form / fallback pra email quando nenhum agente online — follow-up.
- Multi-widget por account com múltiplos sites — MVP suporta; mas controle fino de rules (ex: "só mostrar em /pricing") fica fora.

## Decisions

### D1 — Dois tipos de token: `website_token` público + `hmac_token` secreto

**Escolhido:** Tabela `channels_web_widget`:

- `website_token` — 32 bytes random base64url, exposto no embed script (snippet JS inclui como query param ou data-attr).
- `hmac_token_ciphertext` — AES-GCM via KEK, usado pelo backend do cliente pra assinar identity do usuário autenticado dele antes de mandar pro widget via `identifier_hash`.

**Porquê:** Espelha Chatwoot exatamente. `website_token` público permite embed sem auth; `hmac_token` secreto fica server-side do cliente. Sem HMAC, o widget só conseguiria contact anônimo; com HMAC, cliente garante identidade verificada (não dá pra impersonar).

### D2 — JWT de sessão de visitante, não JWT de agente

**Escolhido:** Claims: `{sub: contact.identifier, website_token, conversation_id?, iat, exp, typ: "visitor"}`. TTL 30 dias (env `WIDGET_SESSION_TTL_DAYS`). Assinado com chave separada `WIDGET_JWT_SECRET` (não a mesma do agent JWT). Persistido em cookie `elodesk_widget_session_<website_token>` (`HttpOnly`, `SameSite=None`, `Secure`).

**Porquê:** Keys separadas garantem que um JWT de visitante vazado não consegue nada no admin. TTL longo porque visitante não volta pra logar; é "identity leve". Cookie reduz atrito de JS storage/race conditions.

### D3 — SSE para server→cliente, REST para cliente→server

**Escolhido:** `GET /widget/:websiteToken/ws` (ENDPOINT SSE, nome legado mantido) abre `text/event-stream` com `X-Accel-Buffering: no`, keepalive a cada 30s (`: heartbeat\n\n`). O backend publica mensagens do agente via pub/sub interno (Redis ou channel Go) e o handler escreve no stream. Cliente manda mensagens via `POST /widget/messages`.

**Porquê:** SSE é mais simples que WS (um endpoint HTTP, autoreconnect nativo no EventSource). O widget não precisa de full-duplex — envio é via REST, só o recebimento precisa ser push. Menos código de cliente, atravessa HTTP/1.1 sem upgrade, infra amigável. WS fica pra quando precisar typing indicators.

### D4 — Fallback polling para navegadores sem EventSource

**Escolhido:** `GET /widget/messages?after=<last_message_id>&limit=20` retorna lista. Cliente faz polling a cada 5s quando EventSource falha 3x. Persiste escolha em localStorage.

**Porquê:** IE11 e alguns corporate proxies bloqueiam SSE. Polling 5s é aceitável pra um canal de chat (vs 60s de email poller). Cliente detecta e faz downgrade graceful.

### D5 — Embed script servido estaticamente pelo backend

**Escolhido:** `GET /widget/:websiteToken` retorna um HTML minimal que carrega `widget.js` + `widget.css` com `defer`. O bundle é built separado (`widget/` subprojeto) com Vite library mode. No deploy, `widget.js` vai pro bucket público CDN (Cloudflare/S3 com CDN); o backend só serve a página stub que referencia a URL via env `WIDGET_PUBLIC_BASE_URL`.

**Porquê:** Bundle em CDN = latência baixa global. Backend só serve script de bootstrap (poucos KB). Separa CI/CD do widget sem precisar redeploy do backend pra mudanças de UI. Chatwoot hospeda o bundle via fastly/cloudfront.

### D6 — Identify via HMAC de lado cliente

**Escolhido:** Cliente envia `POST /widget/identify` com `{identifier, email, name, identifier_hash}`. Backend computa `hmac_sha256(hmac_token, identifier)` hex e compara com `identifier_hash`. Match → upgrade do contact (persiste email/name) ou merge se já existe contact com esse identifier em outro canal. Mismatch → `401`.

**Porquê:** Sem HMAC, qualquer JS no browser poderia chamar identify e impersonar `user@empresa.com`. Com HMAC, só o backend do cliente (que tem o secret) pode gerar. Mesma pattern Chatwoot — muito battle-tested.

### D7 — Contact anônimo com cookie/fingerprint

**Escolhido:** Primeira visita: `POST /widget/sessions` sem JWT → backend gera `contact(identifier=anon_<ulid>, phone=null, email=null, meta={browser, os, city_from_ip})`, emite JWT de visitor, set-cookie. Visitas subsequentes: cookie traz JWT → backend resolve o mesmo contact.

**Porquê:** Sem identify, cookie é a única cola. ULID no identifier evita colisão e é opaco. Após identify, o identifier bate com o do cliente host (`identifier=user@empresa.com` por exemplo); merge lógico.

### D8 — Rate limit em Redis (token bucket)

**Escolhido:** Por request público, Redis `INCR` com TTL:

- Por IP: 60 mensagens/min, 10 sessões/min.
- Por website_token: 1000 sessões/hora.

Exceder → `429 Too Many Requests` com `Retry-After`.

**Porquê:** Widget público é vetor claro de spam. Token bucket em Redis é O(1), baixo overhead. Limites generosos pra UX legítima não sofrer.

### D9 — Pub/sub interno via Redis

**Escolhido:** Ao persistir mensagem outbound do agente, `PUBLISH widget:pubsub:<pubsub_token> <payload>`. Handler SSE faz `SUBSCRIBE` no startup. Mensagem recebida → write no stream.

**Porquê:** Múltiplas réplicas do backend = SSE de um visitante pode estar em réplica diferente da do agente. Redis pub/sub resolve fanout sem lock. Alternativa com channel Go só funciona single-node.

## Risks / Trade-offs

- **[Risco]** `website_token` vazado (público por natureza) → atacante pode abrir sessões. Mitigação: rate limit por IP + token. Rotação manual do token documentada (muda embed script do cliente).
- **[Risco]** HMAC token vazado no backend do cliente → atacante pode impersonar identities. Mitigação: rotação fácil do `hmac_token` via endpoint admin; documentar como segredo tier-1.
- **[Risco]** SSE connection leak (cliente fecha sem desconectar) → handler detecta via `r.Context().Done()` do Go; keepalive 30s força descoberta de conexão morta.
- **[Risco]** Bundle de widget cresce e carrega lento em sites lentos → métrica de tamanho em CI (falha build > 60KB gzip), lazy load de emoji picker.
- **[Trade-off]** SSE não tem typing indicators → aceito no MVP; WS futura.
- **[Trade-off]** Polling fallback tem latência 5s → melhor que nada, melhor que quebrar em IE11/corp proxy.
- **[Trade-off]** Cookie `SameSite=None` exige `Secure` (HTTPS). Dev local usa `SameSite=Lax` via flag.
- **[Trade-off]** Host CDN separado = mais peças pra manter. Mas ganho de latência e desacoplamento compensa.

## Migration Plan

1. Deploy migration `0012_channels_web_widget.sql`.
2. Build e deploy do bundle widget (subprojeto `widget/`) pra CDN/S3 público.
3. Configurar DNS `widget.elodesk.io` apontando pro CDN.
4. Deploy backend com rotas novas + env `WIDGET_PUBLIC_BASE_URL=https://widget.elodesk.io`, `WIDGET_JWT_SECRET=<rand 32 bytes>`.
5. Admin provisiona canal via `POST /api/v1/accounts/:aid/inboxes/web_widget` com config. Backend retorna snippet JS pra copiar/colar.
6. Teste: embutir snippet em página HTML de teste; abrir no browser; iniciar conversa; ver na inbox; agente responde; mensagem aparece no widget.
7. Teste identify: site de teste com endpoint que assina identity via HMAC e passa pro widget; upgrade do contact.
8. **Rollback**: rotas públicas retornam `410 Gone` (script embed detecta e se oculta silenciosamente); depois `DROP TABLE channels_web_widget` + `git revert`.

## Open Questions

- **Multi-idioma auto-detect**: widget autodetecta idioma do `navigator.language` ou pega do canal config? Proponho hybrid — config default mas honra `Accept-Language` se match em lista suportada.
- **Attachment max size pelo widget**: 256MB herdado do upload handler é grande demais pra widget casual. Proponho cap em 10MB via DTO especifico, com override admin.
- **Mobile SDK (iOS/Android)**: widget hoje só web; cliente mobile precisaria SDKs nativos. Proponho fora do MVP — cliente usa webview no app enquanto isso.
- **Offline/closed state**: quando nenhum agente online, mostrar form pra coletar email e virar email ticket? Fica em follow-up quando tivermos "agent presence" rastreado.
- **CSP/iframe embed**: alguns clientes querem embed via iframe (sandbox). Proponho suportar ambos — script inline (default) e iframe via `GET /widget/:websiteToken/iframe`; pouco código extra.
