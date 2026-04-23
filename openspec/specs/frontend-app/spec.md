# frontend-app Specification

## Purpose
TBD - created by archiving change helpdesk-core. Update Purpose after archive.

## Requirements

### Requirement: Stores Pinia pros novos domínios

O frontend SHALL adicionar stores Pinia com estrutura consistente (state, actions, upsert helpers) pra:

- `stores/labels.ts` — `list`, `byId(id)`, `fetchAll()`, `create()`, `update()`, `remove()`, `apply(targetType, targetId, labelId)`, `unapply(...)`, handler pra eventos realtime `label.added`/`label.removed`/`label.deleted`.
- `stores/teams.ts` — `list`, `byId`, `fetchAll`, CRUD, `addMembers(teamId, userIds)`, `removeMembers(teamId, userIds)`.
- `stores/cannedResponses.ts` — `list`, `fetchAll`, CRUD, `search(term)` client-side.
- `stores/notes.ts` — buckets por `contactId` (`Record<number, Note[]>`), `fetchForContact`, `create`, `update`, `remove`, handler de `note.created`.
- `stores/customAttributes.ts` — `definitions: {contact: Def[], conversation: Def[]}`, `fetchAll`, CRUD de definitions, `setValues(target, id, values)`, `removeValues(target, id, keys)`.
- `stores/savedFilters.ts` — `list` (apenas do user atual), `byType(filterType)`, CRUD, `apply(filterId, page)` que chama endpoint `/conversations/filter` ou `/contacts/filter`.

Todos os stores SHALL usar `useApi()` (composable existente) pra HTTP com JWT + X-Account-Id.

#### Scenario: store labels reage a evento realtime

- **WHEN** cliente recebe via websocket `{type:"label.added", payload:{conversation_id:42, label_id:3}}`
- **THEN** store `labels` atualiza a associação em cache (se conversation está aberta) e a UI reflete sem refetch manual

#### Scenario: store customAttributes valida no frontend

- **WHEN** agent tenta setar valor inválido via UI (ex.: string onde é number)
- **THEN** o store rejeita antes do request usando schema Zod gerado a partir da definition

### Requirement: Novas pages de settings

O frontend SHALL adicionar pages sob `app/pages/settings/`:

- `labels.vue` — lista de labels (TanStack table), modal de create/edit, badge com color preview.
- `teams.vue` — lista de teams, modal de create/edit com multi-select de members.
- `canned.vue` — lista de canned responses, modal de create/edit com editor de content (`UTextarea` simples por enquanto).
- `attributes.vue` — lista de custom attribute definitions agrupadas por `attribute_model`; modal de create com selector de tipo e, condicionalmente, editor de `attribute_values` para type=list.

Todas as pages MUST:
- Usar Nuxt UI 4 components (UTable, UModal, UForm, UInput, UButton, UBadge, UDropdown).
- Aplicar i18n via `useI18n()` com keys novas em `pt-BR.json` e `en.json`.
- Guardar acesso via `definePageMeta({ middleware: ['auth'] })` + checagem de role Admin+ no nível do page (mostrar 403 se agent acessar).
- Usar Zod schema pra validação no submit de forms.

#### Scenario: admin cria label pela UI

- **WHEN** admin acessa `/settings/labels`, clica "Nova label", preenche título e color
- **THEN** form valida com Zod; submit chama `stores.labels.create(...)`; após 201, modal fecha e label aparece na lista

#### Scenario: agent redirecionado ao acessar settings admin

- **WHEN** agent (role=Agent) navega pra `/settings/labels`
- **THEN** page exibe estado 403 / "Acesso restrito" sem quebrar a navegação

### Requirement: Contact detail

O frontend SHALL expor uma UI de detalhe de contact acessível via:

- Deep link `/contacts/{id}` (page nova `app/pages/contacts/[id].vue`).
- Slideover a partir do listing `/contacts` (clique no row abre `USlideover`).
- Botão "ver contato" no header da conversation abre o slideover.

A UI MUST exibir:

- Perfil editável: `name`, `email`, `phone_number`, `identifier` com form Zod que chama `PATCH /contacts/{id}`.
- Labels aplicadas: `LabelPicker` component pra adicionar/remover.
- Custom attributes: um field dinâmico por definition (`CustomAttributeField`) renderizando baseado em `attribute_display_type`.
- Notes: editor inline (`NoteEditor`) com lista abaixo ordenada por mais recentes; editar/deletar só aparece pra autor ou admin.
- Histórico de conversations: lista clicável que navega pra `/conversations/{id}`, com paginação e preview.

#### Scenario: abrir slideover a partir do listing

- **WHEN** agent clica num row da tabela `/contacts`
- **THEN** slideover abre à direita, `GET /contacts/{id}` é chamado, fields renderizam; fechar com ESC ou botão X

#### Scenario: editar email e salvar

- **WHEN** agent edita campo email e submete
- **THEN** Zod valida formato; request `PATCH /contacts/{id}` é enviado; em caso de 409 (email taken) mostra toast com erro

### Requirement: Conversation assignment UI

O frontend SHALL adicionar no header da conversation aberta:

- Dropdown de agente (`UDropdown` com busca) populado a partir de `stores.accountUsers` ou endpoint `/accounts/{aid}/agents` (se existir; senão buscar via novo endpoint documentado na Onda 1).
- Dropdown de team populado a partir de `stores.teams`.
- Selecionar uma opção dispara `POST /conversations/{id}/assignments` com o field correspondente.
- Exibir nome do agente + nome do team atualmente atribuídos, ou "Não atribuído".

A sidebar de filtro de conversations MUST ganhar:
- Filtro "Meus conversations" (assignee_id=user atual).
- Filtro "Não atribuídas" (assignee_id=null).
- Lista dos teams do user (filtro por team_id).

#### Scenario: atribuir agente via header

- **WHEN** agent clica no dropdown, busca "maria" e seleciona Maria
- **THEN** POST `/conversations/42/assignments` com `{assignee_id: 15}`; após 200, header atualiza; todos os clientes na room `conversation.42` recebem `conversation.assignment_changed` e atualizam em tempo real

### Requirement: Canned response picker no composer

O frontend SHALL adicionar ao composer de mensagem um picker de canned responses:

- Gatilho: digitar `/` no começo da mensagem OU clicar botão "⚡" no toolbar.
- Dropdown mostra lista de respostas buscando por `short_code` (matching client-side com os dados do `stores.cannedResponses`).
- Ao selecionar, conteúdo da resposta é inserido no composer (substituindo o `/` trigger) e dropdown fecha.

#### Scenario: agente usa canned response

- **WHEN** agent digita `/greet-new` no composer
- **THEN** dropdown filtra pra esse short_code; apertar Enter insere o content da resposta no composer; comando é substituído pelo content completo

### Requirement: Saved filters sidebar

O frontend SHALL adicionar na sidebar de `/conversations` e `/contacts`:

- Seção "Filtros salvos" com lista dos filtros do user atual (`stores.savedFilters.byType(currentType)`).
- Clique aplica filtro via `stores.savedFilters.apply(id)` → popula a lista principal com os resultados.
- Botão "+ Novo filtro" abre `FilterBuilder` modal que permite montar `{operator, conditions}` com pickers de attribute_key (incluindo custom attributes da account) e filter_operator.
- Botão "Salvar filtro atual" habilita se o user já aplicou um filtro ad-hoc.

#### Scenario: aplicar filtro salvo

- **WHEN** agent clica num filtro salvo "Urgentes sem atribuição"
- **THEN** frontend chama `POST /conversations/filter` com o query desse filtro; lista principal repopula com os resultados paginados

### Requirement: Adoção de Zod pra validação de forms

O frontend SHALL adotar `zod` (já presente em `package.json`) como o padrão de validação de forms novos desta onda:

- Schemas ficam em `app/schemas/{domain}.ts` (ex.: `app/schemas/label.ts`).
- Forms usam `const form = reactive({...})` com `const schema = z.object({...})`; submit chama `schema.safeParse(form)` e exibe `result.error.issues` mapeados pro `UFormField` via `error` prop.
- Sem wrappers tipo `@vee-validate/zod`; uso direto.

#### Scenario: form de label valida título vazio

- **WHEN** admin submete form de criar label com `title=""`
- **THEN** `safeParse` retorna erro; `UFormField` de título mostra mensagem "Título obrigatório"; nada é enviado ao backend

### Requirement: i18n keys pt-BR + en pra features da Onda 1

O frontend SHALL adicionar keys em `i18n/locales/pt-BR.json` e `i18n/locales/en.json` cobrindo:

- `labels.*` (list title, create button, edit, delete, title/color/description/show-on-sidebar fields, empty state)
- `teams.*`, `cannedResponses.*`, `notes.*`, `customAttributes.*`, `savedFilters.*`, `contactDetail.*`, `conversation.assignment.*`

pt-BR é primário; en é fallback. Nenhum texto literal em componente Vue (todos via `t('...')`).

#### Scenario: troca de idioma reflete nas novas pages

- **WHEN** user muda idioma pra `en`
- **THEN** pages `/settings/labels`, `/settings/teams`, etc. renderizam textos em inglês via keys `en.json`
