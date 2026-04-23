# backend-auth-mfa Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: POST /auth/mfa/setup

O backend SHALL expor `POST /api/v1/auth/mfa/setup` (autenticado) que gera secret TOTP (RFC 6238, SHA-1, 30s, 6 dígitos), persiste em `users.mfa_secret_ciphertext` (AES-256-GCM via KEK) com `mfa_enabled=false`, e retorna `{otpauth_uri, secret}` para exibição do QR code.

#### Scenario: setup inicia

- **WHEN** usuário autenticado chama setup
- **THEN** retorna URI otpauth `otpauth://totp/Elodesk:<email>?secret=...&issuer=Elodesk`

### Requirement: POST /auth/mfa/enable

O backend SHALL expor `POST /api/v1/auth/mfa/enable` aceitando `{code}`. Valida código TOTP contra secret pendente; se correto, define `mfa_enabled=true`, gera 8 recovery codes (hash SHA-256 em `mfa_recovery_codes`), retorna recovery codes ONCE em plaintext.

#### Scenario: código correto

- **WHEN** POST com código TOTP válido
- **THEN** MFA ativado, 8 recovery codes retornados, plaintext não persiste

### Requirement: Login com MFA ativo

Quando `users.mfa_enabled=true`, `POST /auth/login` após senha correta SHALL retornar `{mfa_required: true, mfa_token}` (token efêmero, 5 min TTL) em vez de JWT. Cliente chama `POST /auth/mfa/verify` com `{mfa_token, code}` para completar login.

#### Scenario: login exige MFA

- **WHEN** user tem MFA e senha está correta
- **THEN** retorna 200 `{mfa_required: true, mfa_token}` sem accessToken

#### Scenario: verify com código correto

- **WHEN** POST verify com código TOTP ou recovery code válido
- **THEN** retorna par JWT completo; se recovery code foi usado, ele é marcado consumed

### Requirement: POST /auth/mfa/disable

O backend SHALL permitir desativar MFA mediante `{currentPassword}`. Limpa `mfa_secret_ciphertext` e zera recovery codes. Registra evento em audit log.

#### Scenario: desativar MFA

- **WHEN** user envia senha correta
- **THEN** `mfa_enabled=false`, secrets apagados, audit log `user.mfa_disabled` registrado

