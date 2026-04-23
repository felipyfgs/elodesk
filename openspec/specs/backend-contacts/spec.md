# backend-contacts Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: Endpoint dedicado GET /accounts/:aid/contacts

O backend SHALL expor `GET /api/v1/accounts/:aid/contacts` com query params `search, labels, page, pageSize` (default page=1, pageSize=25, max=100). O handler MUST usar `contact_repo.List` com filtros server-side e scope por `account_id`. Resposta: `{data: Contact[], pagination: {page, pageSize, total}}`.

#### Scenario: busca por nome parcial

- **WHEN** `GET /contacts?search=mar&page=1&pageSize=10`
- **THEN** retorna contatos cujo `name ILIKE '%mar%'` na conta autenticada, ordenados por `created_at DESC`

#### Scenario: filtro por múltiplos labels

- **WHEN** `GET /contacts?labels=vip,lead`
- **THEN** retorna contatos com pelo menos um dos labels via JOIN em `label_taggings`

### Requirement: Endpoint POST /contacts/import (CSV)

O backend SHALL expor `POST /api/v1/accounts/:aid/contacts/import` aceitando multipart com arquivo CSV (max 10 MB). O parser MUST fazer streaming (não carregar tudo em memória), processar em batches de 500 usando `INSERT ... ON CONFLICT (account_id, email) DO UPDATE`. Resposta: `{inserted, updated, errors: [{line, reason}]}`.

#### Scenario: import válido

- **WHEN** CSV com 1000 linhas válidas é enviado
- **THEN** retorna `{inserted: 1000, updated: 0, errors: []}` em menos de 5s

#### Scenario: erros parciais

- **WHEN** CSV contém 3 linhas com email inválido
- **THEN** 997 linhas são persistidas, resposta inclui array de 3 erros com número de linha + motivo ("invalid email format")

### Requirement: Idempotência por (account_id, email)

A tabela `contacts` SHALL ter índice único em `(account_id, lower(email))`. Import MUST tratar conflito como update dos campos não vazios no CSV, preservando `custom_attributes` existentes não presentes no arquivo.

#### Scenario: re-import do mesmo CSV

- **WHEN** mesmo arquivo é enviado duas vezes
- **THEN** primeiro import insere; segundo retorna `inserted: 0, updated: N` sem duplicar

