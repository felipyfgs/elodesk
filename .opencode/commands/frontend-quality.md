---
description: Run a full quality review on the Nuxt frontend — lint, format, typecheck, and auto-fix everything possible
---

Run a comprehensive quality review on the Nuxt frontend. This checks code health and **automatically fixes every issue it can**.

**What this command does:**

1. **Install** — ensures dependencies are up to date
2. **Lint** — runs ESLint, reports and auto-fixes errors
3. **Typecheck** — runs `nuxt typecheck`, reports errors
4. **Format** — applies Prettier/formatter if configured
5. **i18n** — checks for missing translations (pt-BR + en)
6. **Store consistency** — validates Pinia stores follow naming conventions
7. **Component hygiene** — checks for proper `<script setup lang="ts">` usage

**Steps**

1. **Install dependencies**
   ```bash
   cd frontend && pnpm install
   ```

2. **Lint and auto-fix**
   ```bash
   cd frontend && pnpm lint --fix
   ```
   Report which files were changed and any unfixable errors.

3. **Typecheck**
   ```bash
   cd frontend && pnpm typecheck
   ```
   Report any type errors.

4. **Check i18n completeness**
   Scan `frontend/app/i18n/` for locale files. Compare keys between `pt-BR.json` and `en.json`. Report any missing keys in either locale.

5. **Check Pinia stores**
   Verify all stores in `frontend/app/stores/`:
   - Use `defineStore` with a string id as first argument
   - Export a composable function named `use<Name>Store`
   - Report any deviations

6. **Check composables**
   Verify all composables in `frontend/app/composables/`:
   - Named with `use` prefix
   - Export a function
   - Report any deviations

**Output**

Summarize all actions taken in a clear report:

```
## Frontend Quality Review — Auto-Fix Applied

| Check          | Status  | Fixed | Remaining |
|----------------|---------|-------|-----------|
| Install        | done    | —     | —         |
| Lint           | pass/fail | N   | N         |
| Typecheck      | pass/fail | —   | N         |
| i18n           | checked | —     | N missing |
| Stores         | checked | —     | N issues  |
| Composables    | checked | —     | N issues  |
```

List every file that was modified with a brief description of what was fixed. Flag any issues that require manual review with `file:line` references.

If everything is clean, report: "All quality checks passed — frontend is clean."
