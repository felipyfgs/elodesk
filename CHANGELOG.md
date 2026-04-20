# Changelog

## 2026-04 — complete-product-ui-ux (in progress)

Raises product coverage from ~35% to ~90% of Chatwoot-like scope. Entregue em
fases; cada fase é shippable isoladamente.

### Fase 0 — Shell upgrade (frontend)
- `layouts/dashboard.vue` e `layouts/auth.vue` baseados em `@nuxt/ui` v4
- `UDashboardSidebar` multinível com Cmd+K (`UDashboardSearch`) e atalhos
  `g-h/g-c/g-o/g-i/g-r/g-s/n`
- Componentes shell portados: `TeamsMenu`, `UserMenu`, `NotificationsSlideover`
  ligado ao `useRealtime` e store Pinia

### Fase 1 — Inboxes multi-canal
- `/inboxes` grid de `InboxCard`, `/inboxes/new` com 8 wizards (api, whatsapp,
  sms, instagram, facebook_page, telegram, web_widget, email)
- Sub-rotas `/inboxes/[id]/{settings,agents,webhooks,business-hours}`
- Rotação de HMAC token exibida ONCE
- Backend: `GET/PUT /accounts/:aid/inboxes/:id/agents`

### Fase 2 — Contatos CRM
- `GET /accounts/:aid/contacts` com filtros server-side + `POST /contacts/import`
  (streaming CSV, 10 MB, batch 500, idempotent por `(account_id, lower(email))`)
- Stores + schemas Zod + `UTable` TanStack + bulk actions + segments

### Fase 3 — Conversas avançadas
- Layout 3-panel resizable (sidebar + lista + thread)
- Abas Mine / Unassigned / All / Mentions via query param
- Composer com canned (`/`), mentions (`@`), attachments MinIO, typing indicator

### Fase 4 — Admin & Settings
- Backend: agents (invite magic link, CRUD), users profile, macros (executor
  transacional com 8 ações), SLA (CRUD + attach-on-create), audit logs
  (particionados mensalmente, retenção 90d via job), webhooks outbound
- Frontend: Settings com 11 sub-rotas (Profile, Agents, Macros, SLA,
  Integrations, Audit logs, Notifications + Teams/Labels/Canned/Attributes)

### Fase 5 — Reports
- Backend: índices dedicados + `GET /reports/{overview,conversations,:entity,
  csat,sla}` com agregação on-the-fly
- Frontend: `/reports/overview`, `/reports/conversations`, drill-down por
  `agents|inboxes|teams|labels`, `/reports/csat` (stub), `/reports/sla`

### Fase 6 — Auth hardening
- Password recovery (`/auth/forgot`, `/auth/reset`) com token 32 bytes SHA-256,
  TTL 30m, single-use
- MFA TOTP (RFC 6238, 6 digits, 30s) + recovery codes + login discriminated
  union
- Frontend: `/forgot-password`, `/reset-password`, `/profile/mfa`, MFA step em
  `/login`

### Fase 7 — Notifications center
- Backend: tabela `notifications`, helper com realtime broadcast em
  `account:<aid>:user:<uid>`, preferências JSONB por user
- Frontend: store Pinia + `/notifications`, badge no sino, `/settings/notifications`
- Integração: `conversation.Assign` e SLA breach emitem notificações.
  Mentions em mensagens e nova conversa em inbox continuam pendentes (PR
  follow-up)

### Jobs
- SLA breach (60s) — ticker em processo detecta breaches, emite `sla.breached`,
  persiste notification + audit log
- Audit retention (24h) — purga audit_logs > 90 dias

### Follow-ups adiados
- Campanhas, Help Center, Captain (AI), Custom Roles, automation builder
  completo, drill-down de reports por agent/team/label com timeline real
- Mentions parser e notificações para nova-conversa-em-inbox
- Worker asynq dedicado para outbound webhook e tasks de channels
