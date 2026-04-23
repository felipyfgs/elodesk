## Context

A area de conversas em `frontend/app/components/conversations/` ja usa Nuxt UI 4 e parte dos componentes de chat: `UChatMessages`, `UChatMessage`, `UChatPrompt` e `UChatPromptSubmit`. O comportamento atual, porem, mistura adaptacao de dados, layout e excecoes visuais no mesmo componente. Isso causa estados incoerentes, como `UFileUpload` ocupando uma area vazia dentro do composer, contato exibido como `--`, preview lateral dizendo "Sem conversas ainda." dentro de uma conversa existente e status de mensagem numerico sendo comparado com strings.

O objetivo e refatorar apenas o frontend, mantendo `useConversationRealtime`, stores Pinia e endpoints existentes como fonte de dados.

```
Conversation API / realtime
        |
        v
Pinia stores: conversations + messages
        |
        v
adaptadores de view model
        |
        +--> ConversationsList
        +--> ConversationThread -> UChatMessages -> UChatMessage
        +--> ConversationComposer -> UChatPrompt -> UChatPromptSubmit
```

## Nuxt UI MCP Context

Consulta feita no MCP do Nuxt UI v4 para a documentacao real de `Chat` e dos componentes `Chat*`, mais os temas gerados em `frontend/.nuxt/ui/chat-*.ts`.

| Item | Uso nesta refatoracao | Contexto confirmado |
|---|---|---|
| `Chat` | Referencia arquitetural, nao componente Vue direto | A pagina `/docs/components/chat` descreve o conjunto de componentes para interfaces de chat, originalmente orientado a AI SDK/`UIMessage`. |
| `UChatMessages` | Viewport principal da thread | Aceita `messages?: UIMessage[]`, `status?: ChatStatus`, `shouldAutoScroll`, `shouldScrollToBottom`, `autoScroll`, `spacingOffset`, props `user`/`assistant` e slots `leading`, `files`, `content`, `actions`, `default`, `indicator`, `viewport`. O tema gerado usa `root`, `indicator`, `viewport` e `autoScroll`. |
| `UChatMessage` | Bolha/item individual | Requer `id`, `role` (`system`, `user`, `assistant`) e `parts`. O prop `content` esta deprecado; a refatoracao deve usar `parts` ou slot `#content`. Suporta `variant`, `side`, `avatar`, `actions`, `compact`, `metadata`, slots `files`, `content` e `actions`. O tema tem variantes `solid`, `outline`, `soft`, `subtle`, `naked`; `side=right` limita largura a `max-w-[75%]`. |
| `UChatPrompt` | Composer compacto | Renderiza como `form` por padrao, com `rows`, `autoresize`, `maxrows`, `error`, `disabled`, `placeholder`, slots `header`, `footer`, `leading`, `default`, `trailing`. O tema usa `root`, `header`, `body`, `footer` e `base`. |
| `UChatPromptSubmit` | Botao de envio | Usa `status` `ready`, `submitted`, `streaming`, `error` para alternar icone/cor. Em atendimento humano, usar `ready`, `submitted` e `error`; nao ligar `stop`/`reload` sem operacoes reais de cancelar/repetir. |
| `UChatPalette` | Fora da thread principal | E wrapper para chat em modal/drawer, com slots `root`, `prompt`, `close`, `content`. Pode ser considerado futuramente para o `USlideover` mobile, mas o painel principal deve continuar em `UDashboardPanel`. |
| `UChatReasoning` | Fora do escopo atual | Componente colapsavel para raciocinio de IA/streaming. Nao deve aparecer em conversa humana sem feature de assistente de IA. |
| `UChatTool` | Fora do escopo atual | Componente colapsavel para invocacao de ferramentas de IA. Nao deve ser usado para mensagens de atendimento. |
| `UChatShimmer` | Uso opcional so para carregamento | Texto com shimmer para streaming/loading. Pode apoiar indicador temporario de envio/carregamento, mas nao deve substituir renderizacao normal de mensagens humanas. |

Implicacao principal: a UI pode usar `UChatMessages` com slot `default` e renderizacao manual de `UChatMessage`, em vez de forcar a store a virar `UIMessage[]`. Cada `UChatMessage`, porem, deve receber `parts` validas e nunca o prop deprecado `content`.

## Chatwoot Reference Context

Exploracao feita em `/home/obsidian/dev/project/_refs/chatwoot` para entender a UX de uma inbox de atendimento madura. O objetivo nao e copiar componentes ou classes do Chatwoot, mas usar seus padroes de interacao como referencia para adaptar o chat Nuxt UI ao nosso dominio.

Arquivos de referencia principais:

- `_refs/chatwoot/app/javascript/dashboard/components-next/message/Message.vue`
- `_refs/chatwoot/app/javascript/dashboard/components-next/message/MessageList.vue`
- `_refs/chatwoot/app/javascript/dashboard/components-next/message/bubbles/Base.vue`
- `_refs/chatwoot/app/javascript/dashboard/components/widgets/conversation/MessagesView.vue`
- `_refs/chatwoot/app/javascript/dashboard/components/widgets/conversation/ReplyBox.vue`
- `_refs/chatwoot/app/javascript/dashboard/components/widgets/conversation/ConversationCard.vue`

Padrao observado:

```
ConversationView
  |
  +-- ChatList -> ConversationCard
  |
  +-- ConversationBox
        |
        +-- ConversationHeader
        |
        +-- MessagesView
              |
              +-- MessageList
              |     |
              |     +-- Message -> resolve variant/orientation/kind
              |           |
              |           +-- bubble by kind: text, activity, email, image, file, audio, video, location, unsupported
              |
              +-- ReplyBox -> reply/note mode, reply-to, editor, attachments, action bar
```

Traducao para o nosso Nuxt UI:

```
UDashboardPanel
  |
  +-- Header compacto de atendimento
  |
  +-- UChatMessages
  |     |
  |     +-- unread/date/load indicators
  |     |
  |     +-- UChatMessage
  |           |
  |           +-- #content -> bubble por tipo
  |           +-- #files   -> anexos/chips/media
  |           +-- #actions -> horario, status, erro/retry
  |
  +-- UChatPrompt
        |
        +-- #header   -> reply-to, anexos, banners compactos
        +-- #leading  -> attach, emoji, templates, nota privada
        +-- #footer   -> contador, erros, typing/draft
        +-- #trailing -> UChatPromptSubmit
```

| Padrao Chatwoot | Adaptacao proposta com Nuxt UI | Escopo |
|---|---|---|
| `Message` resolve variante, orientacao e componente de bolha | Criar view model local com `role`, `side`, `variant`, `bubbleKind`, `parts`, `attachments`, `statusMeta`, `replyTo` | Atual |
| Mensagens de contato a esquerda, agente/bot a direita, atividade ao centro | Mapear incoming para `assistant/left`, outgoing para `user/right`, activity/template para `system/center` via slot | Atual |
| Variantes visuais: user, agent, private, activity, bot/template, error, unsupported | Usar classes semanticas Nuxt UI: primary/neutral/warning/error, sem classes `bg-n-*` do Chatwoot | Atual |
| Agrupamento por mesmo remetente, mesmo tipo e mesmo minuto | Reduzir avatar/meta de mensagens consecutivas quando houver sender/timestamp suficientes | Atual se simples; aprofundar depois |
| Unread divider e scroll para primeira nao lida | Adicionar separador se houver marcador confiavel; senao manter auto-scroll do `UChatMessages` | Condicional |
| Reply-to preview dentro da bolha/composer | Renderizar preview se `contentAttributes.inReplyTo` ou campo equivalente existir | Condicional |
| Context menu por mensagem: copiar, deletar, responder, link, traduzir | Manter como backlog de segunda leva para nao inflar a refatoracao visual | Futuro |
| Composer como area de trabalho com reply/note, anexos, templates e restricoes | Usar slots de `UChatPrompt`; primeiro corrigir compactacao/anexos/status, depois adicionar nota privada/reply-to se o contrato existir | Atual + futuro |
| Lista lateral operacional com unread, canal, assignee, prioridade, labels e hover checkbox | Manter densidade da lista e corrigir preview vazio; bulk/context actions seguem fora do foco desta change | Atual parcial |

## Goals / Non-Goals

**Goals:**

- Fazer a thread seguir o modelo Nuxt UI de chat, com `UChatMessages`, `UChatMessage`, slots e props previsiveis.
- Separar adaptacao de dados de renderizacao, criando helpers locais para contato, mensagem, status e anexos.
- Aproximar a UX da thread de uma inbox de atendimento estilo Chatwoot: orientacao clara, estados de mensagem, bolhas por tipo e composer compacto.
- Reduzir o composer para uma altura natural quando nao ha anexos, usando `UChatPrompt` com `rows=1`, `autoresize` e `maxrows`.
- Corrigir estados vazios e fallback de texto sem mudar comportamento backend.
- Manter todas as cores em classes semanticas do Nuxt UI (`text-muted`, `bg-elevated`, `border-default`, etc.).

**Non-Goals:**

- Nao integrar Vercel AI SDK, `@ai-sdk/vue`, Comark ou streaming de IA.
- Nao usar `UChatReasoning`, `UChatTool` ou `UChatPalette` na thread principal de atendimento nesta etapa.
- Nao tratar a pagina `Chat` da documentacao como componente importavel.
- Nao buscar paridade total com Chatwoot nesta primeira refatoracao: email quoted reply, audio recorder, context menu completo, dashboard apps e sidebar de contato ficam fora.
- Nao alterar schema de banco, DTOs backend, endpoints REST ou websocket.
- Nao refatorar bulk actions, filtros avancados ou relatorios.

## Decisions

### 1. Tratar `UChat*` como camada visual, nao como contrato de dados

`Message` continuara vindo da API/stores no formato atual. A thread tera helpers como `messageRole`, `messageSide`, `messageVariant`, `messageBubbleKind`, `messageParts`, `messageStatusMeta`, `messageAttachments`, `messageReplyTo` e `shouldGroupWithNext` para converter dados do produto para props/slots do Nuxt UI.

`messageParts` deve produzir ao menos partes de texto compativeis com `UChatMessage`, por exemplo `{ type: 'text', text }`, e anexos devem ir para `#files` ou partes de arquivo somente se o payload local for compativel com o formato esperado pelo Nuxt UI. O prop deprecado `content` de `UChatMessage` nao deve ser usado.

Alternativa considerada: remodelar a store para usar diretamente `UIMessage`. Rejeitada porque acoplaria o dominio de atendimento a um formato de chat de IA e aumentaria o risco no realtime.

### 2. Usar papeis de chat de forma consistente para atendimento

Mensagens recebidas (`messageType=0`) serao mapeadas para `role='assistant'` e exibidas do lado esquerdo como contato. Mensagens enviadas (`messageType=1`) serao mapeadas para `role='user'` e exibidas do lado direito como agente/sistema de envio. Atividades/templates (`messageType=2|3`) serao mapeadas para `role='system'` e renderizadas por slot customizado como eventos centrais discretos.

Esse mapeamento aproveita os defaults reais do Nuxt UI: `user` tende a direita com variante `soft`, enquanto `assistant` tende a esquerda com variante `naked`. A refatoracao pode sobrescrever `side` e `variant` quando necessario, mas deve manter a regra em um helper unico.

Alternativa considerada: manter `assistant/user` como esta. Rejeitada porque a semantica de IA confunde a UI de atendimento quando nao ha assistente.

### 3. Separar orientacao, variante e tipo de bolha

Inspirado no Chatwoot, o frontend deve distinguir tres decisoes que hoje tendem a ficar misturadas:

- orientacao: esquerda, direita ou centro;
- variante visual: contato, agente, nota privada, atividade, template, erro, unsupported;
- tipo de conteudo: texto, anexo, midia, atividade, deletada, privada, vazia.

No Nuxt UI, `role` e `side` resolvem orientacao basica; `variant`, `ui` e slots resolvem a aparencia; `bubbleKind` escolhe o conteudo dentro de `#content` e `#files`.

Alternativa considerada: tentar resolver tudo com `messageRole` e `messageVariant`. Rejeitada porque fica insuficiente para anexos, privados, deletados, erros e atividades.

### 4. Agrupar mensagens consecutivas quando for seguro

Mensagens consecutivas do mesmo remetente, mesmo `messageType`, status nao falho e criadas no mesmo minuto podem ser agrupadas visualmente, reduzindo avatar e metadados repetidos. Isso aproxima a legibilidade do Chatwoot sem alterar o payload.

Se `senderId` ou timestamp confiavel estiver ausente, a UI deve cair para renderizacao nao agrupada.

Alternativa considerada: sempre agrupar por lado. Rejeitada porque mensagens de agentes diferentes no mesmo lado ficariam visualmente atribuidas de forma errada.

### 5. Composer compacto por padrao, preview de anexo sob demanda

`UFileUpload` sera usado apenas como mecanismo/preview quando houver arquivo selecionado ou quando o usuario acionar anexos. O composer nao deve renderizar uma dropzone vazia permanente. O botao de clipe continuara em `#leading`; `UChatPromptSubmit` ficara em `#trailing`; contador e erros ficarao em `#footer` ou area auxiliar compacta. `UChatPrompt` deve usar `rows=1`, `autoresize=true` e `maxrows` definido para impedir crescimento excessivo.

`UChatPromptSubmit` deve receber `status` derivado do estado real de envio (`ready`, `submitted`, `error`). O estado `streaming` e eventos `stop`/`reload` sao de fluxo de IA e so devem entrar se houver semantica real de cancelar/repetir envio.

O `#header` de `UChatPrompt` fica reservado para elementos contextuais compactos, como preview de anexo, reply-to e banners de restricao de resposta. O `#footer` deve receber contador, erro de upload/envio e indicadores discretos.

Alternativa considerada: substituir `UChatPrompt` por `UTextarea` puro, como a spec antiga menciona. Rejeitada porque `UChatPrompt` ja encapsula o comportamento esperado para chat e integra melhor com `UChatPromptSubmit`.

### 6. Corrigir status e anexos no nivel de tipo/view model

`MessageStatus` e numerico (`0=sent`, `1=delivered`, `2=read`, `3=failed`). A UI deve mapear esses codigos para icone, cor e label i18n antes de renderizar. `Message` deve declarar attachments se o backend ja envia esse campo, sem exigir mudanca de endpoint.

Alternativa considerada: comparar strings vindas do backend. Rejeitada porque contradiz a store atual e mascara status incorreto.

### 7. Estados vazios diferentes para lista e conversa

A lista principal pode usar `conversations.empty` quando nao ha conversas. Um item de conversa sem ultima mensagem deve usar uma chave separada, como `conversations.message.emptyPreview`. Uma thread sem mensagens deve usar estado proprio dentro de `UChatMessages`.

Alternativa considerada: reutilizar a chave atual em todos os lugares. Rejeitada porque gera a contradicao visual observada.

### 8. Manter componentes de IA explicitamente fora da primeira refatoracao

`UChatReasoning`, `UChatTool` e `UChatShimmer` foram consultados porque fazem parte da familia `Chat`, mas os dois primeiros pertencem a raciocinio/tool calling de IA. `UChatShimmer` tambem e voltado a streaming/loading de texto. A refatoracao de atendimento humano nao deve criar dependencias nem estados falsos para esses componentes; eles ficam documentados como caminho futuro para assistente de IA ou indicador de carregamento.

Alternativa considerada: usar `UChatTool` para eventos de sistema ou automacoes. Rejeitada porque a semantica visual comunica invocacao de ferramenta de IA, nao atividade de atendimento.

## Risks / Trade-offs

- Quebrar envio com anexos -> manter o payload atual e cobrir texto, arquivo e texto+arquivo na verificacao manual.
- Perder atualizacao realtime -> nao alterar `useConversationRealtime`; apenas renderizar o estado existente.
- `UChatMessage` nao cobrir algum tipo futuro de mensagem -> manter slots customizados para privado, deletado, atividade, anexos e status.
- Usar `UChatMessage` com formato de partes invalido -> validar o adaptador local e cobrir texto vazio, privado/deletado e anexos.
- Agrupar mensagens de remetentes diferentes por engano -> agrupar apenas com sender/tipo/timestamp confiaveis.
- Inflar escopo tentando copiar todo o Chatwoot -> limitar a primeira leva a presentation model, composer compacto, anexos/status, estados vazios e lista operacional.
- Layout mobile ficar apertado no `USlideover` -> testar breakpoints desktop e mobile com conversa vazia, curta e longa.

## Migration Plan

1. Criar helpers/adaptadores dentro de `ConversationThread.vue` ou em composable local `useConversationMessageView`.
2. Refatorar a thread preservando carregamento, realtime e props atuais.
3. Refatorar o composer para remover a area vazia de upload e preservar upload assinado existente.
4. Adicionar padroes Chatwoot viaveis na primeira leva: bubble kinds, agrupamento simples, status/erro e separadores condicionais.
5. Corrigir preview da lista e chaves i18n.
6. Rodar `pnpm lint` e `pnpm typecheck` em `frontend/`.
7. Validar visualmente `/accounts/:accountId/conversations` em desktop e mobile, comparando contra o checklist de UX inspirado no Chatwoot.

Rollback: reverter apenas os arquivos de frontend alterados; nenhum dado ou endpoint e migrado.

## Open Questions

- O backend ja retorna `attachments` em `GET /conversations/:id/messages` com forma estavel? Se sim, tipar em `Message`; se nao, renderizar apenas `contentAttributes` ate o contrato existir.
- O status `MessageStatus=0` representa "sent" ou "pending" para mensagens recem-criadas? A UI deve refletir a semantica real do backend.
- Existe um marcador confiavel de primeira mensagem nao lida ou ultimo visto do agente para implementar unread divider como no Chatwoot?
- O payload atual expoe `contentAttributes.inReplyTo` ou equivalente para implementar reply-to preview sem mudanca de backend?
- Nota privada deve ser parte desta refatoracao ou uma change propria usando `frontend/app/stores/notes.ts`?
