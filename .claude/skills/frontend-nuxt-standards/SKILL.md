---
name: frontend-nuxt-standards
description: Use when editing Nuxt 4 / Vue frontend files — .vue components, composables (use* prefix), Pinia stores (use<Name>Store), Zod schemas, i18n keys (pt-BR + en) or anything under frontend/app/. Enforces PascalCase components, <script setup lang="ts">, @nuxt/ui v4 and $fetch over provide/inject.
license: MIT
---

## Purpose

Apply consistent patterns for Nuxt/Vue frontend code.

## Architecture

- Nuxt framework with file-based routing
- Pinia for state management
- Composition API with composables
- Zod for form validation
- i18n for internationalization

## File Structure

```
app/
├── components/       # Vue components
├── composables/      # Reusable composition functions
├── layouts/          # Page layouts
├── middleware/       # Route middleware
├── pages/            # File-based routing
├── schemas/          # Zod validation schemas
├── stores/           # Pinia stores
├── types/            # TypeScript type definitions
└── utils/            # Utility functions
```

## Naming Conventions

### Components
- `PascalCase.vue`

### Composables
- `use<Name>.ts`

### Stores
- `<name>.ts` (lowercase)

### Pages
- `kebab-case.vue` or `camelCase.vue`
- Nested routes: `directory/page.vue`

### Schemas and Types
- `camelCase.ts`

## Code Style

- 2-space indentation
- LF line endings
- UTF-8 charset
- Trim trailing whitespace
- Insert final newline
- No trailing comma
- 1TBS brace style

## Composables Pattern

Composables wrap reusable logic using Vue composition APIs:
```typescript
export function useApi() {
  const config = useRuntimeConfig()
  const { token } = useAuth()
  const api = $fetch.create({
    baseURL: config.public.apiUrl,
    headers: { Authorization: `Bearer ${token.value}` }
  })
  return { api }
}
```

## Stores Pattern

Use Pinia `defineStore` with state, getters, and actions:
```typescript
export const useAuthStore = defineStore('auth', {
  state: () => ({ user: null, token: null, isLoading: false }),
  getters: { isAuthenticated: (state) => !!state.token },
  actions: {
    async login(email: string, password: string) { /* ... */ }
  }
})
```

## Validation Pattern

Use Zod schemas for form validation:
```typescript
import { z } from 'zod'

export const formSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email')
})

export type FormInput = z.infer<typeof formSchema>
```

## Middleware Pattern

Guard routes with Nuxt route middleware:
```typescript
export default defineNuxtRouteMiddleware((to, from) => {
  if (!isAuthenticated()) return navigateTo('/login')
})
```

## i18n Pattern

Use `useI18n()` composable for all UI strings:
```typescript
const { t } = useI18n()
const title = t('pages.dashboard.title')
```

## Rules

1. 2-space indentation required
2. LF line endings required
3. PascalCase for components
4. `use` prefix for composables
5. Zod for all form validation
6. i18n for all UI strings
7. Pinia stores for shared state
8. `$fetch` for API calls
9. Never use provide/inject for shared state

## Workflow Commands

### Lint
```bash
cd frontend && pnpm lint
```

### Typecheck
```bash
cd frontend && pnpm typecheck
```

### Lint with auto-fix
```bash
cd frontend && pnpm lint --fix
```

### Build
```bash
cd frontend && pnpm build
```

### Dev server
```bash
cd frontend && pnpm dev
```
