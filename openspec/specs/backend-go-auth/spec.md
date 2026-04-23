# backend-go-auth Specification

## Purpose
TBD - created by archiving change rewrite-backend-in-go. Update Purpose after archive.
## Requirements
### Requirement: Registro de usuário com Argon2id

O backend SHALL expor `POST /api/v1/auth/register` aceitando `{email, password, name, accountName?}`. A senha MUST ser hashada com `argon2id` (parâmetros do `alexedwards/argon2id` padrão). O handler cria `User`, `Account` (slug derivado do email com sufixo random de 3 bytes), e `AccountUser` com role `OWNER` numa única transação pgx.

#### Scenario: registro com email novo

- **WHEN** `POST /api/v1/auth/register` com `{email:"a@b.com", password:"pass1234", name:"A"}`
- **THEN** retorna 201 com `{user:{id,email,name,createdAt}, account:{id,name,slug}, accessToken, refreshToken}`
- **AND** o campo `passwordHash` nunca aparece na resposta

#### Scenario: email duplicado

- **WHEN** já existe User com o mesmo email
- **THEN** retorna 409 com `{message:"email already registered"}`

### Requirement: Login com email/senha

O backend SHALL expor `POST /api/v1/auth/login` aceitando `{email, password}`. Senha validada com `argon2id.ComparePasswordAndHash`. Sucesso retorna `{user, accessToken, refreshToken}`; falha MUST responder 401 com mensagem genérica `"invalid credentials"` (não distinguir email inexistente de senha errada).

#### Scenario: credenciais corretas

- **WHEN** email e senha válidos
- **THEN** retorna 200 com tokens e refresh token é salvo como SHA-256 hash em `refresh_tokens` com TTL 30 dias

#### Scenario: credenciais erradas

- **WHEN** email inexistente ou senha errada
- **THEN** retorna 401 com `{message:"invalid credentials"}`

### Requirement: Refresh com rotação

O backend SHALL expor `POST /api/v1/auth/refresh` aceitando `{refreshToken}`. O handler MUST rotacionar: emite novo par, marca o antigo como `revoked_at=now()`, reaproveita `family_id`. Reuso de refresh revogado MUST revogar TODA a família (defesa anti-replay).

#### Scenario: refresh válido

- **WHEN** refresh ainda ativo
- **THEN** retorna 200 com novo `{accessToken, refreshToken}` e registra revoked_at no antigo

#### Scenario: refresh replay

- **WHEN** refresh já revogado é apresentado
- **THEN** retorna 401 com `{message:"refresh token reuse detected"}`
- **AND** todos os refresh tokens ativos da mesma família ficam `revoked_at=now()`

### Requirement: Logout

O backend SHALL expor `POST /api/v1/auth/logout` (autenticado por JWT) que revoga o refresh token corrente; com body `{allDevices: true}` revoga TODOS os refresh tokens ativos do user.

#### Scenario: logout device atual

- **WHEN** `POST /api/v1/auth/logout {refreshToken}`
- **THEN** o refresh token dessa sessão é revogado e retorna 204

#### Scenario: logout all devices

- **WHEN** `POST /api/v1/auth/logout {allDevices:true}`
- **THEN** todos os refresh ativos do user são revogados e retorna 204

### Requirement: JWT middleware

O backend SHALL fornecer middleware `JwtAuth` que valida header `Authorization: Bearer <token>`, popula `fiber.Ctx.Locals("user", {id,email,name})` e rejeita com 401 em token inválido/expirado.

#### Scenario: Bearer válido

- **WHEN** request chega com `Authorization: Bearer <jwt válido>`
- **THEN** handler acessa `c.Locals("user")` populado e processa normalmente

#### Scenario: Bearer ausente ou inválido

- **WHEN** header ausente ou token expirado
- **THEN** retorna 401 sem chamar handler

### Requirement: Endpoints de recuperação de senha

O backend SHALL adicionar `POST /auth/forgot`, `GET /auth/reset/:token/validate`, `POST /auth/reset` conforme definido na capability `backend-auth-recovery`. A reset bem-sucedida MUST revogar todos os refresh tokens ativos do usuário e registrar evento em audit log `user.password_reset`.

#### Scenario: reset integra com revoke existente

- **WHEN** senha é redefinida com sucesso via `/auth/reset`
- **THEN** a mesma rotina de revoke usada em logout all-devices zera refresh tokens ativos

### Requirement: Endpoints MFA opcionais

O backend SHALL adicionar `POST /auth/mfa/setup`, `POST /auth/mfa/enable`, `POST /auth/mfa/disable`, `POST /auth/mfa/verify` conforme capability `backend-auth-mfa`. Login com `mfa_enabled=true` MUST exigir step adicional antes de emitir par JWT.

#### Scenario: login sem MFA continua inalterado

- **WHEN** user com `mfa_enabled=false` faz login
- **THEN** comportamento atual é preservado — retorna `{user, accessToken, refreshToken}` direto

#### Scenario: login com MFA interrompe fluxo

- **WHEN** user com `mfa_enabled=true` passa na senha
- **THEN** retorna `{mfa_required: true, mfa_token}` sem accessToken; cliente precisa completar via `/auth/mfa/verify`

### Requirement: Aceite de convite

O backend SHALL adicionar `POST /auth/invitations/:token/accept` (público, sem JWT) usando token de convite emitido por `backend-agents`. Sucesso cria `User` + `AccountUser` em transação e retorna par JWT.

#### Scenario: aceite válido

- **WHEN** POST com token válido e senha ≥8 chars
- **THEN** retorna 200 com `{user, accessToken, refreshToken}`; invitation é marcada `consumed_at`

#### Scenario: token expirado

- **WHEN** POST com token expirado
- **THEN** retorna 404 `{error: "invalid_or_expired_invitation"}`

