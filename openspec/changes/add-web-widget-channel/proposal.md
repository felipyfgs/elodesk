## Why

Widget de chat no site é o canal inbound mais barato de operar e o único totalmente controlado pelo elodesk (sem API de terceiro, sem taxa de sessão). No Chatwoot é o canal mais usado: um snippet JS embarcado em qualquer site abre conversação com um contato anônimo que vira identificado depois. Nada disso existe hoje no elodesk, e é a peça que fecha a "suíte de canais" — sem widget, o cliente tem que pagar provider externo pra algo tão simples quanto chat no seu próprio site.

Esta change adiciona `Channel::WebWidget`: backend persiste canal + config de aparência, embed script stub serve bundle do widget, e um endpoint público autenticado por JWT de sessão processa mensagens do visitante. Conversação aparece na inbox do agente como qualquer outro canal.

## What Changes

- Nova tabela `channels_web_widget` com `website_token` (opaco, identifica o widget publicamente), `hmac_token_ciphertext` (AES-GCM, assina contact identity quando cliente opta por identify), `website_url`, `widget_color`, `welcome_title`, `welcome_tagline`, `reply_time` (`in_a_few_minutes|in_a_few_hours|in_a_day`), `feature_flags` JSONB (`attachments`, `emoji_picker`, `end_conversation`).
- Novo subpacote `internal/channel/webwidget/` implementando `channel.Channel`.
- Endpoint público embed: `GET /widget/:websiteToken` serve JS bundle (proxy estático; bundle frontend em build separado).
- API pública do widget (sem JWT de agente, JWT próprio de sessão de visitante):
  - `POST /api/v1/widget/sessions` — cria/retoma sessão, retorna `contact_identifier`, `conversation_id`, `pubsub_token` (usado pra realtime via SSE).
  - `POST /api/v1/widget/messages` — envia mensagem do visitante → vira `message` em conversa aberta.
  - `POST /api/v1/widget/identify` — com HMAC, promove contact anônimo para identificado (email/name).
  - `POST /api/v1/widget/attachments` — upload via MinIO presigned.
  - `GET /api/v1/widget/messages?after=<id>` — polling fallback pra navegadores sem SSE.
- Realtime visitante: `GET /widget/:websiteToken/ws` (SSE, não WebSocket — mais simples, atravessa proxies) entrega mensagens outbound do agente.
- Outbound (agente → visitante): publica no canal realtime do visitante; mensagem cai no widget via SSE.
- Browser fingerprint + cookie `elodesk_widget_session_<websiteToken>` mantém sessão mesmo sem identify.
- i18n do widget carregado via config (`pt-BR`, `en`, `es`) — tradução dos labels default.

## Capabilities

### New Capabilities
- `backend-go-channel-web-widget`: canal web widget com embed script público, sessão de visitante anônimo JWT, endpoint público autenticado, SSE realtime visitante, HMAC identity verification, outbound do agente chegando ao widget.

### Modified Capabilities
- Nenhuma.

## Impact

- **Código novo**: `backend/migrations/0012_channels_web_widget.sql`, `backend/internal/channel/webwidget/{webwidget,session,message,identify,sse}.go`, `backend/internal/repo/channel_web_widget_repo.go`, `backend/internal/handler/widget_public_handler.go`, DTOs.
- **Frontend novo**: `widget/` (novo sub-projeto Nuxt 3 standalone ou Vue 3 + Vite) que compila pra bundle único `widget.js` servido pelo backend.
- **Dependências Go**: nenhuma nova (SSE é `http.Flusher` nativo; JWT já existe).
- **ENV**: `WIDGET_PUBLIC_BASE_URL` (URL pública absoluta pro embed script), `WIDGET_SESSION_TTL_DAYS` (default 30).
- **Rollback**: `DROP TABLE channels_web_widget` + remover rotas públicas + `git revert`; scripts já embarcados em sites retornam 404 (degrada silenciosamente).

## Riscos e mitigações

- **CORS público amplo** → rotas `/api/v1/widget/*` precisam `Access-Control-Allow-Origin: *` (ou dynamic por `website_url`). Preferir validação por `website_token` no path e origem checada só em log. Rate limit agressivo por IP (Redis bucket).
- **Spam / abuso** → rate limit por `website_token` + IP: max 60 msgs/min/IP; 1000 sessões/hora/website_token. Captcha hook opcional (follow-up).
- **HMAC identify implementado errado** → seguir spec Chatwoot: `HMAC-SHA256(hmac_token, identifier)` hex, chegada via `identifier_hash` param. Teste com fixture real.
- **SSE atrás de proxies** → `X-Accel-Buffering: no` + `Cache-Control: no-cache` + keepalive a cada 30s. Fallback polling (`GET /widget/messages?after=`) quando cliente perde SSE.
- **Cookie de sessão leakage cross-origin** → cookie HttpOnly + SameSite=None + Secure (produção); TTL 30 dias default.
- **Bundle do widget gigante** → target <50KB gzip. Framework-less (Vue 3 standalone + Vite library mode); sem Nuxt no widget bundle.
- **Websocket vs SSE** → SSE escolhido porque widget só precisa server→client; cliente manda via `POST /widget/messages`. WebSocket no futuro se precisar typing indicators.
