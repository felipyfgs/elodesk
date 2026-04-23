# backend-user-profile Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: PUT /users/:id para auto-edição

O backend SHALL expor `PUT /api/v1/users/:id` permitindo apenas o próprio usuário editar seu perfil (checagem `ctx.user.id == :id`). Campos editáveis: `name, email, avatar_url, current_password, new_password`. Troca de senha exige `current_password` válido (Argon2id compare).

#### Scenario: auto-edição de nome

- **WHEN** usuário envia `{name: "Novo Nome"}`
- **THEN** retorna 200 com user atualizado; outros campos não mudam

#### Scenario: tentativa de editar outro usuário

- **WHEN** user A envia PUT para `/users/B`
- **THEN** retorna 403 `{error: "forbidden"}`

### Requirement: Troca de senha revoga refresh tokens

Quando a senha for alterada com sucesso, o backend MUST revogar todos os refresh tokens ativos do usuário e emitir evento de audit log `user.password_changed`.

#### Scenario: senha trocada

- **WHEN** usuário troca senha com sucesso
- **THEN** todas as linhas em `refresh_tokens WHERE user_id=... AND revoked_at IS NULL` ganham `revoked_at=now()`

### Requirement: Avatar via presigned MinIO

O backend SHALL aceitar `avatar_url` no PUT apontando para objeto já uploadado via presigned PUT. O caminho MUST começar com `{accountId}/avatars/{userId}/` e ser validado antes de persistir.

#### Scenario: avatar path inválido

- **WHEN** usuário envia `avatar_url: "https://outroservidor.com/pic.png"`
- **THEN** retorna 400 `{error: "invalid_avatar_path"}`

