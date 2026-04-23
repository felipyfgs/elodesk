# backend-auth-recovery Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: POST /auth/forgot

O backend SHALL expor `POST /api/v1/auth/forgot` aceitando `{email}`. Resposta MUST ser sempre 200 com body genérico `{status: "sent"}` — sem vazar se email existe. Se existir, gera token aleatório (32 bytes), hasheado com SHA-256 em rest, TTL 30 min, single-use. Persiste em `password_reset_tokens (user_id, token_hash, expires_at, consumed_at)`.

#### Scenario: email existente

- **WHEN** email existe
- **THEN** token é gerado, logado via structured logger (`.Str("component", "auth").Str("event", "password_reset_requested")`), resposta é 200 genérica

#### Scenario: email inexistente

- **WHEN** email não existe
- **THEN** resposta é 200 idêntica à acima (tempo de resposta constante para prevenir timing attacks)

### Requirement: GET /auth/reset/:token/validate

O backend SHALL expor `GET /api/v1/auth/reset/:token/validate` que retorna `{valid: true}` se o token está ativo (não expirado, não consumido), senão 404.

#### Scenario: token válido

- **WHEN** GET com token ativo
- **THEN** 200 `{valid: true}`

#### Scenario: token expirado

- **WHEN** token existe mas `expires_at < now()`
- **THEN** 404 `{error: "invalid_or_expired_token"}`

### Requirement: POST /auth/reset

O backend SHALL expor `POST /api/v1/auth/reset` aceitando `{token, newPassword}`. Valida token (hash SHA-256 comparison + expiry), re-hasheia nova senha com Argon2id, atualiza `users.password_hash`, marca token como `consumed_at=now()`, revoga todos os refresh tokens ativos do usuário. Retorna 200 vazio.

#### Scenario: reset bem-sucedido

- **WHEN** token válido + senha ≥8 chars
- **THEN** password_hash atualizado, refresh tokens revogados, token marcado consumido

#### Scenario: reuso de token

- **WHEN** POST novamente com mesmo token já consumido
- **THEN** 404 `{error: "invalid_or_expired_token"}`

