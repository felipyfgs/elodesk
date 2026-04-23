# backend-notifications Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: Tabela notifications

O backend SHALL criar tabela `notifications (id BIGSERIAL, account_id BIGINT, user_id BIGINT, type VARCHAR, payload JSONB, read_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ)`. Índice `(user_id, read_at, created_at DESC)` para listar unread rápido.

#### Scenario: migration aplicada

- **WHEN** migration roda
- **THEN** tabela existe com índices

### Requirement: Geração centralizada de notificações

O backend SHALL expor helper `notifications.Create(ctx, accountID, userID, type, payload)` usado por: mentions em mensagens, novo assignment, SLA breach, nova conversa em inbox (se configurado). Após persistir, emite evento WebSocket `notification.new` para a room `account:<aid>:user:<uid>`.

#### Scenario: mention gera notification

- **WHEN** agente é mencionado em mensagem via `@maria`
- **THEN** linha em `notifications` com `type='mention'`, WebSocket emite para user.Maria

### Requirement: GET /accounts/:aid/notifications

O backend SHALL expor `GET /notifications?status=unread|all&limit=&cursor=`. Só retorna do próprio user (`WHERE user_id = ctx.user.id`). Default status=unread, limit=25.

#### Scenario: listar unread

- **WHEN** `GET /notifications`
- **THEN** retorna notificações com `read_at IS NULL` ordenadas por `created_at DESC`

### Requirement: POST /notifications/:id/read e /mark_all_read

O backend SHALL expor `POST /notifications/:id/read` (marca uma) e `POST /notifications/mark_all_read` (marca todas do user). Ambos são idempotentes.

#### Scenario: marcar uma como lida

- **WHEN** POST com id válido pertencente ao user
- **THEN** `read_at = now()` e retorna 204

### Requirement: Preferências em users.notification_preferences

O backend SHALL adicionar coluna `users.notification_preferences JSONB` default `{}`. `PUT /users/:id/notification_preferences` atualiza. Geração MUST consultar preferências antes de criar; tipo desabilitado não persiste nem emite.

#### Scenario: usuário desabilita SLA breach

- **WHEN** `notification_preferences.sla_breached=false`
- **THEN** helper `notifications.Create` para esse user com type `sla_breached` retorna sem persistir

