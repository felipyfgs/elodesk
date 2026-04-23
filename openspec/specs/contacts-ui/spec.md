# contacts-ui Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: /contacts segue o padrão pages/customers.vue do template

O frontend SHALL reescrever `frontend/app/pages/contacts/index.vue` adotando o padrão completo de `_refs/dashboard/app/pages/customers.vue`: `UDashboardPanel id="contacts"` → `UDashboardNavbar` (leading `UDashboardSidebarCollapse`, right `ContactsAddModal` trigger) → `UDashboardToolbar` (left busca + filtros, right seletor de colunas) → `#body` com `UTable` + paginação.

A tabela MUST usar `@tanstack/table-core` com `TableColumn<Contact>[]` definindo:

- Coluna `select` (checkbox header + cell) — mesmo padrão do template
- Coluna `name` com `UAvatar` + nome + username (célula idêntica a `pages/customers.vue`)
- Coluna `email` com sorting header (`UButton` neutral/ghost com ícones `arrow-up-narrow-wide`/`arrow-down-wide-narrow`/`arrow-up-down`)
- Colunas `phone`, `location`, `labels` (badges), `last_activity`, `created_at`
- Coluna `status` com `UBadge` colorido (subscribed=success, unsubscribed=error, bounced=warning) e `filterFn: 'equals'`
- Coluna `actions` com `UDropdownMenu` alinhado à direita

Controles reativos:

- `columnFilters = ref([{ id: 'email', value: '' }])`
- `columnVisibility = ref()`
- `rowSelection = ref({})`
- `pagination = ref({ pageIndex: 0, pageSize: 10 })`

#### Scenario: busca server-side

- **WHEN** usuário digita "maria" no `UInput` de busca
- **THEN** após debounce 300 ms, `GET /accounts/:aid/contacts?search=maria&page=1` retorna dados, store atualiza e a tabela re-renderiza

### Requirement: Decomposição de componentes em `components/contacts/`

O frontend SHALL criar os componentes abaixo em `frontend/app/components/contacts/` — mesmo formato das pastas `customers/` e `home/` do template:

| Componente | Responsabilidade | Baseado em |
|---|---|---|
| `ContactsAddModal.vue` | Modal de criação com `UForm` Zod (name, email obrigatórios) | `customers/AddModal.vue` |
| `ContactsDeleteModal.vue` | Modal de confirmação com contagem dinâmica ("Delete N contacts") | `customers/DeleteModal.vue` |
| `ContactsFilterSidebar.vue` | Sidebar de filtros + lista de segments salvos | — (novo) |
| `ContactsBulkToolbar.vue` | Toolbar contextual (Add label, Remove label, Delete, Export CSV) | — (novo) |
| `ContactsImportDropzone.vue` | Drag-and-drop + preview das 10 primeiras linhas | — (novo) |
| `ContactsImportReport.vue` | Relatório pós-import (inseridos/atualizados/erros) | — (novo) |

Store Pinia Options em `frontend/app/stores/contacts.ts` com actions: `fetchPage`, `setAll`, `upsert`, `removeMany`, `applyBulkLabel`.

Schemas Zod em `frontend/app/schemas/contacts.ts`: `contactCreateSchema`, `contactUpdateSchema`, `contactImportRowSchema`, `contactSegmentSchema`.

#### Scenario: componente segue o template

- **WHEN** `ContactsAddModal.vue` é implementado
- **THEN** expõe `<UModal>` com slot de trigger, `<UForm :schema="schema" :state="state">`, `UFormField`s para name/email, botão submit com toast `color: 'success'` — API idêntica a `customers/AddModal.vue`

### Requirement: Filtros avançados e segments reutilizam FilterBuilder

`ContactsFilterSidebar.vue` SHALL reutilizar o componente `FilterBuilder.vue` existente em `frontend/app/components/FilterBuilder.vue` (movê-lo para `components/filters/FilterBuilder.vue` se for compartilhado com conversations/reports). Segments persistem via store `useSavedFiltersStore` (já existe) com `entity: 'contact'`.

Rotas de segment: `frontend/app/pages/contacts/segments/[id].vue` — re-export de `index.vue` com filtro pré-aplicado.

#### Scenario: criar segment

- **WHEN** usuário monta filtro "Clientes ativos com label VIP" e salva
- **THEN** `POST /saved_filters` persiste, sidebar lista o novo segment, rota `/contacts/segments/:id` fica acessível

### Requirement: Bulk select com toolbar contextual

Quando `Object.keys(rowSelection).length > 0`, `ContactsBulkToolbar.vue` SHALL aparecer no slot `#right` do `UDashboardNavbar`, replicando o padrão do `DeleteModal` do template (mostra contagem dinâmica). Ações: Add label (dropdown de labels), Remove label, Delete (via `ContactsDeleteModal`), Export CSV (download do subset).

#### Scenario: aplicar label em bulk

- **WHEN** usuário seleciona 10 contatos e aplica label "VIP"
- **THEN** um `POST /contacts/bulk_label` com `{ids, label_id, op: 'add'}` é chamado; store atualiza em lote; toast confirma "10 contatos atualizados"

### Requirement: Import CSV em /contacts/import

O frontend SHALL expor `frontend/app/pages/contacts/import.vue` (layout dashboard) com:

- `ContactsImportDropzone.vue` para upload drag-and-drop (aceita `.csv`, máx 10 MB)
- Preview em `UTable` das 10 primeiras linhas
- Mapeamento de colunas via `USelect` por coluna destino (name, email, phone, identifier, custom attrs)
- Botão "Importar" chama `POST /accounts/:aid/contacts/import` (multipart) e navega para tela de resultado
- `ContactsImportReport.vue` exibe: inseridos, atualizados, erros (tabela com número da linha + motivo), botão "Baixar CSV de erros"

#### Scenario: import com erros parciais

- **WHEN** usuário envia CSV com 100 linhas, 3 com email inválido
- **THEN** backend retorna `{inserted: 97, updated: 0, errors: [{row: 12, reason: 'invalid email'}, ...]}`; o componente renderiza tudo e oferece download dos erros

### Requirement: Detalhe do contato com UTabs

O frontend SHALL atualizar `frontend/app/pages/contacts/[id].vue` adotando o padrão de `pages/settings.vue` do template: `UDashboardPanel` com `UDashboardNavbar` + `UDashboardToolbar` contendo `UNavigationMenu highlight` com as abas, e `<NuxtPage />` renderizando cada aba via rotas aninhadas:

- `frontend/app/pages/contacts/[id]/index.vue` — "Visão geral" (perfil editável com `UForm` + custom attrs)
- `frontend/app/pages/contacts/[id]/conversations.vue` — histórico paginado (lista virtualizada)
- `frontend/app/pages/contacts/[id]/notes.vue` — `NoteEditor.vue` já existente
- `frontend/app/pages/contacts/[id]/events.vue` — timeline do audit log

Componentes de apoio em `frontend/app/components/contacts/detail/`: `ContactHeader.vue`, `ContactCustomAttributes.vue`, `ContactEventsTimeline.vue`.

#### Scenario: edição inline do nome

- **WHEN** usuário clica no nome, troca e confirma
- **THEN** `PATCH /contacts/:id` é chamado, valor atualiza sem refresh, aba "Eventos" ganha item "Nome alterado"

