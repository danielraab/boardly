# Proposal

## Why

The repository is still an untouched scaffold. It has a generated TanStack frontend named `my-homebase`, an empty Go module and no shared conventions. Magic-link and passkey auth, and later dashboards and favourites, need one runnable base to build on: a single Go binary that serves a REST API and the built frontend, a translated UI shell, and a public landing page.

## What Changes

- **BREAKING** Rename the project from `my-homebase` / Dashnoli to **Famodash** everywhere: Go module, `package.json`, page title, README, CLAUDE.md and UI strings.
- Remove scaffold leftovers that serve no purpose (duplicate router setup in `router.tsx`, `.cta.json`, the generated frontend README, unused CLI config). Keep the files that are needed or that are best practice.
- Add a Go HTTP server: configuration from environment variables, a SQLite database with embedded migrations, a JSON REST API under `/api`, and a health endpoint.
- Embed the production frontend build into the Go binary and serve it with SPA fallback. Unknown `/api/*` paths return JSON 404s instead of the SPA.
- In development, proxy `/api` from the Vite dev server to the Go server.
- Add i18n with English and German, browser language detection and a language switcher in the header.
- Add the app shell: a header with the Famodash name, a language switcher and a generic person avatar that opens a user menu. In this change the menu only has the signed-out content ("Sign in" / "Anmelden", linking to `/login`).
- Add a public landing page with placeholder sections for a future screenshot and feature descriptions.
- Add a placeholder `/login` page and a not-found page.
- Add frontend tests (Vitest + Testing Library), Go tests, and a root `Makefile` for dev and build commands.
- Fill in `openspec/config.yaml` with the project context.

## Capabilities

### New Capabilities

- `app-hosting`: how the single binary serves the REST API and the embedded web app, including configuration, the health endpoint, the JSON error format and SPA fallback.
- `i18n`: supported languages, language detection, switching and persisting the choice.
- `app-shell`: the global header, the avatar and the user menu shown on every page.
- `landing-page`: the public page at `/` for visitors who are not signed in.

### Modified Capabilities

None. No specs exist yet.

## Impact

- **Backend**: new code under `backend/` (module renamed to `famodash`). New dependency: `modernc.org/sqlite`.
- **Frontend**: new dependencies `@base-ui/react`, `lucide-react`, `i18next`, `react-i18next`, `i18next-browser-languagedetector`, plus `vitest` and `@testing-library/react` as dev dependencies. The production build output moves into the Go embed directory.
- **Tooling**: new root `Makefile` and `.gitignore`. The README development section is rewritten.
- **Follow-up changes**: `magic-link-auth` and `passkeys` build on this change.
