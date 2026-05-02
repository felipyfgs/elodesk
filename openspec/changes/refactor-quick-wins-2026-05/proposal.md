## Why

A auditoria sistemática registrada em [`openspec/changes/codebase-refactor-audit/atlas/`](../codebase-refactor-audit/atlas/) (capturada em 2026-05-01, commit `1bbf9cc`) identificou cinco clusters de refatoração que combinam **alto impacto, baixo risco e baixo esforço** — todos com `Risco = 1` na pontuação tridimensional do atlas. Atacá-los em uma única proposta guarda-chuva produz ~700 linhas de código deletado, ~20 ocorrências de duplicação eliminadas e legibilidade plana em 10 callsites mecânicos, sem tocar fluxos críticos (auth, realtime, persistence). É o ponto de partida natural pós-auditoria: dá tração visível, livra revisão futura de ruído acumulado, e deixa o terreno limpo para os clusters de risco médio/alto (ContactInbox, channel provisioning, repo splits) que merecem propostas próprias.

## What Changes

Cada item desta change cobre um cluster do atlas. Implementações são independentes e podem mergear separadamente, mas vivem na mesma proposta porque compartilham natureza (low-risk, high-value).

- **[C01] Remover dead code frontend** — deletar 12 componentes Vue + 2 composables sem nenhum referenciador (verificado contra naming convention auto-import do Nuxt 4): `components/contacts/{MergeModal,FilterSidebar,ListLayout}.vue`, `components/conversations/MentionPicker.vue`, `components/auth/mfa/{QrCode,StatusCard,RecoveryCodes}.vue`, `components/auth/AuthFooterLinks.vue`, `components/inboxes/settings/{AgentsPicker,SecretField}.vue`, `components/settings/integrations/WebhookTestButton.vue`, `components/settings/agents/BadgeList.vue`, `composables/{useAccountRoute,useAccountUrl}.ts`. Cobre achados F01–F12.

- **[C05] Mover dados de `countries.ts` para JSON** — extrair o array de 1463 linhas para `frontend/app/utils/countries.data.json`; reduzir `frontend/app/utils/countries.ts` para ≤10 linhas (import JSON + export tipo `Country` + export `countries`). Os 3 importadores (`PhoneNumberInput.vue`, `contacts/AddModal.vue`, `contacts/EditForm.vue`) não precisam mudar. Cobre achado F46.

- **[C08] Refatorar `else`-after-return em 10 callsites** — converter padrão `if cond { return ... } else { ... }` para guard clause em: `handler/user_access_token.go:47`, `service/conversation.go:289`, `service/forward.go:147`, `service/forward.go:285`, `service/custom_attribute.go:235`, `service/custom_attribute.go:271`, `service/agent.go:143`, `repo/contact.go:356`, `repo/participant.go:148`, `config/config.go:140`. Cobre achado F27.

- **[C10] Consolidar emit de audit em `LabelService`** — extrair helper `emitLabelAudit(ctx, taggableType, taggableID, action, label)` chamado por `Apply` e `Remove` (`backend/internal/service/label.go:107-161`). Elimina ~25 linhas duplicadas. Cobre achado F32.

- **[C03] Extrair helpers em `CustomAttributeService`** — criar `targetForEntity(entityType) (table string, repo entityRepo)` e `mergeAdditionalAttrs(existing *string, updates map[string]any) (string, error)` em `backend/internal/service/custom_attribute.go`. Substituir 4 ocorrências do switch `if isContact` (linhas 207, 235, 271, 298) e 2 ocorrências do merge (linhas 222, 242). Cobre achados F31, F33.

- Adicionar nota nos 5 itens correspondentes em [`priorities.md`](../codebase-refactor-audit/atlas/priorities.md) seção "Proposed follow-up changes" linkando para esta change.

## Capabilities

### New Capabilities
<!-- Nenhuma. Esta change é puramente refatoração — não introduz nova capability funcional. -->

### Modified Capabilities
<!-- Nenhuma. Os comportamentos observáveis (request/response, audit log structure, persistência) ficam idênticos.
     A change é refatoração estrita: melhora forma sem mudar função. -->

## Impact

- **Backend**: edita `backend/internal/service/{label,custom_attribute,conversation,forward,agent}.go`, `backend/internal/handler/user_access_token.go`, `backend/internal/repo/{contact,participant}.go`, `backend/internal/config/config.go`. Sem mudanças de API, DTO ou comportamento. Sem migrations.
- **Frontend**: deleta 14 arquivos (12 `.vue` + 2 `.ts`); altera 1 (`utils/countries.ts`); cria 1 (`utils/countries.data.json`). Sem mudanças em pages, stores, ou rotas.
- **Tests**: backend tem zero testes nos arquivos tocados (registrado como `missing-tests` em F47/F48 do atlas). Esta change não adiciona testes — manter escopo. Validação via `go vet ./...`, `golangci-lint`, e teste manual dos fluxos audit/custom-attr/conversa.
- **Linters**: backend `make lint` deve continuar reportando 0 issues. Frontend `pnpm lint` requer `nuxt prepare` rodando — fora do escopo.
- **Bundle frontend**: redução de ~700 linhas de componentes não usados + extração de dados de `countries.ts` para JSON. Tree-shake já elimina componentes não importados, mas a remoção física simplifica grep/diff/IDE.
- **Atlas link de retorno**: ao mergear, atualizar `priorities.md` com referência para `openspec/changes/refactor-quick-wins-2026-05/` na seção "Proposed follow-up changes".

## Achados do atlas cobertos

Cluster → IDs em `findings.md`:

| Cluster | Achados | Score do cluster |
|---|---|---:|
| C01 Frontend dead code | F01, F02, F03, F04, F05, F06, F07, F08, F09, F10, F11, F12 | 14.00 |
| C03 Custom attribute helpers | F31, F33 | 4.00 |
| C05 Countries data extraction | F46 | 2.00 |
| C08 Guard clauses | F27 | 1.00 |
| C10 Label audit emit | F32 | 1.00 |
| **Total** | **18 achados** | **22.00** |
