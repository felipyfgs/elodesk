## 1. Backend — Remoção de código morto

- [ ] 1.1 Remover `realtime.NewHub()` e `go hub.Run()` de `cmd/worker/main.go`
- [ ] 1.2 Remover import `backend/internal/realtime` de `cmd/worker/main.go`
- [ ] 1.3 Substituir `_ = referencedMessageID` em `channel/tiktok/send.go` por TODO documentado
- [ ] 1.4 Rodar `make lint` e `make test` no backend

## 2. Frontend — Limpeza de logging

- [ ] 2.1 Substituir `console.error` em catch blocks por `useErrorHandler().handle()` nos 9 arquivos identificados
- [ ] 2.2 Adicionar guard `import.meta.dev` nos `console.warn` remanescentes
- [ ] 2.3 Rodar `pnpm lint` e `pnpm typecheck`

## 3. Frontend — Deduplicação do emoji picker

- [ ] 3.1 Criar composable `frontend/app/composables/useEmojiPicker.ts`
- [ ] 3.2 Migrar `ComposerToolbar.vue` para usar `useEmojiPicker`
- [ ] 3.3 Migrar `SendMessageEmojiButton.vue` para usar `useEmojiPicker`
- [ ] 3.4 Rodar `pnpm typecheck` e verificar emoji picker nos dois pontos de uso

## 4. Frontend — Deduplicação do formulário de contato

- [ ] 4.1 Criar componente `ContactFormFields.vue` com campos compartilhados
- [ ] 4.2 Migrar `EditForm.vue` para usar `ContactFormFields`
- [ ] 4.3 Migrar `AddModal.vue` para usar `ContactFormFields`
- [ ] 4.4 Rodar `pnpm lint` e `pnpm typecheck`

## 5. Verificação final

- [ ] 5.1 Rodar `make lint` no backend (deve passar sem novos erros)
- [ ] 5.2 Rodar `pnpm lint && pnpm typecheck` no frontend (deve passar sem erros)
- [ ] 5.3 Rodar `make test` no backend (todos os testes passam)
