## ADDED Requirements

### Requirement: Atlas registra esta change na seção "Proposed follow-up changes"

Ao concluir esta change, `openspec/changes/codebase-refactor-audit/atlas/priorities.md` SHALL ter uma entrada na seção "Proposed follow-up changes" com nome desta change e link relativo, declarando quais clusters do atlas (C01, C03, C05, C08, C10) ela cobre. Este requirement materializa a regra "Geração obrigatória de proposta(s) de refatoração ao concluir a auditoria" da capability `refactor-audit`.

#### Scenario: Atlas linka para esta change após merge
- **WHEN** esta change tiver tasks completas e merge aprovado
- **THEN** `openspec/changes/codebase-refactor-audit/atlas/priorities.md` contém na seção "Proposed follow-up changes" uma entrada `refactor-quick-wins-2026-05` com link relativo `openspec/changes/refactor-quick-wins-2026-05/`
- **AND** a entrada lista os clusters cobertos: C01, C03, C05, C08, C10
- **AND** a entrada lista os IDs dos achados cobertos: F01–F12, F27, F31, F32, F33, F46

### Requirement: Refatoração preserva comportamento observável

Esta change SHALL não alterar nenhuma resposta de API, payload de audit log, evento realtime, ou persistência. Apenas representação interna (organização de arquivos, helpers, guard clauses) muda.

#### Scenario: Resposta de endpoints inalterada
- **WHEN** um cliente chama qualquer endpoint backend tocado pela change (ex.: `POST /labels`, `POST /custom-attributes`, `POST /conversations`, etc.)
- **THEN** a resposta JSON tem estrutura e valores idênticos ao comportamento pré-change para o mesmo input

#### Scenario: Audit log preserva schema
- **WHEN** uma label é aplicada ou removida via `LabelService.Apply`/`Remove` após o refactor
- **THEN** o registro em `audit_logs` tem mesma `action`, `entity_type`, `entity_id`, `payload`, `actor_id` que tinha antes do refactor

#### Scenario: Custom attributes preservam semântica de merge
- **WHEN** `CustomAttributeService.SetAttributes` é chamado com `existing` não-nulo e `updates` contendo chave nova + chave a sobrescrever
- **THEN** o JSON resultante mantém todas as chaves de `existing` que não estão em `updates`, sobrescreve as que estão, e adiciona as novas — comportamento idêntico ao pré-refactor

### Requirement: Linter limpo após cada cluster

Após cada cluster ser implementado, `make lint` no backend SHALL continuar reportando `0 issues`. Esta requirement protege contra regressão de qualidade introduzida pelos refactors.

#### Scenario: golangci-lint limpo após C03 (helpers extraction)
- **WHEN** o cluster C03 (custom attribute helpers) é aplicado e os 6 callsites convertidos para usar os novos helpers
- **THEN** `cd backend && make lint` retorna `0 issues`
- **AND** `cd backend && go vet ./...` retorna sem warnings

#### Scenario: golangci-lint limpo após C08 (guard clauses)
- **WHEN** o cluster C08 (10 guard-clause refactors) é aplicado
- **THEN** `cd backend && make lint` retorna `0 issues`

### Requirement: Diff da change é restrito ao escopo declarado

Esta change SHALL tocar apenas os arquivos enumerados em `proposal.md` "Impact". Mudanças adicionais SHALL ser rejeitadas em revisão de PR ou movidas para change própria.

#### Scenario: PR final só toca arquivos declarados
- **WHEN** o PR final desta change é aberto
- **THEN** `git diff --name-only main...HEAD` mostra apenas:
  - 14 arquivos deletados em `frontend/app/components/**` e `frontend/app/composables/**`
  - `frontend/app/utils/countries.ts` modificado
  - `frontend/app/utils/countries.data.json` adicionado
  - 10 arquivos backend em `service/`, `handler/`, `repo/`, `config/` modificados (lista exata em `proposal.md`)
  - `openspec/changes/refactor-quick-wins-2026-05/` (artefatos)
  - `openspec/changes/codebase-refactor-audit/atlas/priorities.md` atualizado com link de retorno
- **AND** nenhum outro arquivo aparece no diff
