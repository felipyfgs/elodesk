## ADDED Requirements

### Requirement: Cliente Socket.IO com auto-reconnect

O frontend SHALL fornecer composable `useRealtime()` baseado em WebSocket nativo (via `@vueuse/core#useWebSocket` ou cliente manual `new WebSocket(...)`) que:

- Conecta em `runtimeConfig.public.wsUrl` (ex: `ws://localhost:3001/realtime`) passando JWT via `Sec-WebSocket-Protocol: bearer,<token>` ou query `?token=<token>` como fallback em dev.
- Reemitir `join.account`, `join.inbox`, `join.conversation` ativos após reconexão automática.
- Expor `on(type, handler)` reativo sobre a estrutura `{type,payload}` trocada em JSON.
- Implementar reconexão com backoff (1s, 2s, 4s, max 30s).

#### Scenario: eventos entregues reativamente

- **WHEN** user abre `/conversations/:id` e componente chama `useRealtime().on('message.new', ...)`
- **THEN** componente recebe toda nova mensagem daquela conversation em tempo real

#### Scenario: re-join após reconexão

- **WHEN** socket reconecta depois de perda de rede
- **THEN** os rooms em que estava são re-entrados automaticamente

### Requirement: Stores Pinia

O frontend SHALL usar `@pinia/nuxt` com stores:

- `auth` — `user`, `account`, `accessToken`, `refreshToken`, métodos `login`/`register`/`logout`/`refresh`.
- `accounts` — lista de accounts em que o user tem membership + account atual.
- `inboxes` — inboxes da account atual com tipo `channelApi: {identifier, webhookUrl, hmacMandatory}` (substitui `channelWhatsapp`).
- `conversations` — conversations da account atual com paginação.
- `messages` — messages da conversation aberta com paginação reversa.

`accessToken` e `refreshToken` SHALL ser persistidos em `localStorage` via plugin.

#### Scenario: refresh de página preserva sessão

- **WHEN** user recarrega a página
- **THEN** tokens são recuperados de `localStorage` e o user continua logado

#### Scenario: tipo Inbox não referencia WhatsApp

- **WHEN** inspeção do tipo `Inbox` em `stores/inboxes.ts`
- **THEN** não existe mais campo `channelWhatsapp` nem referência a `wzapSessionId`/`wzapToken`
