## Why

Auditoria de "lixo" no codebase encontrou código morto, stubs, duplicações e side effects indesejados acumulados ao longo do desenvolvimento. Remover esses artefatos reduz superfície de bugs, elimina goroutines ociosas em produção e melhora a manutenibilidade.

## What Changes

- **Remover `realtime.Hub` do worker**: `cmd/worker/main.go` cria um hub e goroutine que nunca são usados (worker não tem servidor HTTP).
- **Limpar stub do TikTok**: `channel/tiktok/send.go:24` descarta `referencedMessageID` com `_` — substituir por TODO documentado ou implementar.
- **Consolidar `console.*` no frontend**: 9 chamadas de `console.error`/`console.warn` sem guard `import.meta.dev` — rotear via `useErrorHandler`.
- **Deduplicar lógica de emoji picker**: `ComposerToolbar.vue` e `SendMessageEmojiButton.vue` compartilham ~40 linhas idênticas de emoji picker.
- **Deduplicar formulário de contato**: `EditForm.vue` e `AddModal.vue` têm ~70% de sobreposição.
- **Documentar migrations wzap órfãs**: Migrations 0027-0030 criam e dropam tabelas/colunas wzap cujo código Go já foi removido.

## Capabilities

### New Capabilities

- `code-dead-removal`: Remoção de goroutine ociosa no worker e stub de TikTok
- `frontend-logging-cleanup`: Consolidação de console.* via useErrorHandler
- `frontend-dedup-emoji`: Extração de lógica compartilhada de emoji picker
- `frontend-dedup-contact-form`: Extração de formulário base de contato

### Modified Capabilities

<!-- Nenhuma capability existente tem seus requisitos alterados -->

## Impact

- **Backend**: `cmd/worker/main.go` (remoção de 3 linhas + import), `channel/tiktok/send.go` (1 linha)
- **Frontend**: `ComposerToolbar.vue`, `SendMessageEmojiButton.vue` (+1 componente novo), `EditForm.vue`, `AddModal.vue` (+1 componente base), 9 arquivos com `console.*` direto
- **Migrations**: somente documentação (não é possível deletar migrations forward-only)
