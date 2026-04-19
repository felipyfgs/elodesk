## ADDED Requirements

### Requirement: Bootstrap do NestJS com Config, Logger, Health e Swagger

O backend SHALL inicializar com `ConfigModule` validando env com Zod, `LoggerModule` singleton usando pino (todo log começa com `component=<module>`), rota `/health` retornando `{status:"ok"}` e Swagger em `/docs` listando todos os endpoints.

#### Scenario: app sobe com env válido

- **WHEN** o desenvolvedor executa `pnpm dev` com `.env` populado
- **THEN** o servidor escuta na porta configurada
- **AND** `GET /health` responde 200 com `{"status":"ok"}`
- **AND** `GET /docs` renderiza a UI do Swagger

#### Scenario: app falha com env inválido

- **WHEN** `.env` tem uma variável obrigatória ausente
- **THEN** o app falha no startup com mensagem listando a variável faltante

### Requirement: Modelagem de User, Account, AccountUser e RefreshToken

O schema Prisma SHALL definir:
- `User` com `id`, `email` (unique), `passwordHash`, `name`, `createdAt`.
- `Account` com `id`, `name`, `slug` (unique), `features Json`, `createdAt`.
- `AccountUser` com `accountId`, `userId`, `role` enum `OWNER|ADMIN|AGENT`, chave primária composta.
- `RefreshToken` com `id`, `userId`, `tokenHash`, `expiresAt`, `revokedAt?`.

#### Scenario: criar User + Account na mesma transação

- **WHEN** o serviço de registro recebe email, senha e nome
- **THEN** insere User, Account (slug derivado do email), AccountUser com `role=OWNER` atomicamente
- **AND** rollback completo acontece se qualquer passo falhar

### Requirement: Registro de usuário

O backend SHALL expor `POST /api/v1/auth/register` aceitando `{email, password, name, accountName?}`. A senha MUST ser armazenada como hash Argon2id. Se `accountName` omitido, usa o `name` como nome da Account.

#### Scenario: registro com email novo

- **WHEN** `POST /api/v1/auth/register` com dados válidos
- **THEN** retorna 201 com `{user, account, accessToken, refreshToken}`
- **AND** `user.passwordHash` nunca é exposto na resposta

#### Scenario: registro com email duplicado

- **WHEN** já existe um User com o mesmo email
- **THEN** retorna 409 com mensagem genérica `"email already registered"` (não vaza se o email existe ou não em respostas 400/401)

### Requirement: Login com email e senha

O backend SHALL expor `POST /api/v1/auth/login` aceitando `{email, password}`. Senha é verificada com `argon2.verify`. Sucesso retorna `accessToken` (15min) e `refreshToken` (30d). Falha SHALL retornar 401 sem distinguir entre email inexistente ou senha errada.

#### Scenario: credenciais corretas

- **WHEN** email e senha válidos
- **THEN** retorna 200 com `{accessToken, refreshToken, user}`
- **AND** `refreshToken` é salvo como hash em `RefreshToken` com TTL de 30 dias

#### Scenario: credenciais erradas

- **WHEN** email não existe OU senha incorreta
- **THEN** retorna 401 com `{message: "invalid credentials"}`

### Requirement: Refresh de access token

O backend SHALL expor `POST /api/v1/auth/refresh` aceitando `{refreshToken}`. O handler MUST rotacionar o token: gera novo `refreshToken`, marca o antigo como `revokedAt=now()`, e retorna novo par `{accessToken, refreshToken}`.

#### Scenario: refresh válido rotaciona token

- **WHEN** `refreshToken` válido é enviado
- **THEN** retorna 200 com novo par de tokens
- **AND** o token antigo é marcado `revokedAt`

#### Scenario: refresh revogado

- **WHEN** `refreshToken` já foi usado (`revokedAt != null`)
- **THEN** retorna 401 e TODOS os outros refresh tokens do mesmo user SHALL ser revogados (defesa contra token replay)

### Requirement: Logout

O backend SHALL expor `POST /api/v1/auth/logout` autenticado, que revoga o `refreshToken` corrente e opcionalmente todos os demais do mesmo usuário quando `{allDevices: true}` for enviado.

#### Scenario: logout device atual

- **WHEN** `POST /api/v1/auth/logout` com body vazio
- **THEN** apenas o refresh token da sessão atual é revogado

#### Scenario: logout todos os devices

- **WHEN** `POST /api/v1/auth/logout {allDevices: true}`
- **THEN** todos os refresh tokens ativos do user são revogados
