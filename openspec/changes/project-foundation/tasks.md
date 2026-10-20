# Tasks

## 1. Rename and cleanup

- [x] 1.1 Rename to Famodash: Go module `famodash`, `package.json` name `famodash`, `index.html` title, README, CLAUDE.md. Verify `grep -ri "my-homebase\|dashnoli" --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=openspec .` returns nothing
- [x] 1.2 Remove `frontend/src/router.tsx`, `frontend/.cta.json`, `frontend/README.md`, `frontend/tsr.config.json`, the `@tanstack/router-cli` dev dependency and the `generate-routes` script. Verify `pnpm build` still succeeds
- [x] 1.3 Add a root `.gitignore` (`/backend/internal/web/dist/*`, `!/backend/internal/web/dist/.gitkeep`, `/data/`, `/bin/`, `.env`) and verify with `git status` that only intended files are untracked
- [x] 1.4 Fill `openspec/config.yaml` `context` with the tech stack, conventions (JSON error format, i18n keys, Base UI + Tailwind, tests per group) and the domain summary. Verify `openspec validate project-foundation` still passes

## 2. Go server skeleton

- [x] 2.1 Implement `internal/config` (`FAMODASH_PORT`, `FAMODASH_DB_PATH`, `FAMODASH_LOG_FORMAT` with defaults and validation) and verify table tests for defaults and an invalid port pass
- [x] 2.2 Implement `internal/httpapi` JSON helpers, the `apiError` type, the handler adapter, the `/api/` catch-all 404 and the JSON 405 rewrite. Verify `httptest` tests for `GET /api/does-not-exist` → 404 `not_found` and `DELETE /api/health` → 405 `method_not_allowed`
- [x] 2.3 Add request-logging middleware with `log/slog` and graceful shutdown in `main.go`. Verify by running `go run .` and stopping it with Ctrl+C: the log shows a clean shutdown

## 3. Database

- [x] 3.1 Add `modernc.org/sqlite`, implement `internal/db` Open (WAL, foreign keys, busy timeout, parent dir creation) and verify a test opens a temp-dir DB
- [x] 3.2 Implement the embedded migration runner with `schema_migrations` and `0001_init.sql`. Verify tests for a fresh DB (all applied) and a rerun (nothing reapplied)
- [x] 3.3 Implement `GET /api/health` with a DB ping. Verify tests for 200 `{"status":"ok"}` and 503 with a closed DB

## 4. Embedded web app

- [x] 4.1 Create `internal/web` with `//go:embed all:dist`, a committed `dist/.gitkeep` and the SPA handler (file or `index.html` fallback, cache headers, 503 "frontend not built" when `index.html` is missing). Verify `httptest` tests with a fake FS for asset, deep link `/profile` and the missing-build case
- [x] 4.2 Set Vite `build.outDir` to `../backend/internal/web/dist` with `.gitkeep` preserved, and add the `/api` dev proxy. Verify `pnpm build` writes there and `.gitkeep` survives
- [x] 4.3 Add a root `Makefile` with `dev-backend`, `dev-frontend`, `build` (frontend then `go build -o bin/famodash ./backend`), `test`, `check`. Verify `make build && ./bin/famodash` serves the app at `http://localhost:8080/` and `/api/health`

## 5. Frontend foundation

- [x] 5.1 Add `vitest`, `jsdom`, `@testing-library/react`, `@testing-library/user-event`, `@testing-library/jest-dom` and the `test` script. Verify a trivial test passes with `pnpm test`
- [x] 5.2 Add `@base-ui/react` (pinned), `lucide-react`, `i18next`, `react-i18next`, `i18next-browser-languagedetector`. Verify `pnpm install` and `pnpm build` succeed
- [x] 5.3 Set up i18n (`src/i18n/index.ts`, `src/locales/en.json`, `de.json`, detector config, `lang` attribute sync). Verify tests for catalog key parity, `de-AT` → German, `fr` → English and stored-choice precedence
- [x] 5.4 Make devtools dev-only in `__root.tsx` and verify the production bundle doesn't contain devtools (`grep -r TanStackDevtools backend/internal/web/dist` is empty)

## 6. App shell

- [x] 6.1 Build `AppShell` with a header (name linking to `/`, language switcher, avatar) wrapping the `Outlet`. Verify a test that the header renders on `/` and `/login`
- [x] 6.2 Build `LanguageSwitcher` (Base UI Menu with radio items English/Deutsch). Verify a test that switching changes the texts and `document.documentElement.lang` without reload
- [x] 6.3 Build `UserMenu` (Base UI Menu, `CircleUserRound` icon, accessible name, signed-out item "Sign in" → `/login`). Verify tests for keyboard open plus Escape with focus return, outside-click close and navigation to `/login`
- [x] 6.4 Add a `/login` placeholder route and the root `notFoundComponent` with a link home. Verify a test for `/no-such-page` showing the not-found message

## 7. Landing page

- [x] 7.1 Replace `routes/index.tsx` with the landing page: hero with CTA, screenshot placeholder (`aspect-video`), data-driven feature cards (dashboards, favourites, karakeep). Verify tests for the hero, the CTA href `/login` and one card per feature entry
- [x] 7.2 Check the responsive layout manually at 320px and 1280px in the browser (no horizontal scroll, one column on narrow screens)

## 8. Integration and docs

- [x] 8.1 Rewrite the README development section (Makefile targets, env vars, dev proxy, where the build output goes, rollback means restoring the DB file). Verify the documented commands run as written
- [x] 8.2 Run `make check` and `make test` and verify both pass

## Workflow follow-up

- Archive with `/opsx:archive` before applying `magic-link-auth`, because that change adds requirements to `app-shell` and `landing-page`.
