## ADDED Requirements

### Requirement: Thread de conversa usa componentes de chat Nuxt UI

`ConversationThread.vue` SHALL renderizar mensagens com `UChatMessages` e `UChatMessage`, usando uma camada de adaptacao entre `Message` do dominio e props/slots do Nuxt UI. A adaptacao MUST definir papel, lado, variante, avatar, partes de texto, anexos, eventos de sistema, estado privado/deletado, horario e status de envio sem alterar a store Pinia nem o contrato backend.

`UChatMessage` MUST receber `parts` validas, preferencialmente produzidas por helper local como `messageParts(message)`, e MUST NOT usar o prop deprecado `content`. A thread MAY usar o slot `default` de `UChatMessages` para renderizar manualmente cada `UChatMessage` enquanto `Message` nao for convertido para `UIMessage[]`.

A camada de adaptacao SHALL expor um view model de mensagem inspirado em inboxes de atendimento como Chatwoot, separando pelo menos `role`, `side`, `variant`, `bubbleKind`, `parts`, `attachments`, `statusMeta` e `grouping`.

#### Scenario: renderizacao de mensagens recebidas e enviadas

- **WHEN** uma conversa possui mensagens recebidas (`messageType=0`) e enviadas (`messageType=1`)
- **THEN** a thread mapeia recebidas para `role='assistant'` no lado do contato e enviadas para `role='user'` no lado do agente, com avatares, bolhas e acoes coerentes com Nuxt UI

#### Scenario: renderizacao de eventos de sistema

- **WHEN** uma mensagem possui tipo de atividade ou template (`messageType=2` ou `messageType=3`)
- **THEN** a thread mapeia a mensagem para `role='system'` e exibe o conteudo como evento discreto por slot customizado, sem bolha dominante de agente ou contato

#### Scenario: partes compativeis com UChatMessage

- **WHEN** uma mensagem textual normal e renderizada
- **THEN** o adaptador fornece `parts` no formato aceito por `UChatMessage`, como uma parte `{ type: 'text', text }`, sem depender do prop deprecado `content`

#### Scenario: status numerico da mensagem

- **WHEN** uma mensagem enviada possui `MessageStatus` numerico (`0`, `1`, `2` ou `3`)
- **THEN** a UI mapeia o codigo para icone, label i18n e cor semantica corretos, sem comparar o status com strings

#### Scenario: tipo visual da bolha

- **WHEN** uma mensagem e privada, deletada, atividade, template, falha, unsupported, texto ou anexo
- **THEN** o adaptador resolve `bubbleKind` e a UI usa slots de `UChatMessage` para renderizar a aparencia apropriada sem misturar regras no template principal

#### Scenario: agrupamento de mensagens consecutivas

- **WHEN** duas mensagens consecutivas tem mesmo remetente, mesmo `messageType`, status nao falho e foram criadas no mesmo minuto
- **THEN** a thread reduz avatar/metadados repetidos e ajusta o espacamento para parecer um grupo de mensagens

#### Scenario: agrupamento inseguro

- **WHEN** uma mensagem nao possui remetente ou timestamp confiavel
- **THEN** a thread renderiza a mensagem sem agrupamento para evitar atribuir visualmente a mensagem ao remetente errado

#### Scenario: separador de nao lidas disponivel

- **WHEN** a conversa possui um marcador confiavel de primeira mensagem nao lida ou ultimo visto do agente
- **THEN** `UChatMessages` exibe um separador discreto antes da primeira mensagem nao lida e preserva o auto-scroll sem pular contexto

### Requirement: Header da conversa possui fallback informativo

`ConversationThread.vue` SHALL exibir um header de contato baseado nos dados disponiveis da conversa. O nome MUST usar, em ordem, nome do contato, telefone, WhatsApp JID, email, `meta.sender.name` ou `#displayId`; o identificador MUST usar telefone, WhatsApp JID, email, inbox ou `displayId`. A UI MUST evitar renderizar apenas `--` quando houver qualquer dado alternativo.

#### Scenario: contato sem nome

- **WHEN** a conversa nao possui `contact.name`, mas possui telefone, WhatsApp JID ou `displayId`
- **THEN** o header exibe um identificador util em vez de `--`

#### Scenario: conversa com etiquetas e inbox

- **WHEN** a conversa possui `labels`, `inbox.name` e `displayId`
- **THEN** o header exibe esses dados com componentes Nuxt UI e classes semanticas

### Requirement: Lista diferencia estado vazio de preview vazio

`ConversationsList.vue` SHALL usar textos diferentes para lista sem conversas e conversa sem ultima mensagem. O estado vazio da lista MUST continuar usando `conversations.empty`; o preview de um item sem `lastNonActivityMessage` MUST usar uma chave especifica, como `conversations.message.emptyPreview`.

#### Scenario: lista sem conversas

- **WHEN** o filtro atual nao retorna conversas
- **THEN** a lista exibe o estado vazio geral com `conversations.empty`

#### Scenario: conversa existente sem ultima mensagem

- **WHEN** uma conversa existe mas nao possui `meta.lastNonActivityMessage`
- **THEN** o item da lista exibe um preview de conversa sem mensagens, sem dizer que nao existem conversas

### Requirement: Anexos aparecem na thread e no composer

A UI de conversas SHALL preservar o fluxo atual de upload assinado e SHALL exibir anexos tanto no composer antes do envio quanto na thread apos o carregamento da mensagem, quando o backend retornar metadados de anexo. O tipo `Message` MUST declarar os campos de anexo usados pelo frontend se eles ja fizerem parte do payload recebido.

Anexos de mensagens carregadas SHOULD ser renderizados pelo slot `#files` de `UChatMessage` ou por slot `#content` customizado quando o payload do backend nao for compativel com partes de arquivo do Nuxt UI. A renderizacao MUST manter texto, horario e status visiveis.

#### Scenario: preview de anexo antes do envio

- **WHEN** o agente seleciona um arquivo no composer
- **THEN** o composer exibe preview compacto do arquivo com estado de upload, erro e acao de remover

#### Scenario: anexo em mensagem carregada

- **WHEN** uma mensagem carregada possui anexos
- **THEN** a thread exibe os anexos dentro ou junto ao `UChatMessage`, mantendo texto, horario e status visiveis

### Requirement: Lista lateral preserva densidade operacional de inbox

`ConversationsList.vue` SHALL continuar exibindo uma visao operacional densa da inbox, inspirada no Chatwoot: contato, canal/inbox, horario, preview da ultima mensagem, unread badge, selecao por hover, labels e assignee quando disponivel. A lista MUST evitar sobreposicoes em larguras pequenas e MUST manter preview vazio separado do estado vazio geral.

#### Scenario: conversa com metadados operacionais

- **WHEN** uma conversa possui inbox, labels, assignee, unread count e ultima mensagem
- **THEN** a lista mostra esses sinais de forma escaneavel sem esconder nome do contato nem preview

#### Scenario: conversa selecionada ou em hover

- **WHEN** o agente passa o mouse sobre um item ou o item esta selecionado
- **THEN** a lista permite selecao/bulk action sem deslocar o layout principal do item

## MODIFIED Requirements

### Requirement: Composer com triggers `/` e `@`

`ConversationComposer.vue` SHALL usar `UChatPrompt` com `rows=1`, `autoresize`, `maxrows` e `UChatPromptSubmit` para compor mensagens de atendimento. O composer MUST manter suporte aos triggers `/` e `@`, upload de attachments via presigned MinIO, indicador "digitando..." via realtime e contador de caracteres por canal. O upload via `UFileUpload` MUST ficar oculto/compacto quando nao houver anexos, evitando uma area de dropzone vazia permanente.

`UChatPromptSubmit` MUST refletir apenas estados reais do fluxo humano (`ready`, `submitted`, `error`). O componente MUST NOT expor acoes de `stop` ou `reload` se a implementacao nao tiver cancelamento ou repeticao de envio.

O composer SHALL usar os slots de `UChatPrompt` como uma area de trabalho compacta: `#header` para reply-to/anexos/banners compactos, `#leading` para ferramentas, `#footer` para contador/erros/indicadores e `#trailing` para envio.

- Trigger `/` abre `CannedResponsePicker.vue` com respostas rapidas da conta.
- Trigger `@` abre `MentionPicker.vue` com agentes da inbox atual.
- Upload de attachments usa botao de clipe, presigned URL e preview compacto.
- Indicador "digitando..." publica `conversation.typing` via realtime com throttle de 3 s.
- Contador de caracteres respeita WhatsApp 4096 e SMS 160.

#### Scenario: composer sem anexos

- **WHEN** a conversa esta aberta e nenhum arquivo foi selecionado
- **THEN** o composer exibe apenas o prompt compacto, botao de anexo, contador quando aplicavel e botao de envio, sem painel vazio de upload

#### Scenario: preview contextual no composer

- **WHEN** existe anexo selecionado, reply-to ativo ou restricao de resposta disponivel no estado da conversa
- **THEN** o composer exibe esse contexto de forma compacta no `#header`, sem aumentar a altura quando nenhum contexto esta ativo

#### Scenario: anexar imagem

- **WHEN** usuario seleciona ou arrasta uma imagem sobre o composer
- **THEN** cliente pede URL assinada (`POST /accounts/:accountId/uploads/signed-url`), faz PUT direto ao storage e envia mensagem com `attachments: [{ url, type }]`

#### Scenario: enviar texto simples

- **WHEN** o agente digita uma mensagem valida e aciona envio
- **THEN** `UChatPromptSubmit` entra em estado `submitted`, a mensagem e enviada para `POST /accounts/:accountId/conversations/:conversationId/messages` e o prompt e limpo apos sucesso

#### Scenario: limite de caracteres

- **WHEN** o texto excede o limite do canal
- **THEN** o contador usa cor de erro semantica e o envio fica desabilitado ate o conteudo voltar ao limite permitido

#### Scenario: canal com restricao de texto e anexo

- **WHEN** o canal exigir texto e anexos em mensagens separadas
- **THEN** a UI nao deve combinar payloads de forma visualmente enganosa; se o backend atual nao suportar separacao por canal, a restricao deve ser documentada antes de alterar o envio

### Requirement: Componentes Chat de IA permanecem fora do fluxo principal

A refatoracao SHALL tratar `Chat` como pagina-guia da familia de componentes, nao como componente importavel. A thread principal de atendimento SHALL NOT introduzir Vercel AI SDK, `@ai-sdk/vue`, Comark, `UChatReasoning` ou `UChatTool`. `UChatShimmer` MAY ser usado apenas para estados reais de carregamento/envio, e `UChatPalette` MAY ser considerado apenas para wrappers futuros de modal/drawer, sem substituir o `UDashboardPanel` principal.

#### Scenario: conversa humana sem IA

- **WHEN** uma conversa comum de atendimento e aberta
- **THEN** a UI usa `UChatMessages`, `UChatMessage`, `UChatPrompt` e `UChatPromptSubmit`, sem renderizar `UChatReasoning` ou `UChatTool`

#### Scenario: wrapper mobile

- **WHEN** a thread e aberta no fluxo mobile com `USlideover`
- **THEN** a implementacao preserva o layout atual e so usa `UChatPalette` se ele melhorar o wrapper sem alterar contratos nem criar nova semantica de IA
