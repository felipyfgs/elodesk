## ADDED Requirements

### Requirement: Isolamento multi-tenant obrigatório por accountId

Toda tabela com dados de negócio (Inbox, ChannelWhatsapp, Contact, ContactInbox, Conversation, Message, Attachment, Label) SHALL incluir coluna `accountId` com foreign key para `Account`. Todos os repositórios MUST filtrar por `accountId` em toda query que lê/escreve essas tabelas. Índices compostos `(accountId, ...)` SHALL ser criados nas colunas frequentemente consultadas.

#### Scenario: query sem accountId falha em teste

- **WHEN** qualquer query de repo omite o filtro `accountId` no WHERE
- **THEN** teste unitário dedicado falha (regra enforceada via lint + teste explícito por repo)

#### Scenario: índice composto existe

- **WHEN** inspeção do schema Prisma
- **THEN** tabelas `Conversation`, `Message` e `Contact` têm `@@index([accountId, ...])` em campos de filtro comum

### Requirement: OrgScopeGuard valida membership

O backend SHALL fornecer `OrgScopeGuard` que extrai `accountId` da request (via header `X-Account-Id` ou parâmetro de rota `:accountId`), verifica que o user autenticado tem `AccountUser` para aquele `accountId`, e rejeita com 403 caso contrário. O guard MUST popular `request.accountId` para uso downstream.

#### Scenario: user pertence à account

- **WHEN** user autenticado faz request com `X-Account-Id` de uma account em que ele tem membership
- **THEN** a request passa e `request.accountId` é populado

#### Scenario: user não pertence à account

- **WHEN** user autenticado faz request com `X-Account-Id` de outra account
- **THEN** retorna 403 com `{message: "account access denied"}`

#### Scenario: account inexistente

- **WHEN** `X-Account-Id` aponta para uma Account que não existe
- **THEN** retorna 404 com `{message: "account not found"}` (não vaza existência)

### Requirement: RolesGuard para operações privilegiadas

O backend SHALL fornecer decorator `@Roles('OWNER', 'ADMIN')` + `RolesGuard` que valida o `role` da `AccountUser` do user corrente na `accountId` do request. Rotas de configuração (criar/deletar inbox, convidar agentes, mudar settings) MUST exigir `OWNER` ou `ADMIN`.

#### Scenario: AGENT tenta criar inbox

- **WHEN** user com role `AGENT` chama `POST /accounts/:id/inboxes`
- **THEN** retorna 403

#### Scenario: ADMIN cria inbox

- **WHEN** user com role `ADMIN` chama `POST /accounts/:id/inboxes`
- **THEN** passa pelo guard e prossegue para o service

### Requirement: Decorators @CurrentUser e @CurrentAccount

O backend SHALL fornecer parâmetros de rota `@CurrentUser()` (retorna `{id, email, name}`) e `@CurrentAccount()` (retorna `{id, slug}`) populados a partir do JWT e do `OrgScopeGuard`.

#### Scenario: controller acessa user autenticado

- **WHEN** controller declara `handle(@CurrentUser() user)` em rota protegida por JwtAuthGuard
- **THEN** `user` é o registro da tabela User correspondente ao `sub` do JWT

#### Scenario: controller acessa account do request

- **WHEN** controller declara `handle(@CurrentAccount() account)` em rota protegida por OrgScopeGuard
- **THEN** `account` é o registro da tabela Account correspondente ao `accountId` do request
