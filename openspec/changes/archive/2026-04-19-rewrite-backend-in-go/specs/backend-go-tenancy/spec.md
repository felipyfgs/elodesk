## ADDED Requirements

### Requirement: OrgScopeMiddleware valida membership

O backend SHALL fornecer middleware `OrgScope` que:

1. Extrai `accountId` de (a) path param `:accountId`, (b) header `X-Account-Id`.
2. Confere existência do `Account`; senão 404 `"account not found"`.
3. Confere `AccountUser` do user corrente naquele account; senão 403 `"account access denied"`.
4. Popula `c.Locals("accountId", id)` e `c.Locals("role", role)`.

#### Scenario: user pertence à account

- **WHEN** JWT válido + `:accountId` pertence a account em que user tem membership
- **THEN** passa e `c.Locals("accountId")` é populado

#### Scenario: account inexistente

- **WHEN** `:accountId` não existe no DB
- **THEN** retorna 404 com `{message:"account not found"}`

#### Scenario: cross-tenant

- **WHEN** user autenticado tenta acessar account alheia
- **THEN** retorna 403 com `{message:"account access denied"}`

### Requirement: RolesMiddleware para operações privilegiadas

O backend SHALL fornecer `RolesRequired(roles ...Role)` middleware chainable que lê `c.Locals("role")` e rejeita 403 se a role não está na lista permitida.

#### Scenario: AGENT tenta criar inbox

- **WHEN** user AGENT chama `POST /api/v1/accounts/:aid/inboxes` (exige OWNER/ADMIN)
- **THEN** retorna 403

#### Scenario: ADMIN cria inbox

- **WHEN** user ADMIN chama a mesma rota
- **THEN** passa e service é invocado

### Requirement: Helpers CurrentUser e CurrentAccount

O backend SHALL expor helpers `CurrentUser(c *fiber.Ctx) *model.User` e `CurrentAccount(c *fiber.Ctx) *model.Account` que lêem de `c.Locals(...)` e retornam structs populadas; em ausência, retornam `nil` (caller decide).

#### Scenario: handler acessa usuário corrente

- **WHEN** handler em rota com `JwtAuth` chama `CurrentUser(c)`
- **THEN** retorna struct `User{ID, Email, Name}` correspondente ao JWT `sub`

#### Scenario: handler acessa account do scope

- **WHEN** handler em rota com `JwtAuth + OrgScope` chama `CurrentAccount(c)`
- **THEN** retorna struct `Account{ID, Name, Slug}`

### Requirement: Isolamento multi-tenant em repos

Todos os repos (`contact_repo`, `conversation_repo`, `message_repo`, `inbox_repo`, `channel_api_repo`, `attachment_repo`) SHALL exigir `accountID` em toda query de leitura/escrita que toca tabelas tenant-scoped. Índices compostos `(account_id, ...)` em `contacts`, `conversations`, `messages`, `inboxes`.

#### Scenario: query sem accountId falha

- **WHEN** um repo expõe `FindByID(ctx, id)` sem `accountID`
- **THEN** teste unitário dedicado acusa violação (repo tem que aceitar `accountID` ou não expor função pública sem ele)

#### Scenario: teste cross-tenant explícito

- **WHEN** user da account A chama `GET /api/v1/accounts/B/conversations`
- **THEN** 403 e nenhuma linha é retornada
