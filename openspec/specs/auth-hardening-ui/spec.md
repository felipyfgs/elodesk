# auth-hardening-ui Specification

## Purpose
TBD - created by archiving change complete-product-ui-ux. Update Purpose after archive.
## Requirements
### Requirement: Páginas públicas usam layout `auth` separado

O frontend SHALL criar `frontend/app/layouts/auth.vue` minimalista (sem sidebar) para as rotas públicas `/login`, `/forgot-password`, `/reset-password` — fora do `UDashboardGroup` do template. Estrutura: `<UApp>` wrapper + `<UContainer>` centralizado + card central `UPageCard` contendo o form.

Todas as páginas públicas MUST declarar `definePageMeta({ layout: 'auth', auth: false })`.

#### Scenario: rota pública não carrega o dashboard

- **WHEN** usuário acessa `/forgot-password` sem estar autenticado
- **THEN** o layout `auth` renderiza, não há tentativa de carregar `useRealtime()` nem stores protegidas

### Requirement: `/forgot-password` com form Zod e resposta neutra

`frontend/app/pages/forgot-password.vue` SHALL usar `UForm :schema="forgotSchema" :state` com `UFormField` de email único (padrão `pages/settings/index.vue` do template). Schema Zod em `frontend/app/schemas/auth/forgot.ts`:

```ts
export const forgotSchema = z.object({ email: z.string().email() })
```

Após submit, `POST /auth/forgot` é chamado; UI mostra sempre mensagem neutra "Se o email existir, um link foi enviado" (não vaza existência).

Componentes em `frontend/app/components/auth/`:

- `AuthCard.vue` — `UPageCard` wrapper com slot de título + descrição + slot default
- `AuthFooterLinks.vue` — links "Voltar para login", "Criar conta"

#### Scenario: usuário solicita reset

- **WHEN** usuário preenche email e submete
- **THEN** request é enviado, `AuthCard` troca para estado de sucesso com mensagem neutra + link "voltar para login"

### Requirement: `/reset-password` com validação pré-mount

`frontend/app/pages/reset-password.vue` SHALL:

1. Ler `token` do query param via `useRoute().query.token`
2. No `onMounted`, chamar `GET /auth/reset/:token/validate` — se 404/410, renderizar estado de erro
3. Renderizar `UForm` com schema Zod (padrão `settings/security.vue` do template):

```ts
export const resetSchema = z.object({
  password: z.string().min(8),
  confirm: z.string().min(8)
}).refine((d) => d.password === d.confirm, { path: ['confirm'], message: 'Senhas não conferem' })
```

4. Submit chama `POST /auth/reset` com `{ token, password }`

Componente em `frontend/app/components/auth/`: `PasswordField.vue` — `UInput type="password"` com toggle visibility (reutilizável por `/reset` e `/profile`).

#### Scenario: token inválido ou expirado

- **WHEN** usuário abre link com token inválido
- **THEN** UI mostra `AuthCard` com ícone `i-lucide-triangle-alert`, texto "Link expirado ou inválido" e CTA para `/forgot-password`

#### Scenario: reset bem-sucedido

- **WHEN** nova senha é definida com sucesso
- **THEN** backend revoga todos refresh tokens ativos do user; frontend redireciona para `/login` com toast `color: 'success'` "Senha redefinida, faça login com a nova senha"

### Requirement: `/profile/mfa` para setup TOTP

`frontend/app/pages/profile/mfa.vue` SHALL exibir o fluxo em etapas dentro do layout dashboard (`definePageMeta({ layout: 'dashboard' })`):

1. **Setup** — `MfaQrCode.vue` renderiza QR via lib `qrcode.vue` a partir do `otpauth_uri` retornado por `POST /auth/mfa/setup`, mais o segredo em texto (fallback manual)
2. **Verificar** — `MfaCodeInput.vue` (6 dígitos) + botão "Ativar" chama `POST /auth/mfa/enable` com code
3. **Recovery codes** — `MfaRecoveryCodes.vue` exibe 8 códigos ONCE, botões "Copiar todos" e "Baixar .txt"

Componentes em `frontend/app/components/auth/mfa/`:

- `MfaQrCode.vue` — props: `uri`, `secret`
- `MfaCodeInput.vue` — 6 inputs separados com auto-advance (reutilizável pelo step de login)
- `MfaRecoveryCodes.vue` — props: `codes: string[]`, emits `done`
- `MfaStatusCard.vue` — card mostrando status (ativo/inativo) + botão desativar

Schemas em `frontend/app/schemas/auth/mfa.ts`.

#### Scenario: ativar MFA

- **WHEN** usuário escaneia QR e digita código correto
- **THEN** `POST /auth/mfa/enable` é chamado, `MfaRecoveryCodes` aparece com 8 códigos, botões de copiar/baixar ficam habilitados; navegar para outra rota antes de confirmar mostra `UModal` de aviso

### Requirement: Step MFA no login quando `mfa_required`

`frontend/app/pages/login.vue` SHALL tratar a resposta de `POST /auth/login` como discriminated union:

- `{ mfa_required: true, mfa_token: string }` → trocar para step MFA usando `MfaCodeInput.vue` + link "usar recovery code" (swap para input simples), submeter via `POST /auth/mfa/verify` com `{ mfa_token, code }`
- `{ accessToken, refreshToken }` → fluxo normal de login

Schema em `frontend/app/schemas/auth/login.ts` cobre ambos casos via `z.union`.

Rate limit: após 3 tentativas incorretas em 5 min, UI desabilita o input e mostra countdown.

#### Scenario: login com MFA ativo

- **WHEN** usuário com MFA ativo insere senha correta
- **THEN** backend retorna `{mfa_required: true, mfa_token}`; frontend mostra step MFA (mesmo `AuthCard`, conteúdo trocado); código válido completa o login; 3 tentativas erradas bloqueiam o input por 5 min com countdown

