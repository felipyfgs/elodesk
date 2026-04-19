## ADDED Requirements

### Requirement: Modelagem de Inbox polimórfica

O schema Prisma SHALL definir:
- `Inbox` com `id`, `accountId`, `name`, `channelType` (enum começando com `whatsapp`), `channelId`, `greetingMessage?`, `workingHours Json?`, `createdAt`, `updatedAt`.
- `ChannelWhatsapp` com `id`, `inboxId` (unique 1:1), `wzapSessionId`, `wzapToken` (criptografado), `webhookSecret`, `status` enum (`DISCONNECTED|CONNECTING|QR|CONNECTED|LOGGED_OUT|ERROR`), `jid?`, `phoneNumber?`, `pushName?`, `lastConnectedAt?`.

#### Scenario: criar inbox cria channel vinculado

- **WHEN** `POST /api/v1/accounts/:id/inboxes {name, channelType:"whatsapp"}` é processado
- **THEN** uma linha em `Inbox` + uma linha em `ChannelWhatsapp` são criadas atomicamente
- **AND** `Inbox.channelId` referencia o `ChannelWhatsapp.id`

### Requirement: Lifecycle de criação de sessão WhatsApp

`POST /api/v1/accounts/:accountId/inboxes` SHALL executar, numa única transação de negócio:
1. Chamar `wzap.createSession({name, token: generated})` — obtém `sessionId` e recebe o `token` gerado pelo wzap como API key da sessão.
2. Chamar `wzap.createWebhook(sessionId, {url: "{API_URL}/wzap/webhook/{channelId}", secret: generated, events: ["All"]})`.
3. Criar `Inbox` + `ChannelWhatsapp` com `wzapSessionId`, `wzapToken` criptografado e `webhookSecret`.
4. Chamar `wzap.connectSession(sessionId)` para iniciar emparelhamento.
5. Responder 201 com `{inbox, channel}`.

Em caso de falha em qualquer passo após (1), o handler MUST chamar `wzap.deleteSession()` para limpar o wzap antes de retornar erro.

#### Scenario: criação bem-sucedida

- **WHEN** todos os passos completam
- **THEN** retorna 201 com `status="CONNECTING"`

#### Scenario: falha ao criar webhook

- **WHEN** `wzap.createWebhook` falha
- **THEN** `wzap.deleteSession` é chamado e a rota retorna 502 sem gravar nada no DB local

### Requirement: Obter QR code

`GET /api/v1/channels/:channelId/qr` SHALL retornar o QR atual se `ChannelWhatsapp.status=QR`, caso contrário 409. Não faz polling ativo — o frontend deve escutar `qr.update` via Socket.IO.

#### Scenario: QR disponível

- **WHEN** status é `QR` e webhook `QR` recebeu um valor recente
- **THEN** retorna 200 com `{qr: "dataURL ou base64", expiresAt}`

#### Scenario: QR não disponível

- **WHEN** status é `CONNECTED`, `DISCONNECTED` ou `ERROR`
- **THEN** retorna 409 com `{message: "qr not available in current status"}`

### Requirement: Desconectar sessão sem apagar

`POST /api/v1/channels/:channelId/disconnect` SHALL chamar `wzap.disconnectSession(sessionId)` e atualizar `ChannelWhatsapp.status=DISCONNECTED`. Mensagens históricas MUST permanecer intactas.

#### Scenario: desconexão preserva dados

- **WHEN** inbox conectado é desconectado
- **THEN** `ChannelWhatsapp.status=DISCONNECTED` mas `Inbox`, `Conversation`s e `Message`s continuam existindo

### Requirement: Deletar inbox apaga remotamente também

`DELETE /api/v1/inboxes/:inboxId` SHALL chamar `wzap.deleteSession(sessionId)` antes de remover localmente. Se o wzap retornar erro diferente de 404, a rota retorna 502 sem apagar nada local.

#### Scenario: wzap já deletou (404)

- **WHEN** `wzap.deleteSession` retorna 404
- **THEN** o backend deleta localmente e retorna 204 (consistência eventual)

#### Scenario: erro transiente no wzap

- **WHEN** `wzap.deleteSession` retorna 5xx
- **THEN** o backend NÃO deleta nada localmente e retorna 502 para o cliente tentar novamente

### Requirement: wzapToken criptografado em repouso

O campo `ChannelWhatsapp.wzapToken` SHALL ser criptografado com AES-256-GCM usando KEK de env `BACKEND_KEK` antes de ser persistido. O token em claro MUST existir só em memória do processo que precisa chamar o wzap.

#### Scenario: DB dump não expõe tokens

- **WHEN** `SELECT wzap_token FROM channel_whatsapp`
- **THEN** retorna bytes criptografados, não o token em claro

#### Scenario: rotação de KEK

- **WHEN** `BACKEND_KEK` rotaciona
- **THEN** existe script `pnpm rotate-kek` que re-encripta todos os `wzapToken` com a nova chave
