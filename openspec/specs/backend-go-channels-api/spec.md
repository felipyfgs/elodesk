## ADDED Requirements

### Requirement: Assignment de conversation via endpoint unificado

O backend SHALL expor `POST /api/v1/accounts/{aid}/conversations/{id}/assignments` aceitando body `{assignee_id?: number|null, team_id?: number|null}`:

- Passar `assignee_id=null` desatribui o agente; omitir a key não altera.
- Passar `team_id=null` desatribui o team; omitir não altera.
- `assignee_id` informado MUST existir em `account_users` ativo da account.
- `team_id` informado MUST existir em `teams` da account.
- Após update bem-sucedido, emitir evento realtime `{type:"conversation.assignment_changed", payload:{conversation_id, assignee_id, team_id, account_id}}` na room `conversation.{id}` e `account.{aid}`.
- Role: `Agent+` pode atribuir/desatribuir conversations; `Admin+` pode reassinar de outro agente sem estar nele.

#### Scenario: atribuir agente e team simultaneamente

- **WHEN** agent envia `POST /conversations/42/assignments` com `{"assignee_id":10,"team_id":5}`
- **THEN** conversation 42 passa a ter `assignee_id=10` e `team_id=5`; broadcast `conversation.assignment_changed` dispara com payload completo

#### Scenario: desatribuir agente mantendo team

- **WHEN** body é `{"assignee_id":null}` (team_id omitido)
- **THEN** `assignee_id` vira NULL, `team_id` permanece, broadcast dispara

#### Scenario: rejeitar assignee fora da account

- **WHEN** body contém `assignee_id` de user que não pertence à account
- **THEN** retorna 400 com erro `assignee_not_in_account`, nada é alterado

### Requirement: Listagem de conversations com filtros expandidos

O endpoint `GET /api/v1/accounts/{aid}/conversations` SHALL aceitar os query params adicionais:

- `?team_id=<id>` — filtra por team específico.
- `?team_id=null` — retorna apenas conversations sem team.
- `?assignee_id=<id>` ou `?assignee_id=null` (existia implicitamente; documentar).
- `?label=<title>` ou `?labels=urgente,vip` — filtra por label(s) (match por `label_taggings` com `taggable_type='conversation'`).

Filtros são combinados com AND. Paginação e ordenação existentes mantêm-se.

#### Scenario: filtrar por team e label juntos

- **WHEN** agent envia `GET /conversations?team_id=5&labels=urgente`
- **THEN** retorna apenas conversations com `team_id=5` E com label "urgente" aplicada

### Requirement: PATCH de contact

O backend SHALL expor `PATCH /api/v1/accounts/{aid}/contacts/{id}` body parcial com qualquer subset de `{name, email, phone_number, identifier}`:

- `email` MUST ser unique por account quando informado; null permitido.
- `phone_number` valida formato E.164 quando informado.
- `identifier` MUST ser unique quando informado.
- Role: `Agent+`.

#### Scenario: atualizar nome do contact

- **WHEN** agent envia `PATCH /contacts/7` com `{"name":"João Silva"}`
- **THEN** retorna 200 com contact atualizado, demais campos inalterados

#### Scenario: rejeitar email duplicado

- **WHEN** agent tenta setar email já usado por outro contact na mesma account
- **THEN** retorna 409 com erro `email_taken`

### Requirement: Histórico de conversations de um contact

O backend SHALL expor `GET /api/v1/accounts/{aid}/contacts/{id}/conversations` que retorna todas as conversations desse contact, ordenadas por `last_activity_at DESC`, com paginação (default 25/page).

Resposta inclui os mesmos fields do listing de conversations (status, assignee, team, labels resumidas, last_message preview).

Role: `Agent+`.

#### Scenario: listar histórico do contact

- **WHEN** agent envia `GET /contacts/7/conversations`
- **THEN** retorna conversations do contact 7 da account do path, ordenadas da mais recente pra mais antiga, paginadas

### Requirement: Endpoints de labels em conversation e contact

O backend SHALL expor em `backend-go-channels-api`:

- `GET|POST|DELETE /api/v1/accounts/{aid}/conversations/{id}/labels` — ver spec `backend-go-labels`.
- `GET|POST|DELETE /api/v1/accounts/{aid}/contacts/{id}/labels` — ver spec `backend-go-labels`.

(Esta requirement documenta a presença dos endpoints no router de channels-api, com detalhes comportamentais na capability `backend-go-labels`.)

#### Scenario: label endpoints roteados dentro de conversations/contacts

- **WHEN** frontend faz POST/DELETE em `/conversations/{id}/labels` ou `/contacts/{id}/labels`
- **THEN** o router de channels-api roteia pra handler de labels, que retorna shape consistente com os outros endpoints da API

### Requirement: Endpoints de notes em contact

O backend SHALL expor em `backend-go-channels-api`:

- `GET|POST /api/v1/accounts/{aid}/contacts/{cid}/notes`
- `PATCH|DELETE /api/v1/accounts/{aid}/contacts/{cid}/notes/{nid}`

Detalhes comportamentais em `backend-go-notes`.

#### Scenario: notes endpoint retorna shape padronizado

- **WHEN** frontend envia GET em `/contacts/7/notes`
- **THEN** response segue o envelope padrão da API (objeto com `data` array ou array direto, consistente com outros listings)

### Requirement: Endpoints de custom_attributes em conversation e contact

O backend SHALL expor em `backend-go-channels-api`:

- `POST|DELETE /api/v1/accounts/{aid}/conversations/{id}/custom_attributes`
- `POST|DELETE /api/v1/accounts/{aid}/contacts/{id}/custom_attributes`

Detalhes em `backend-go-custom-attributes`.

#### Scenario: setar custom attrs retorna objeto atualizado

- **WHEN** frontend envia POST com `{churn_risk: 0.8}`
- **THEN** retorna o contact/conversation com `additional_attributes` atualizado

### Requirement: Endpoints de filter apply

O backend SHALL expor em `backend-go-channels-api`:

- `POST /api/v1/accounts/{aid}/conversations/filter` body `{query, page?, per_page?}`
- `POST /api/v1/accounts/{aid}/contacts/filter` body `{query, page?, per_page?}`

Detalhes em `backend-go-saved-filters`.

#### Scenario: filter apply retorna paginado

- **WHEN** frontend envia POST com query válida e per_page=50
- **THEN** retorna `{data: [...], meta: {page, per_page, total}}` com até 50 resultados
