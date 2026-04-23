# conversations-ui Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: /conversations segue o padrão UDashboardPanel do template

O frontend SHALL reescrever `frontend/app/pages/conversations/index.vue` seguindo o padrão `pages/inbox.vue` do `_refs/dashboard` — múltiplos `UDashboardPanel` com `id` próprio, `UDashboardNavbar` em cada um, composição via slots `#header`/`#body`. O layout de três painéis MUST usar `UDashboardPanel` com props `:default-size`/`:min-size`/`:max-size` + `resizable` para permitir o arrasto — nada de CSS grid custom.

Decomposição obrigatória em `frontend/app/components/conversations/`:

| Componente | Responsabilidade |
|---|---|
| `ConversationsSidebar.vue` | Árvore de filtros (Mine/Unassigned/All/Mentions, inboxes, labels, teams, saved filters) |
| `ConversationsList.vue` | Lista virtualizada (reaproveita o padrão de `inbox/InboxList.vue`): `defineModel<Conversation|null>()`, `defineShortcuts({ arrowup, arrowdown })` |
| `ConversationThread.vue` | Thread ativa (reaproveita o padrão de `inbox/InboxMail.vue`): navbar com assunto + dropdown de ações |
| `ConversationsToolbar.vue` | Toolbar contextual de bulk actions (aparece via `v-if="selection.length"`) |
| `ConversationComposer.vue` | Composer com `UTextarea` (autoresize), triggers `/`, `@`, attachments |

#### Scenario: layout de três painéis redimensionáveis

- **WHEN** usuário abre `/conversations`
- **THEN** três `UDashboardPanel` (ids `conversations-sidebar`, `conversations-list`, `conversations-thread`) ficam lado a lado; arrastar o divisor ajusta a largura e persiste em `localStorage` via `useStorage` do `@vueuse/core`

### Requirement: Abas Mine / Unassigned / All / Mentions via UTabs

O `ConversationsList.vue` SHALL expor `UTabs` no `UDashboardNavbar` (slot `#right`), espelhando `pages/inbox.vue:L tabItems`. Abas:

- `mine` — `assignee_id = currentUserId`
- `unassigned` — `assignee_id IS NULL`
- `all` — todas
- `mentions` — conversas onde o usuário foi mencionado

A aba ativa MUST estar sincronizada com query param `?tab=` (padrão: `mine`). Filtros adicionais — inbox, label, team, status, date range — são combináveis e acumulam como query params (URL compartilhável).

Esquemas Zod em `frontend/app/schemas/conversations.ts`:

- `conversationFiltersSchema` (tab, inbox, label, team, status, from, to)
- `conversationBulkActionSchema` (ids[], action, payload)

#### Scenario: filtro compartilhável por URL

- **WHEN** usuário aplica filtros e copia a URL
- **THEN** outro agente abrindo a URL vê exatamente a mesma lista filtrada

### Requirement: Bulk actions com UDashboardToolbar contextual

Quando `selection.length > 0`, `ConversationsList.vue` SHALL renderizar `ConversationsToolbar.vue` no `UDashboardToolbar` do panel da lista, com ações: Resolve, Snooze (dropdown de duração 1h/4h/Amanhã/Próx. Semana), Assign agent, Assign team, Add/Remove label, Mark unread, Delete. Checkbox de seleção segue o padrão de `pages/customers.vue` (checkbox de header + cell, `rowSelection` como `ref`).

#### Scenario: resolver em bulk

- **WHEN** usuário seleciona 5 conversas e clica "Resolve"
- **THEN** 5 chamadas `PATCH /conversations/:id` com `{ status: 'resolved' }` são disparadas em paralelo; toast "5 conversas resolvidas" aparece e seleção limpa

### Requirement: Composer com triggers `/` e `@`

`ConversationComposer.vue` SHALL usar `UTextarea` (variant `none`, autoresize) — mesmo padrão do `InboxMail.vue` do template — e adicionar:

- Trigger `/` abre `CannedResponsePicker.vue` (já existe em `frontend/app/components/`), movido para `components/conversations/` se compartilhado só aqui
- Trigger `@` abre `MentionPicker.vue` (novo em `components/conversations/`) com agents da inbox atual
- Upload de attachments via presigned MinIO (botão clipe), exibe thumbnails em `ComposerAttachments.vue`
- Indicador "digitando..." publicado via `useRealtime().emit('conversation.typing', { conversationId })` com throttle 3 s
- Contador de caracteres quando o canal tem limite (WhatsApp 4096, SMS 160) — lê do `inbox.channel_type`

#### Scenario: anexar imagem

- **WHEN** usuário arrasta imagem sobre o composer
- **THEN** cliente pede presigned URL (`POST /uploads/presign`), faz PUT direto ao MinIO, e envia mensagem com `attachments: [{url, type, size}]`

### Requirement: Rotas scoped reaproveitam o mesmo index

O frontend SHALL expor rotas scoped que carregam `conversations/index.vue` com filtros pré-aplicados via `definePageMeta({ middleware: 'conversations-scope' })`:

- `/conversations/inbox/[id].vue`
- `/conversations/label/[name].vue`
- `/conversations/team/[id].vue`
- `/conversations/filter/[id].vue` (saved filter)
- `/conversations/mentions.vue`
- `/conversations/unattended.vue`

Cada arquivo MUST ser um re-export de `index.vue` com `definePageMeta` injetando o filtro correspondente — evita duplicação de lógica.

#### Scenario: scoped por inbox

- **WHEN** usuário clica em uma inbox na sidebar
- **THEN** URL vira `/conversations/inbox/12`, middleware injeta `inboxId=12` no store de filtros, a lista atualiza

