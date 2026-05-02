## ADDED Requirements

### Requirement: Error logging in catch blocks SHALL use useErrorHandler
All `console.error()` calls in catch blocks within frontend components and stores SHALL be routed through the existing `useErrorHandler` composable instead of direct `console.error` calls.

#### Scenario: Catch block uses useErrorHandler
- **WHEN** an error is caught in a component or store
- **THEN** the error SHALL be passed to `useErrorHandler().handle(error, context)`
- **AND** no direct `console.error()` call SHALL be present outside `useErrorHandler` internals

### Requirement: Development-only warnings SHALL be guarded by import.meta.dev
All `console.warn()` calls that exist solely for development diagnostics SHALL be wrapped in `if (import.meta.dev)` guards to prevent logging in production builds.

#### Scenario: Console warn does not fire in production
- **WHEN** the frontend is built with `pnpm build`
- **THEN** `console.warn()` calls guarded by `import.meta.dev` SHALL be tree-shaken from the production bundle
- **AND** no unprotected `console.warn()` calls SHALL remain in the codebase
