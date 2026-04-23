# backend-sla Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: CRUD /accounts/:aid/slas

O backend SHALL expor CRUD em `/api/v1/accounts/:aid/slas`. Modelo: `{id, name, first_response_minutes, resolution_minutes, business_hours_only (bool), inbox_ids (int[]), label_ids (int[])}`. Uma política pode estar vinculada a múltiplos inboxes e labels.

#### Scenario: criar SLA com binding

- **WHEN** POST com `{name: "Premium", first_response_minutes: 60, inbox_ids: [1,2]}`
- **THEN** persiste na tabela `sla_policies` e cria linhas em `sla_bindings (sla_id, inbox_id)`

### Requirement: Tracking automático de SLA em conversas

Ao criar uma conversa, o backend MUST resolver a política SLA aplicável (por inbox → label → default) e persistir `sla_policy_id`, `sla_first_response_due_at`, `sla_resolution_due_at` em `conversations`. Asynq job periódico (1 min) detecta breach e emite evento `sla.breached`.

#### Scenario: conversa recebe SLA

- **WHEN** nova conversa em inbox com SLA "Premium" (60min)
- **THEN** `sla_first_response_due_at = created_at + 60min` é gravado

#### Scenario: breach detectado

- **WHEN** `now() > sla_first_response_due_at` sem resposta
- **THEN** conversa ganha flag `sla_breached=true`, job emite `sla.breached` no realtime e persiste notification

### Requirement: GET /accounts/:aid/reports/sla

O backend SHALL expor `GET /reports/sla?from=&to=` agregando: total de conversas SLA no período, cumpridas, em risco, breached, por política.

#### Scenario: relatório mensal

- **WHEN** `GET /reports/sla?from=2026-04-01&to=2026-04-30`
- **THEN** retorna `{total, met, breached, by_policy: [...]}`

