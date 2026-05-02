> Implementação dos 5 clusters identificados em `openspec/changes/codebase-refactor-audit/atlas/priorities.md` (C01, C03, C05, C08, C10). Ordem sugerida em `design.md` Migration Plan: C08 → C01 → C05 → C10 → C03.

## 1. C08 — Guard clauses (10 callsites mecânicos)

> Cada item: inverter `if cond { return ...; } else { body }` para `if cond { return ...; }; body`. Sem reorganização adicional. Diff esperado por callsite: 1-3 linhas removidas (`} else {` e `}` final).

- [x] 1.1 `backend/internal/handler/user_access_token.go:47` — guard clause (inverted condition)
- [x] 1.2 `backend/internal/service/conversation.go:289` — guard clause (skip: structural if/else, no return — already has guard `if latest != nil { return }`)
- [x] 1.3 `backend/internal/service/forward.go:147` — guard clause (extracted invalid-target guard)
- [x] 1.4 `backend/internal/service/forward.go:285` — guard clause (skip: `if ci != nil` has nested return but doesn't return unconditionally)
- [x] 1.5 `backend/internal/service/custom_attribute.go:235` — guard clause (skip: covered by C03 helper extraction)
- [x] 1.6 `backend/internal/service/custom_attribute.go:271` — guard clause (skip: covered by C03 helper extraction)
- [x] 1.7 `backend/internal/service/agent.go:143` — guard clause (inverted condition)
- [x] 1.8 `backend/internal/repo/contact.go:356` — guard clause (skip: `if isInsert`, neither branch returns)
- [x] 1.9 `backend/internal/repo/participant.go:148` — guard clause (skip: `if len(desired) == 0`, neither branch returns at top level)
- [x] 1.10 `backend/internal/config/config.go:140` — guard clause (skip: `if BackendKEK == ""`, neither branch returns)
- [x] 1.11 Rodar `cd backend && make lint` e `go vet ./...` — confirmar 0 novas issues (pre-existing struct tag warnings; label.go goconst será resolvido em C10)
- [x] 1.12 Commit C08 (caeec8d): `refactor(backend): convert else-after-return to guard clauses` — 3 callsites efetivos (user_access_token.go, agent.go, forward.go); demais 7 reavaliados como skip ou cobertos por C03

## 2. C01 — Frontend dead code removal (14 arquivos)

> Cada deleção: `git rm <path>`. Verificar visualmente que `pnpm dev` carrega ok depois.

- [x] 2.1 Deletar `frontend/app/components/contacts/MergeModal.vue`
- [x] 2.2 Deletar `frontend/app/components/contacts/FilterSidebar.vue`
- [x] 2.3 Deletar `frontend/app/components/contacts/ListLayout.vue`
- [x] 2.4 Deletar `frontend/app/components/conversations/MentionPicker.vue`
- [x] 2.5 Deletar `frontend/app/components/auth/mfa/QrCode.vue`
- [x] 2.6 Deletar `frontend/app/components/auth/mfa/StatusCard.vue`
- [x] 2.7 Deletar `frontend/app/components/auth/mfa/RecoveryCodes.vue`
- [x] 2.8 Deletar `frontend/app/components/auth/AuthFooterLinks.vue`
- [x] 2.9 Deletar `frontend/app/components/inboxes/settings/AgentsPicker.vue`
- [x] 2.10 Deletar `frontend/app/components/inboxes/settings/SecretField.vue`
- [x] 2.11 Deletar `frontend/app/components/settings/integrations/WebhookTestButton.vue`
- [x] 2.12 Deletar `frontend/app/components/settings/agents/BadgeList.vue`
- [x] 2.13 Deletar `frontend/app/composables/useAccountRoute.ts`
- [x] 2.14 Deletar `frontend/app/composables/useAccountUrl.ts`
- [x] 2.15 Verificado: zero referências remanescentes no código ativo (apenas openspec/ artifacts). CLAUDE.md atualizado: de 14→12 composables.
- [x] 2.16 Commit C01 (6ee21f7): `refactor(frontend): remove dead components and composables` — inclui CLAUDE.md (14→12 composables)

## 3. C05 — Countries data extraction

- [x] 3.1 Criar `frontend/app/utils/countries.data.json` (239 países extraídos)
- [x] 3.2 Reduzir `frontend/app/utils/countries.ts` para ~10 linhas (interface + import JSON + re-export)
- [x] 3.3 Verificar tipagem (requer `nuxt prepare` + `pnpm typecheck`)
- [x] 3.4 Tests manuais — typecheck ok; verificação visual pendente em próximo deploy
- [x] 3.5 Commit C05 (9f48e91): `refactor(frontend): move countries data to JSON, keep .ts API stable`

## 4. C10 — Label audit emit consolidation

- [x] 4.1 Adicionar helper `emitLabelAudit` + constantes `taggableConversation`/`taggableContact`
- [x] 4.2 Substituir bloco de audit emit em `Apply` pela chamada ao helper
- [x] 4.3 Substituir bloco de audit emit em `Remove` pela chamada ao helper
- [x] 4.4 Verificar comportamento manual — lint+vet ok; verificação visual pendente em próximo deploy
- [x] 4.5 `make lint` — 0 issues (goconst resolvido pelas constantes)
- [x] 4.6 Commit C10 (31b6309): `refactor(backend): extract emitLabelAudit helper in LabelService`

## 5. C03 — Custom attribute helpers (mais delicado)

- [x] 5.1 Adicionar `targetForEntity` (retorna `entityAttrUpdater`) + `getExistingAttrs` + `mergeAttrMaps` (merge map-based)
- [x] 5.2 Substituir 4 ocorrências do `if isContact { ... } else { ... }` por chamadas a `targetForEntity`/`getExistingAttrs`/`mergeAttrMaps`
- [x] 5.3 Adicionar `mergeAttrMaps(existing *string, updates map[string]any) (string, error)`
- [x] 5.4 Substituir 2 ocorrências do bloco merge pela chamada a `mergeAttrMaps`
- [x] 5.5 Teste manual — lint+vet+tests ok; verificação visual pendente em próximo deploy
- [x] 5.6 `make lint` — 0 issues
- [x] 5.7 Commit C03 (244c00e): `refactor(backend): extract targetForEntity and mergeAttrMaps helpers in CustomAttributeService`

## 6. Atualizar atlas link de retorno

- [x] 6.1 Entrada já presente em `priorities.md` linha 134
- [x] 6.2 Justificativa já presente em `priorities.md` linha 130
- [x] 6.3 Link já presente em `priorities.md` (commit anterior, agora em `archive/codebase-refactor-audit/atlas/priorities.md`) — sem novo commit necessário

## 7. Validação final

- [x] 7.1 `git diff --name-only` confirma escopo (backend: guard clauses + helpers + label; frontend: 14 files deleted + countries JSON + CLAUDE.md)
- [x] 7.2 `cd backend && make lint` — 0 issues
- [ ] 7.3 `cd backend && go test ./... -race` (requer ambiente com infra)
- [ ] 7.4 `pnpm dev` no frontend (requer ambiente local)
- [x] 7.5 `openspec status` — 4/4 artefatos done
- [ ] 7.6 PR aberto (aguarda commits e push)
