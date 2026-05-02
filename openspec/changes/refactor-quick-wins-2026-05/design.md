## Context

Esta change implementa cinco clusters de refatoração identificados por `openspec/changes/codebase-refactor-audit/atlas/priorities.md` (C01, C03, C05, C08, C10). O atlas justifica cada um com achados verificados em `findings.md` (F01–F12, F27, F31, F32, F33, F46) e atribuiu pontuação de risco baixo a todos. A motivação não é arquitetural — é higiene: tirar do caminho aquilo que polui revisão e grep, antes de atacar refatorações de risco médio/alto (ContactInbox creation, channel provisioning, repo splits).

Não há nova feature. Não há mudança de comportamento observável. Os 5 clusters são independentes e podem mergear em PRs separados se conveniente, mas vivem na mesma proposta porque compartilham natureza (refactor mecânico, baixo risco, ganho rápido).

## Goals / Non-Goals

**Goals:**
- Deletar 14 arquivos frontend identificados como dead code (12 componentes Vue + 2 composables).
- Mover dados de `countries.ts` para JSON, mantendo API pública (`countries`, `type Country`).
- Converter 10 padrões `else`-after-return para guard clauses.
- Extrair helper `emitLabelAudit()` em `LabelService` e substituir 2 callers.
- Extrair helpers `targetForEntity()` e `mergeAdditionalAttrs()` em `CustomAttributeService` e substituir 6 callers.
- Manter `golangci-lint` em 0 issues após cada cluster.

**Non-Goals:**
- Adicionar testes. Os arquivos backend tocados estão em pacotes com testes mínimos (atlas registra em F47/F48 como follow-up). Adicionar cobertura aqui inflaria a proposta — fica para `refactor-backend-test-coverage` futura.
- Tocar fluxos de risco médio/alto (ContactInbox, channel provisioning, conversation repo split). Eles têm propostas próprias planejadas.
- Mudar comportamento. Nenhum cluster altera o que o sistema faz — só como o código está escrito.
- Renomear símbolos para "padronizar" além do necessário. Cada cluster faz exatamente o que está descrito; cleanup adicional é distração.

## Decisions

### D1: Uma proposta com cinco clusters em vez de cinco propostas

Os 5 clusters compartilham: risco 1, esforço 1, sem mudança de comportamento, sem dependência mútua. Empacotar em uma só proposta evita 5× a sobrecarga OpenSpec (proposal+design+specs+tasks×5) sem perder rastreabilidade — `tasks.md` separa cada cluster em grupo numerado.

**Alternativa considerada:** uma proposta por cluster (C01, C03, C05, C08, C10 separadas). Rejeitada porque o overhead de 5 propostas para refatorações de <1 dia cada é desproporcional, e revisão fica fragmentada quando o tema é coeso ("primeira rodada de cleanup pós-auditoria").

### D2: Dead code é deletado, não comentado nem deprecated

Os 14 arquivos têm zero referências verificadas (cross-check com naming convention auto-import do Nuxt 4). Não há razão para manter arquivo morto comentado ou marcado `@deprecated` — git history preserva tudo.

**Alternativa considerada:** mover para `frontend/app/_archive/` antes de deletar. Rejeitada porque adiciona complexidade sem benefício real; se algum desses componentes for útil de novo, `git log --diff-filter=D --all -- '<path>'` recupera.

### D3: `countries.ts` mantém API pública estável

`countries.ts` continua exportando `countries: Country[]` e `type Country`. Internamente, importa `countries.data.json` via `import countriesData from './countries.data.json'`. Os 3 importadores (`PhoneNumberInput.vue`, `contacts/AddModal.vue`, `contacts/EditForm.vue`) não precisam de mudança.

**Alternativa considerada:** trocar todos os 3 importadores para `import countries from '~/utils/countries.data.json'` direto e deletar `countries.ts`. Rejeitada porque perde o tipo `Country` (precisaria de `types/country.ts` separado), aumenta diff, e não traz ganho real.

### D4: Helpers em `CustomAttributeService` são privados ao pacote, não exportados

`targetForEntity()` e `mergeAdditionalAttrs()` ficam em `backend/internal/service/custom_attribute.go` como funções não exportadas (lowercase). Não exportar evita que outros pacotes adquiram dependência prematura.

**Alternativa considerada:** mover para `backend/internal/service/helpers.go` exportadas. Rejeitada por princípio de mínima superfície — só `CustomAttributeService` precisa hoje.

### D5: Guard-clause refactor é mecânico, sem reordenação adicional

Cada um dos 10 callsites em F27 segue padrão `if cond { return ... } else { body }`. A conversão é literal: `if cond { return ... }; body`. Sem aproveitar para reorganizar o resto da função, nem mudar nomes de variáveis. Mantém revisão fácil — diff só mostra remoção do `else { ... }`.

**Alternativa considerada:** aproveitar para extrair sub-funções nas funções tocadas. Rejeitada — fora do escopo desta proposta; eleva risco sem precisar.

### D6: Cluster C03 (custom attribute helpers) é a parte mais delicada

`CustomAttributeService` lida com persistência de `additional_attributes` JSON em `contacts` e `conversations` — fluxo testável mas com gotcha em merge (não substituir, fundir). O helper `mergeAdditionalAttrs` precisa preservar a semântica atual: `nil existing` → marshal só de updates; `non-nil existing` → unmarshal + spread updates por cima + remarshal. Testar manualmente antes/depois com payload contendo chaves novas e chaves a sobrescrever.

**Mitigação:** PR separado dentro da change só para C03 (commit isolado), com checklist no body do PR: (1) chave nova adicionada; (2) chave existente sobrescrita; (3) attrs nulo no contact; (4) JSON malformed retorna erro.

## Risks / Trade-offs

- **[Componente "morto" tem uso oculto que grep não pegou]** → Mitigação: rodar `pnpm dev` localmente após deleção e navegar fluxos relacionados (settings/inboxes, contacts merge, MFA login). 14 deleções são poucas — verificação visual cabe em ~30min.
- **[`countries.data.json` muda comportamento de bundle/SSR]** → Mitigação: Nuxt 4 está em `ssr: false` (CLAUDE.md confirma); JSON import via Vite é tratado como módulo estático, equivalente a array literal em runtime. Sem diferença observável.
- **[Helper `mergeAdditionalAttrs` reintroduz bug em casos de borda]** → Mitigação: D6. Comparar saída antes/depois com mesmo payload em 4 cenários antes de mergear.
- **[Guard-clause refactor é tedioso de revisar em diff]** → Mitigação: 10 callsites não estão no mesmo arquivo. PR separado para guard-clauses (commit C08) facilita revisão por arquivo. Cada hunk é 5-15 linhas.
- **[Atlas envelhece durante a implementação desta change]** → Mitigação: implementar nas próximas ~2 semanas. Atlas declara TTL de 30 dias / 10 PRs no escopo. Se a janela passar, atualizar `priorities.md` antes de mergear.

## Migration Plan

Sem migration de runtime. "Deploy" = merge.

Recomendação para minimizar risco de revisão:
1. Mergear cluster por cluster (5 PRs) ou commit por cluster dentro de PR único.
2. Ordem sugerida (mais barato primeiro): **C08** (guard clauses) → **C01** (dead code) → **C05** (countries data) → **C10** (label audit) → **C03** (custom attribute helpers, último porque é o mais delicado).

## Open Questions

- **Componentes dead code que parecem ter sido planejados mas nunca foram cabeados (ex.: `auth/mfa/{QrCode,StatusCard,RecoveryCodes}`):** vale capturar em ADR/note antes de deletar, caso o roadmap MFA esteja parado mas planejado retomar? Decisão pragmática: não. Se retomarmos, escrevemos do zero ou recuperamos via `git log`. Ficar com código não cabeado por hipótese é justamente o que a auditoria veio combater.
- **Linker frontend (`pnpm lint`) está bloqueado por falta de `.nuxt/`:** registrado como limitação em `findings.md`. Esta change não corrige; permanece como nota de processo. Re-rodar quando ambiente tiver `nuxt prepare` ok.
