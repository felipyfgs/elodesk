## 1. Migração e modelo

- [x] 1.1 Criar `backend/migrations/0012_channels_web_widget.sql` (id, inbox_id FK, website_token unique, hmac_token_ciphertext, website_url, widget_color, welcome_title, welcome_tagline, reply_time enum, feature_flags jsonb, created_at, updated_at)
- [x] 1.2 Adicionar struct `ChannelWebWidget` em `backend/internal/model/models.go` (ciphertext com `json:"-"`)
- [x] 1.3 Criar `backend/internal/repo/channel_web_widget_repo.go` com CRUD + `FindByWebsiteToken(token)` + scopes por accountID
- [x] 1.4 Adicionar coluna `pubsub_token` na `conversations` (se ainda não existir) — 32 bytes random base64url, gerado na criação

## 2. Pacote `internal/channel/webwidget`

- [x] 2.1 Criar `backend/internal/channel/webwidget/types.go` com `WidgetConfig`, `VisitorClaims`, `InboundMessage`, `OutboundEvent`
- [x] 2.2 Criar `backend/internal/channel/webwidget/jwt.go` — `IssueVisitorJWT(contactID, websiteToken, conversationID)`, `ParseVisitorJWT(token)` usando `WIDGET_JWT_SECRET`
- [x] 2.3 Criar `backend/internal/channel/webwidget/hmac.go` — `ComputeIdentifierHash(hmacToken, identifier)` + `VerifyIdentifierHash(hmacToken, identifier, providedHash)` com `crypto/subtle.ConstantTimeCompare`
- [x] 2.4 Criar `backend/internal/channel/webwidget/session.go` — `CreateOrResumeSession(ctx, websiteToken, cookie, ip)` retornando contact/conversation/JWT/pubsubToken
- [x] 2.5 Criar `backend/internal/channel/webwidget/identify.go` — upgrade de contact anônimo + merge com existente
- [x] 2.6 Criar `backend/internal/channel/webwidget/sse.go` — handler SSE com `http.Flusher`, subscribe Redis, keepalive ticker
- [x] 2.7 Criar `backend/internal/channel/webwidget/pubsub.go` — `Publish(pubsubToken, event)` via Redis PUBLISH
- [x] 2.8 Criar `backend/internal/channel/webwidget/webwidget.go` implementando `channel.Channel`: `SendOutbound` publica em Redis (e persiste message antes)

## 3. Rate limit

- [x] 3.1 Criar `backend/internal/middleware/widget_ratelimit.go` — token bucket Redis; funções `LimitByIP(key, max, window)` e `LimitByToken(key, max, window)`
- [x] 3.2 Aplicar middleware nas rotas públicas com limites documentados no spec

## 4. Handler público

- [x] 4.1 Criar `backend/internal/handler/widget_public_handler.go` com:
  - `GET /widget/:websiteToken` — embed script estático
  - `POST /api/v1/widget/sessions`
  - `POST /api/v1/widget/messages`
  - `POST /api/v1/widget/identify`
  - `POST /api/v1/widget/attachments`
  - `GET /api/v1/widget/messages?after=&limit=` — polling fallback
  - `GET /widget/:websiteToken/ws` — SSE
- [x] 4.2 Middleware CORS aberto (`*`) pros paths `/api/v1/widget/*` e `/widget/*`
- [x] 4.3 Middleware auth JWT visitor pros endpoints que precisam (messages, identify, attachments, polling)
- [x] 4.4 Set-cookie no `POST /widget/sessions` — `HttpOnly; SameSite=None; Secure` (prod) ou `SameSite=Lax` (dev)

## 5. Provisioning

- [x] 5.1 Criar `backend/internal/handler/web_widget_inbox_handler.go` com `POST /api/v1/accounts/:aid/inboxes/web_widget`
- [x] 5.2 Gerar `website_token` (32 bytes base64url), gerar `hmac_token` (32 bytes base64url), encrypt via KEK
- [x] 5.3 Resposta inclui `embedScript` pronto (snippet `<script>` com `data-website-token` + URL do bundle)
- [x] 5.4 `POST /api/v1/accounts/:aid/inboxes/:id/rotate_hmac` — gera novo hmac, retorna uma vez
- [x] 5.5 DTOs em `backend/internal/dto/web_widget.go`

## 6. Wiring

- [x] 6.1 Registrar `webwidget.Channel` no `channel.Registry` em `backend/internal/server/router.go`
- [x] 6.2 Registrar rotas públicas com middlewares corretos (CORS + rate limit + visitor JWT onde aplicável)
- [x] 6.3 Registrar rotas admin de provisioning
- [x] 6.4 Config: adicionar `WIDGET_PUBLIC_BASE_URL`, `WIDGET_JWT_SECRET`, `WIDGET_SESSION_TTL_DAYS` em `backend/internal/config/config.go` + `.env.example`

## 7. Bundle do widget (subprojeto frontend)

- [x] 7.1 Criar `widget/` (subprojeto standalone) com Vite library mode + Vue 3 (sem Nuxt)
- [x] 7.2 UI mínima: botão flutuante, janela expansível, lista de mensagens, campo de input, botão emoji, botão anexo
- [x] 7.3 Cliente HTTP: `createSession`, `sendMessage`, `identify`, `uploadAttachment`, `pollMessages`, `subscribeSSE`
- [x] 7.4 Fallback polling quando EventSource falha 3x
- [x] 7.5 i18n embarcado: `pt-BR`, `en`, `es` com labels default; auto-detect via `navigator.language`
- [x] 7.6 Build target `widget.js` + `widget.css` — size budget <60KB gzip, falha build se exceder
- [x] 7.7 CI job publica artifact pra CDN (S3/Cloudflare R2) no deploy

## 8. Testes

- [x] 8.1 `backend/internal/channel/webwidget/jwt_test.go` — issue, parse, exp expirada, assinatura errada
- [x] 8.2 `backend/internal/channel/webwidget/hmac_test.go` — compute/verify, timing attack resistente
- [x] 8.3 `backend/internal/channel/webwidget/session_test.go` — primeira sessão (anônimo), resume via cookie, JWT expirado
- [x] 8.4 `backend/internal/channel/webwidget/identify_test.go` — HMAC válido (cria/update/merge), inválido (401), sem hash (meta.verified=false)
- [x] 8.5 `backend/internal/channel/webwidget/sse_test.go` — keepalive, publish/receive, desconexão do cliente, pubsubToken errado
- [x] 8.6 `backend/internal/middleware/widget_ratelimit_test.go` — bucket por IP, por token, reset após janela
- [ ] 8.7 Integration test — full flow: criar canal → chamar embed → create session → send message → agente responde → widget recebe via SSE

## 9. Documentação

- [x] 9.1 `backend/README.md` seção Web Widget — como provisionar, embed script, identify com HMAC (exemplo em Node/Ruby/PHP), ENV vars, CDN setup
- [x] 9.2 `widget/README.md` — como buildar o bundle, onde publicar, tamanho máximo
- [x] 9.3 `README.md` raiz lista Web Widget
- [x] 9.4 Swagger annotations nos handlers públicos + admin

## 10. Validação ponta-a-ponta

- [ ] 10.1 Provisionar canal via API admin; copiar `embedScript`
- [ ] 10.2 Embutir em página HTML de teste local; carregar; widget aparece
- [ ] 10.3 Clicar no widget, digitar mensagem; conferir que conversa aparece na inbox do agente no frontend
- [ ] 10.4 Agente responde; conferir mensagem aparece no widget em tempo real (SSE)
- [ ] 10.5 Upload de anexo (imagem) pelo widget; conferir que aparece no frontend do agente
- [ ] 10.6 Simular identify: página host computa HMAC e chama `identify`; contact anônimo vira identificado; checar merge se já havia contact com mesmo identifier
- [ ] 10.7 Fechar aba, reabrir; conferir que sessão é retomada via cookie (mesma conversation_id)
- [ ] 10.8 Rate limit: 70 mensagens em 1 minuto do mesmo IP; conferir 429 com Retry-After
- [ ] 10.9 Forçar fallback polling (desabilitar SSE via devtools); conferir polling entrega mensagens novas
- [ ] 10.10 Rotacionar hmac_token via admin; conferir que hash antigo não funciona mais
