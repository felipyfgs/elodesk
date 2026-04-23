## Why

A tela de conversas ja usa alguns componentes `UChat*`, mas ainda apresenta inconsistencias visuais e comportamentais: composer grande mesmo sem anexos, fallback `--` no contato, preview lateral com texto de estado vazio incorreto e status de mensagens comparando enums numericos com strings. Esta mudanca organiza a refatoracao para transformar a thread de atendimento em uma experiencia coesa baseada em Nuxt UI 4, sem alterar contratos backend.

## What Changes

- Refatorar `frontend/app/components/conversations/ConversationThread.vue` para centralizar a adaptacao `Message -> UChatMessage`, com papeis, lados, variantes, avatares, anexos, eventos privados/deletados e status consistentes.
- Refatorar `frontend/app/components/conversations/ConversationComposer.vue` para usar `UChatPrompt`, `UChatPromptSubmit` e `UFileUpload` sem area vazia quando nao houver anexos, com `rows=1`, autoresize, contador de caracteres e estados de envio claros.
- Ajustar `frontend/app/components/conversations/ConversationsList.vue` para diferenciar "sem conversas" de "conversa sem mensagens", melhorando previews e estados vazios.
- Melhorar o header da conversa com fallback de contato mais informativo, status, inbox, `displayId`, etiquetas e acoes via Nuxt UI.
- Usar o contexto real do Nuxt UI v4.6.1 consultado via MCP: `Chat` como pagina-guia, `UChatMessages`/`UChatMessage`/`UChatPrompt`/`UChatPromptSubmit` na thread principal, e `UChatPalette`/`UChatReasoning`/`UChatTool`/`UChatShimmer` apenas quando sua semantica real for adequada.
- Incorporar contexto de UX explorado em `_refs/chatwoot`: view model de mensagem com `bubbleKind`, orientacao, agrupamento visual, status/erro, anexos, separador de nao lidas quando houver marcador confiavel e composer como area de trabalho compacta.
- Manter o layout `UDashboardPanel`/`UDashboardNavbar` existente, usando componentes Nuxt UI e cores semanticas sempre que houver alteracao visual.
- Adicionar chaves i18n necessarias em `frontend/i18n/locales/pt-BR.json` e `frontend/i18n/locales/en.json`.

## Nao-objetivos

- Nao alterar endpoints, payloads ou regras de negocio do backend.
- Nao adicionar assistente de IA, Vercel AI SDK, `UChatReasoning` ou `UChatTool` nesta etapa.
- Nao tratar `Chat` como componente importavel nem adicionar `ai`, `@ai-sdk/vue` ou Comark ao frontend.
- Nao buscar paridade completa com Chatwoot nesta change: email quoted reply, audio recorder, dashboard apps, context menu completo de mensagem e sidebar detalhada de contato ficam para mudancas futuras.
- Nao reescrever filtros, bulk actions ou rotas scoped de conversas fora do necessario para corrigir integracao visual.
- Nao introduzir novas dependencias alem das ja presentes em `frontend/package.json`.

## Capabilities

### New Capabilities

- Nenhuma.

### Modified Capabilities

- `conversations-ui`: refinar os requisitos da thread, composer e lista de conversas para usar os componentes de chat do Nuxt UI de forma consistente e corrigir estados visuais observados.

## Impact

- Arquivos principais: `frontend/app/components/conversations/ConversationThread.vue`, `ConversationComposer.vue`, `ConversationsList.vue`, `frontend/app/stores/messages.ts`, `frontend/app/stores/conversations.ts`.
- i18n: `frontend/i18n/locales/pt-BR.json` e `frontend/i18n/locales/en.json`.
- Testes/verificacao: `pnpm lint`, `pnpm typecheck` e validacao visual manual ou Playwright da tela `/accounts/:accountId/conversations`.

## Riscos e mitigacoes

| Risco | Mitigacao |
|---|---|
| Quebrar envio de mensagens com anexos | Preservar endpoint atual e testar envio com texto, arquivo e texto+arquivo |
| Regressao no realtime de mensagens | Manter stores Pinia e `useConversationRealtime` como fonte de atualizacao |
| Nuxt UI gerar layout inesperado no mobile | Validar desktop e `USlideover` mobile com conversas vazias, curtas e longas |
| Escopo crescer tentando reproduzir todo o Chatwoot | Limitar esta refatoracao a presentation model, composer compacto, status/anexos, estados vazios e lista operacional |
