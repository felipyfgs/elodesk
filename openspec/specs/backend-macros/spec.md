# backend-macros Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: CRUD /accounts/:aid/macros

O backend SHALL expor CRUD REST completo em `/api/v1/accounts/:aid/macros`. Modelo: `{id, name, visibility (personal/account), conditions (jsonb), actions (jsonb), created_by, created_at, updated_at}`. Scope por `account_id` em todas as queries.

#### Scenario: criar macro

- **WHEN** `POST /macros` com payload válido
- **THEN** 201 retorna macro persistida; `created_by` vem do JWT

#### Scenario: listar apenas do account

- **WHEN** `GET /macros` por user de account A
- **THEN** retorna apenas macros com `account_id=A`, nunca da account B

### Requirement: Executar macro em conversa

O backend SHALL expor `POST /accounts/:aid/conversations/:convId/apply_macro/:macroId`. Executa ações em ordem; ações suportadas: `assign_agent`, `assign_team`, `add_label`, `remove_label`, `change_status`, `snooze_until`, `send_message (texto)`, `add_note`. Erro em uma ação MUST interromper e reverter transacionalmente.

#### Scenario: execução atômica

- **WHEN** macro tem 3 ações e a 2ª falha (ex: label não existe)
- **THEN** nenhuma das 3 é persistida; retorna 500 `{error, failed_action_index: 1}`

### Requirement: Validação de conditions/actions via schema

O backend SHALL validar `conditions` e `actions` contra um JSON schema estático antes de persistir. Ações desconhecidas MUST ser rejeitadas com 400.

#### Scenario: ação inválida

- **WHEN** payload inclui `action_name: "delete_account"`
- **THEN** retorna 400 `{error: "invalid_action", allowed: [...]}`

