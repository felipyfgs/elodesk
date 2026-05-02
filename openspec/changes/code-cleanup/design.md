## Context

O codebase acumulou artefatos residuais ao longo do desenvolvimento: goroutine ociosa no worker, stub incompleto no canal TikTok, logs de console não condicionais no frontend e duplicações de lógica em componentes Vue. Todos são de baixo risco e escopo bem definido.

O worker asynq (`cmd/worker/main.go`) inicializa um `realtime.Hub` e dispara `go hub.Run()` mas nunca o passa para nenhum processor — o worker não tem servidor HTTP e não transmite eventos realtime. Isso consome uma goroutine e conexão Redis desnecessárias em produção.

## Goals / Non-Goals

**Goals:**
- Remover `realtime.Hub` + goroutine do worker (3 linhas + import)
- Transformar stub `_ = referencedMessageID` do TikTok em TODO documentado
- Consolidar `console.error`/`console.warn` sem guard em blocos catch via `useErrorHandler` ou `import.meta.dev`
- Extrair lógica de emoji picker para um composable `useEmojiPicker`
- Extrair formulário base de contato para componente `ContactFormBase.vue`

**Non-Goals:**
- Não alterar migrations existentes (forward-only, não podem ser deletadas)
- Não modificar o canal de email `channel/email/` (half-baked por design, documentado no AGENTS.md)
- Não unificar as 7 páginas `[...path].vue` de redirect — cada diretório do Nuxt precisa do seu próprio catch-all; tentar consolidar quebraria o roteamento

## Decisions

### 1. Worker: remover Hub completamente

**Decisão**: Deletar `hub := realtime.NewHub()` e `go hub.Run()` de `cmd/worker/main.go`, e remover o import `backend/internal/realtime`.

**Alternativa considerada**: Passar o hub para os processors caso precisem broadcastar eventos no futuro. Rejeitada — o worker já tem acesso ao Redis (asynq) e pode usar o `realtime.Hub` do servidor HTTP, que é o único ponto de broadcast. Se no futuro o worker precisar emitir eventos realtime, pode-se adicionar via Redis pub/sub, não criando outro hub.

### 2. TikTok: converter stub em TODO

**Decisão**: Substituir `_ = referencedMessageID // placeholder...` por um comentário TODO claro indicando que a feature está pendente, mantendo o `_` para evitar unused variable error.

**Alternativa considerada**: Implementar `referencedMessageID`. Rejeitada — a feature `FEATURE_CHANNEL_TIKTOK` está desabilitada por padrão e o TikTok não expõe `referenced_message_info` na API de mensagens atual. A implementação requer mudanças no modelo de mensagens.

### 3. Frontend: rotear console.* para useErrorHandler

**Decisão**: Em blocos `catch`, substituir `console.error(...)` por `useErrorHandler().handle(error, context)`. Em `console.warn`, adicionar guard `import.meta.dev` se for aviso de desenvolvimento.

**Alternativa considerada**: Adicionar `import.meta.dev` em todos. Rejeitada — `console.error` em catch blocks deve logar em produção também, mas via canal estruturado (`useErrorHandler`) em vez de `console.error` direto.

### 4. Emoji picker: extrair composable

**Decisão**: Criar `frontend/app/composables/useEmojiPicker.ts` com:
- Interface `EmojiPickerSelectEvent`
- Computeds `emojiTheme` e `emojiGroupNames`
- Função `onEmojiSelect`
Ambos `ComposerToolbar.vue` e `SendMessageEmojiButton.vue` passam a usar o composable.

### 5. Contact form: extrair componente base

**Decisão**: Criar `ContactFormFields.vue` com os campos compartilhados (name, email, phone, country, avatar, custom_attrs). `EditForm.vue` e `AddModal.vue` usam o componente base via slots para as diferenças (botão submit, título, etc).

## Risks / Trade-offs

- **[Baixo] Worker não compila**: Se `realtime` for usado em outro lugar de `cmd/worker/main.go` além das linhas 18, 50-51. → Verificar antes de deletar.
- **[Baixo] Regressão no emoji picker**: Se o composable não capturar toda a lógica de ambos os componentes. → Testar em dev com ambos os pontos de uso.
- **[Baixo] Formulário de contato quebra validação**: Se `AddModal` e `EditForm` tiverem schemas Zod diferentes. → O componente base deve receber schema como prop ou ser agnóstico a validação.
