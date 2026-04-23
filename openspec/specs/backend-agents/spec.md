# backend-agents Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: GET /accounts/:aid/agents

O backend SHALL expor `GET /api/v1/accounts/:aid/agents` retornando membros da account (join `users` + `account_users`) com `{id, name, email, role, last_active_at, status}`. Acesso: Admin+ (role ≥ 1). Status ∈ [active, invited, disabled].

#### Scenario: admin lista agentes

- **WHEN** `GET /agents` por admin autenticado
- **THEN** retorna array de membros da account, ordenados por `name ASC`

### Requirement: POST /accounts/:aid/agents/invite

O backend SHALL expor `POST /api/v1/accounts/:aid/agents/invite` aceitando `{email, role, name?}`. Gera magic link token (SHA-256 em rest, TTL 48h), persiste em `agent_invitations`, loga o link via structured logger. Retorna 201 com `{invitation_id, status: 'pending'}`.

#### Scenario: convite duplicado

- **WHEN** email já convidado com invite pendente
- **THEN** retorna 409 `{error: "invitation_already_pending"}`

### Requirement: POST /auth/invitations/:token/accept

O backend SHALL expor endpoint público `POST /api/v1/auth/invitations/:token/accept` com `{password, name?}`. Valida token (não expirado + não consumido), cria `User` se necessário, cria `AccountUser` com role do convite, marca invitation como `consumed_at=now()`, retorna par de tokens JWT.

#### Scenario: aceitar convite válido

- **WHEN** POST com token ativo e senha válida
- **THEN** retorna 200 com `{user, accessToken, refreshToken}` e agent aparece como `active` no `/agents`

### Requirement: PATCH /accounts/:aid/agents/:userId

O backend SHALL expor `PATCH /api/v1/accounts/:aid/agents/:userId` aceitando `{role?, status?}`. Só Owner pode promover outro Owner. Owner não pode ser rebaixado se for único da conta (retorna 400).

#### Scenario: rebaixar último owner

- **WHEN** owner tenta se rebaixar sendo único da account
- **THEN** retorna 400 `{error: "cannot_demote_last_owner"}`

