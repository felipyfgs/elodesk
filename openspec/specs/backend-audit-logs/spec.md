# backend-audit-logs Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: Tabela audit_logs

O backend SHALL criar tabela `audit_logs` com `(id BIGSERIAL, account_id BIGINT, user_id BIGINT NULL, action VARCHAR, entity_type VARCHAR, entity_id BIGINT NULL, metadata JSONB, ip_address INET, user_agent TEXT, created_at TIMESTAMPTZ)`. Índices em `(account_id, created_at DESC)` e `(account_id, entity_type, entity_id)`. Particionamento mensal.

#### Scenario: migration aplicada

- **WHEN** migration roda
- **THEN** tabela existe com índices e primeira partição do mês corrente

### Requirement: Middleware de registro

O backend SHALL expor helper `audit.Log(ctx, action, entityType, entityID, metadata)` usado nos handlers críticos: `user.invited`, `user.role_changed`, `user.password_changed`, `inbox.created`, `inbox.deleted`, `conversation.resolved`, `conversation.deleted`, `macro.executed`, `sla.breached`, `webhook.configured`, `agent.removed`.

#### Scenario: resolver conversa gera audit log

- **WHEN** PATCH conversation `{status: Resolved}`
- **THEN** linha em `audit_logs` com action=`conversation.resolved`, entity_type=`conversation`, entity_id=<id>, metadata inclui status anterior

### Requirement: GET /accounts/:aid/audit_logs

O backend SHALL expor `GET /accounts/:aid/audit_logs` com filtros: `from, to, action, entity_type, user_id, page, pageSize` (default 50, max 200). Apenas Admin+ acessa. Scope rigoroso por `account_id`.

#### Scenario: filtrar por ação

- **WHEN** `GET /audit_logs?action=user.role_changed&from=2026-04-01`
- **THEN** retorna apenas eventos desse tipo no período, sem vazar de outra account

### Requirement: Retenção via job

Um job asynq diário SHALL deletar entradas com `created_at < now() - 90 days` em batches. Job MUST logar volume removido.

#### Scenario: retenção expira partição antiga

- **WHEN** job roda e há partição com dados > 90 dias
- **THEN** partição é dropada por `DROP TABLE IF EXISTS audit_logs_YYYYMM`, log registra contagem

