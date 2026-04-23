## 1. Preparacao e tipos

- [ ] 1.1 Revisar payload real de `GET /accounts/:accountId/conversations/:conversationId/messages` para confirmar campos de anexos e status retornados.
- [x] 1.2 Atualizar `frontend/app/stores/messages.ts` com tipos auxiliares para anexos, se o payload ja expuser esses campos.
- [x] 1.3 Criar helpers locais ou composable para adaptar `Message` em view model de chat: papel, lado, variante, partes, anexos, horario e status.
- [x] 1.4 Mapear `MessageStatus` numerico (`0..3`) para icone, label i18n e cor semantica.
- [ ] 1.5 Confirmar durante a implementacao os slots/variantes reais nos temas gerados `frontend/.nuxt/ui/chat-*.ts` antes de customizar classes.
- [x] 1.6 Garantir que a refatoracao nao adicione dependencias de IA (`ai`, `@ai-sdk/vue`, `@comark/nuxt`) nem trate `Chat` como componente importavel.
- [x] 1.7 Mapear o contexto explorado no Chatwoot para helpers locais: `bubbleKind`, orientacao, agrupamento, unread divider, reply-to e composer slots.
- [ ] 1.8 Confirmar se existem campos confiaveis para primeira mensagem nao lida, ultimo visto do agente e `contentAttributes.inReplyTo`.

## 2. Thread com UChatMessages e UChatMessage

- [x] 2.1 Refatorar `frontend/app/components/conversations/ConversationThread.vue` para consumir o adaptador de mensagens em vez de misturar regra de negocio no template.
- [x] 2.2 Garantir que mensagens recebidas, enviadas e eventos de sistema renderizem com lado/variante consistentes usando `UChatMessage`.
- [x] 2.3 Usar `parts` em `UChatMessage` por meio de helper `messageParts`, sem usar o prop deprecado `content`.
- [ ] 2.4 Decidir localmente entre `UChatMessages` com slot `default` manual ou prop `messages`, mantendo `Message` de dominio fora da store no formato `UIMessage[]`.
- [x] 2.5 Criar `messageBubbleKind` para texto, anexo, privado, deletado, atividade, template, erro, unsupported e vazio.
- [x] 2.6 Preservar renderizacao de mensagens privadas e deletadas usando slots de `UChatMessage`.
- [x] 2.7 Exibir horario e status de envio como acoes/metadados da mensagem, com icones e cores semanticas.
- [x] 2.8 Renderizar anexos retornados pelo backend via slot `#files` ou `#content` customizado, sem esconder texto, horario ou status.
- [x] 2.9 Implementar agrupamento simples de mensagens consecutivas quando mesmo remetente, mesmo tipo, status nao falho e mesmo minuto.
- [x] 2.10 Adicionar separador de nao lidas quando houver marcador confiavel; se nao houver, documentar a limitacao e manter auto-scroll.
- [x] 2.11 Adicionar preview de reply-to quando `contentAttributes.inReplyTo` ou campo equivalente existir.
- [x] 2.12 Adicionar estado visual para thread sem mensagens dentro da area de `UChatMessages`.

## 3. Header da conversa

- [x] 3.1 Melhorar helpers de contato em `ConversationThread.vue` para evitar fallback `--` quando houver telefone, WhatsApp JID, email, `meta.sender.name` ou `displayId`.
- [x] 3.2 Exibir contato, identificador, inbox, `#displayId`, etiquetas e status usando componentes Nuxt UI e classes semanticas.
- [x] 3.3 Revisar acoes do header para manter `UDropdownMenu`, `UTooltip`, `UButton` e `UBadge` com labels acessiveis.

## 4. Composer com UChatPrompt

- [x] 4.1 Refatorar `frontend/app/components/conversations/ConversationComposer.vue` para `UChatPrompt` compacto com `rows=1`, `autoresize` e `maxrows`.
- [x] 4.2 Manter `UChatPromptSubmit` em `#trailing`, refletindo estado `ready`, `submitted` e `error` quando aplicavel.
- [x] 4.3 Nao ligar `@stop` ou `@reload` em `UChatPromptSubmit` sem semantica real de cancelar ou reenviar mensagem.
- [x] 4.4 Manter botao de anexo em `#leading` com `UTooltip` e `UButton`, sem renderizar dropzone vazia permanente.
- [x] 4.5 Usar `UFileUpload` para selecao/preview apenas quando houver arquivos ou interacao ativa de upload.
- [x] 4.6 Preservar upload assinado atual (`/accounts/:accountId/uploads/signed-url`) e envio para `/accounts/:accountId/conversations/:conversationId/messages`.
- [x] 4.7 Validar texto, texto+anexo, anexo sem texto, erro de upload, remocao de anexo e limite de caracteres.
- [ ] 4.8 Preparar os encaixes para triggers `/` e `@` sem bloquear a correcao visual inicial.
- [ ] 4.9 Reservar `#header` do `UChatPrompt` para reply-to, anexos e banners compactos de restricao quando esses estados existirem.
- [x] 4.10 Avaliar se nota privada pertence a esta change ou deve virar change propria usando `frontend/app/stores/notes.ts`.
- [x] 4.11 Documentar comportamento por canal quando texto+anexo precisar ser enviado como mensagens separadas.

## 5. Lista de conversas e i18n

- [x] 5.1 Ajustar `frontend/app/components/conversations/ConversationsList.vue` para usar texto especifico quando uma conversa nao tiver ultima mensagem.
- [x] 5.2 Adicionar chaves i18n em `frontend/i18n/locales/pt-BR.json` e `frontend/i18n/locales/en.json` para preview vazio, thread vazia e labels de status, se ausentes.
- [x] 5.3 Verificar que o estado vazio geral da lista continua usando `conversations.empty` apenas quando nao ha conversas.
- [ ] 5.4 Revisar icones de canal, unread badge e checkbox de selecao para nao causar sobreposicao ou quebra em larguras pequenas.
- [x] 5.5 Preservar densidade operacional inspirada no Chatwoot: canal/inbox, horario, preview, unread, labels, assignee quando disponivel e selecao por hover.

## 6. Qualidade e validacao

- [x] 6.1 Rodar `pnpm lint` em `frontend/` e corrigir problemas relacionados.
- [x] 6.2 Rodar `pnpm typecheck` em `frontend/` e corrigir problemas relacionados.
- [x] 6.3 Verificar que `UChatReasoning`, `UChatTool` e dependencias de AI SDK/Comark nao foram introduzidos na thread principal.
- [ ] 6.4 Validar manualmente `/accounts/:accountId/conversations` em desktop com conversa vazia, conversa curta, conversa longa, contato sem nome e conversa com anexos.
- [ ] 6.5 Validar o fluxo mobile via `USlideover`, incluindo abertura/fechamento da thread e composer compacto.
- [ ] 6.6 Validar checklist de UX inspirado no Chatwoot: orientacao, agrupamento, status, erro/retry, anexos, unread divider quando disponivel, composer compacto e lista densa.
- [ ] 6.7 Registrar capturas ou observacoes da validacao visual antes de concluir a implementacao.
