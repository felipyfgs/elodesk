## ADDED Requirements

### Requirement: Cliente HTTP com interceptor JWT

O frontend SHALL fornecer composable `useApi()` que retorna uma instância `$fetch.create(...)` com:
- `baseURL` vindo de `runtimeConfig.public.apiUrl`.
- `onRequest` injetando `Authorization: Bearer <accessToken>` quando presente na store.
- `onResponseError` chamando o endpoint de refresh quando recebe 401 e reexecutando a request original; se o refresh falhar, redireciona para `/login`.

#### Scenario: request autenticada

- **WHEN** user está logado e chama `useApi().get('/accounts/me')`
- **THEN** a request sai com header `Authorization: Bearer ...`

#### Scenario: access token expirado

- **WHEN** response é 401
- **THEN** `useApi` chama `/auth/refresh` e reexecuta a request original transparentemente

#### Scenario: refresh falha

- **WHEN** refresh retorna 401
- **THEN** tokens são limpos da store e o user é redirecionado para `/login`

### Requirement: Cliente Socket.IO com auto-reconnect

O frontend SHALL fornecer composable `useRealtime()` que:
- Conecta em `runtimeConfig.public.wsUrl` com `auth.token = accessToken`.
- Reemitir `join.account`, `join.inbox`, `join.conversation` ativos após reconexão.
- Expor `on(event, handler)` reativo.

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
- `inboxes` — inboxes da account atual + status.
- `conversations` — conversations da account atual com paginação.
- `messages` — messages da conversation aberta com paginação reversa.

`accessToken` e `refreshToken` SHALL ser persistidos em `localStorage` via plugin.

#### Scenario: refresh de página preserva sessão

- **WHEN** user recarrega a página
- **THEN** tokens são recuperados de `localStorage` e o user continua logado

#### Scenario: logout limpa tudo

- **WHEN** `auth.logout()` é chamado
- **THEN** todos os stores resetam e `localStorage` é limpo

### Requirement: Páginas de autenticação e rotas protegidas

O frontend SHALL ter:
- `/login` com form `{email, password}`.
- `/register` com form `{email, password, name, accountName?}`.
- `middleware/auth.global.ts` que redireciona para `/login` quando `auth.user` é null e a rota não é pública.

#### Scenario: acesso não autenticado

- **WHEN** user acessa `/conversations` sem estar logado
- **THEN** é redirecionado para `/login?redirect=/conversations`

#### Scenario: login redireciona de volta

- **WHEN** user faz login vindo de `/login?redirect=/conversations`
- **THEN** é redirecionado para `/conversations`

### Requirement: Páginas de produto (sessions + conversations)

O frontend SHALL ter:
- `/sessions` — lista de inboxes com status, botão "nova sessão", modal com QR ao vivo.
- `/conversations` — lista de conversations com filtros básicos (status, assignee).
- `/conversations/:id` — thread de mensagens com compositor de texto + attachment.

#### Scenario: fluxo de conectar nova sessão

- **WHEN** user cria nova inbox em `/sessions`
- **THEN** aparece modal com QR ao vivo atualizado via `qr.update` Socket.IO
- **AND** quando o status muda para `CONNECTED`, o modal fecha e a inbox aparece como online na lista

#### Scenario: enviar mensagem na thread

- **WHEN** user digita e envia mensagem em `/conversations/:id`
- **THEN** a mensagem aparece imediatamente com status `PENDING`, evoluindo para `SENT` (com check cinza) e `DELIVERED`/`READ` quando webhook chega

### Requirement: Remover mocks antigos

Os endpoints mockados em `frontend/server/api/customers.ts`, `mails.ts`, `members.ts`, `notifications.ts` SHALL ser removidos. O frontend NÃO deve ter nenhum endpoint Nitro próprio exceto utilitários (ex: healthcheck do frontend).

#### Scenario: mocks removidos

- **WHEN** inspeção de `frontend/server/api/`
- **THEN** os arquivos mockados não existem mais
- **AND** nenhum componente referencia `/api/customers`, `/api/mails`, `/api/members` ou `/api/notifications`

### Requirement: i18n com PT-BR padrão

O frontend SHALL incluir `@nuxtjs/i18n` configurado com:
- Locale padrão `pt-BR`.
- Locale adicional `en` (preenchimento futuro — pode começar como espelho do `pt-BR` até tradução).
- Detecção automática baseada no `navigator.language` com fallback para `pt-BR`.

#### Scenario: textos em português por padrão

- **WHEN** user acessa a aplicação sem override de locale
- **THEN** todos os textos da UI são em `pt-BR`

#### Scenario: textos dinâmicos usam o sistema de i18n

- **WHEN** componente novo é criado
- **THEN** strings visíveis ao user passam por `$t('chave')` e existem em `locales/pt-BR.json`
