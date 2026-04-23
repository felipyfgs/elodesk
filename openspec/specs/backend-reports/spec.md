# backend-reports Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: GET /reports/overview

O backend SHALL expor `GET /api/v1/accounts/:aid/reports/overview?from=&to=` retornando `{open_count, resolved_count, first_response_avg_minutes, resolution_avg_minutes, csat_avg, volume_by_day: [{date, count}], status_breakdown: [{status, count}]}`. P95 da query MUST ser ≤500ms para datasets ≤100k conversations.

#### Scenario: overview últimos 7 dias

- **WHEN** `GET /reports/overview?from=2026-04-12&to=2026-04-19`
- **THEN** retorna métricas agregadas do período

### Requirement: GET /reports/conversations

O backend SHALL expor `GET /reports/conversations?from=&to=&inbox_id=&label_id=&sort=&page=` retornando tabela com `first_response_minutes, resolution_minutes, handling_minutes, reopened_count` por conversa. Sort support em qualquer coluna numérica.

#### Scenario: ordenar por resolução desc

- **WHEN** `GET /reports/conversations?sort=resolution_minutes:desc`
- **THEN** resultados ordenados corretamente com paginação cursor-based

### Requirement: GET /reports/:entity e /reports/:entity/:id

O backend SHALL expor para cada `entity` ∈ [agents, inboxes, teams, labels]: lista agregada + drill-down individual. Métricas comuns: conversas tratadas, first response médio, resolução média, CSAT.

#### Scenario: drill-down em agente

- **WHEN** `GET /reports/agents/42?from=&to=`
- **THEN** retorna timeline diária + métricas do agente 42 no período

### Requirement: Índices para performance

O backend SHALL criar migration com índices:
- `idx_conversations_account_status_created` em `(account_id, status, created_at DESC)`
- `idx_conversations_assignee_created` em `(assignee_id, created_at DESC)`
- `idx_messages_conversation_created` em `(conversation_id, created_at DESC)` (se não existir)

#### Scenario: EXPLAIN no report overview

- **WHEN** `EXPLAIN ANALYZE` é rodado na query de `/reports/overview` com 100k conversations
- **THEN** plano usa `idx_conversations_account_status_created` e tempo total ≤500ms

