# frontend

Nuxt 4 + Nuxt UI + Pinia + socket.io-client. Cliente exclusivo do [../backend/](../backend/).

## Variáveis de ambiente

Veja [.env.example](.env.example).

| Var                   | Descrição                                            |
|-----------------------|------------------------------------------------------|
| `NUXT_PUBLIC_API_URL` | Base URL do backend (ex: `http://localhost:3001/api/v1`) |
| `NUXT_PUBLIC_WS_URL`  | Base URL Socket.IO do backend (ex: `http://localhost:3001`) |

## Scripts

```bash
pnpm install       # da raiz do monorepo resolve deps de backend + frontend
pnpm dev           # nuxt dev, em 3000
pnpm build
pnpm lint
pnpm typecheck
```

## Rotas

| Rota                         | Descrição                                              |
|------------------------------|--------------------------------------------------------|
| `/login`                     | Login com email+senha                                  |
| `/register`                  | Registro (cria user + account OWNER)                   |
| `/sessions`                  | Lista inboxes WhatsApp + botão nova sessão + QR modal  |
| `/conversations`             | Lista de conversations com filtros                     |
| `/conversations/:id`         | Thread + compositor                                    |
| `/settings`                  | Dados do user e da organização                         |

Todas as rotas exceto `/login` e `/register` exigem sessão (middleware `auth.global.ts`).

## Stores (Pinia)

- `auth` — user, account, tokens (persistido em `localStorage`).
- `accounts` — memberships do user.
- `inboxes` — sessões WhatsApp da account atual.
- `conversations` — conversations paginadas.
- `messages` — mensagens por conversation com paginação reversa.

## Composables

- `useApi()` — `$fetch.create` com `Authorization: Bearer`, refresh automático em 401.
- `useAuth()` — `login`/`register`/`logout`.
- `useRealtime()` — Socket.IO com `join.*` persistido e re-entrada em reconexão.

## i18n

`@nuxtjs/i18n` com `pt-BR` default e `en` secundário. Todas strings visíveis passam por `t('chave')`. Detecção via cookie + fallback `pt-BR`.
