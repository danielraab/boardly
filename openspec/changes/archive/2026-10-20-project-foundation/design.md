# Design

## Context

What's in the repo now: `frontend/` is a TanStack Router file-router scaffold (React 19, Tailwind 4, Vite 8, Biome, pnpm). It has two competing router setups (`main.tsx` and an unused `router.tsx`, each declaring `Register`). `backend/` contains only `go.mod` (`at.draab/my-homebase`). There's no root `.gitignore` and no tests. The product vision is in the README. See proposal.md for why this change is needed.

The constraints come from the next sessions. A Dockerfile will build the frontend and stack it into the Go build, so the production artifact must be one Go binary. Auth (magic link, passkeys) follows in `magic-link-auth` and `passkeys` and needs a database, a JSON API and the app shell's menu.

## Goals / Non-Goals

**Goals:**
- One `go build` produces a binary that serves the API and the embedded SPA.
- Running `pnpm dev` and `go run` side by side gives a hot-reloading dev setup with no CORS.
- Conventions (error format, config, migrations, i18n, tests) that later changes can copy.

**Non-Goals:**
- Dockerfile and CI (next session).
- Any auth or user data. The user menu only knows the signed-out state here.
- Server-side rendering.

## Decisions

### Layout

```
famodash/
├── Makefile                 dev-backend, dev-frontend, build, test, check
├── backend/
│   ├── main.go              wiring: config → db → migrate → http server
│   └── internal/
│       ├── config/          FAMODASH_* env parsing + defaults
│       ├── db/              open SQLite, migrate.go, migrations/*.sql (embedded)
│       ├── httpapi/         router, JSON helpers, error type, handlers
│       └── web/             embed.go + dist/ (Vite output, gitignored except .gitkeep)
└── frontend/
    └── src/
        ├── main.tsx, routes/, components/, i18n/, locales/{en,de}.json
        └── test/            Vitest setup
```

### Embedding the SPA: Vite builds straight into `backend/internal/web/dist`
`go:embed` cannot reach outside the package directory. So `vite.config.ts` sets `build.outDir` to `../backend/internal/web/dist` with `emptyOutDir: true`. The directory keeps a committed `.gitkeep` and is otherwise gitignored. The package uses `//go:embed all:dist`, so `go build` and `go test` work without a frontend build. When `index.html` is missing, non-API requests get a short plain-text "frontend not built" response with status 503.
- *Alternative*: copy `frontend/dist` into place in the Makefile or Dockerfile. That's an extra step and easy to forget locally. Rejected.

### HTTP: standard library `net/http` with Go 1.22+ method patterns
`http.ServeMux` with patterns like `GET /api/health` covers routing, including 405s. Routes are registered in a single place in `httpapi`. Every `/api/` request that matches no route goes to a catch-all `/api/` handler that writes the JSON `not_found` error. Go's built-in 405 is plain text, so the mux is wrapped to rewrite it into the JSON error format. Everything else goes to the SPA handler: serve the file from the embed FS if it exists, otherwise `index.html`. Hashed files under `/assets/` get `Cache-Control: public, max-age=31536000, immutable`, `index.html` gets `no-cache`.
- *Alternative*: chi or echo. No feature needed here justifies the dependency.

### Errors
There is one `apiError{Status int, Code, Message string}` type and a `writeJSON` / `writeError` helper pair. Handlers return errors instead of writing them, through a small `func(w, r) error` adapter. That keeps the error format from the `app-hosting` spec consistent.

### Logging and shutdown
`log/slog` with a JSON handler in production and a text handler when `FAMODASH_LOG_FORMAT=text`. A request-logging middleware logs method, path, status and duration. SIGINT/SIGTERM trigger `http.Server.Shutdown` with a 10s timeout.

### Configuration
| Variable | Default | Meaning |
|---|---|---|
| `FAMODASH_PORT` | `8080` | listen port |
| `FAMODASH_DB_PATH` | `./data/famodash.db` | SQLite file (parent dir created) |
| `FAMODASH_LOG_FORMAT` | `json` | `json` or `text` |

Later changes add their own variables here (base URL, SMTP, WebAuthn). Parsing is hand-written with `os.Getenv`. It's a small amount of code and keeps a dependency out.

### Database: SQLite through `modernc.org/sqlite`, with hand-rolled embedded migrations
It's pure Go, so it builds without cgo and works in a scratch or distroless image. Connection settings: `journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`. Migrations are numbered files `migrations/NNNN_name.sql` in an `embed.FS`. A `schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT)` table records what ran, and each file runs in its own transaction. The runner creates `schema_migrations` itself. This change ships only a baseline `0001_init.sql` that holds a comment and no tables, because there's no domain data yet. Runner tests use their own fixture FS.
- *Alternatives*: `mattn/go-sqlite3` needs cgo. goose and golang-migrate are more machinery than a handful of SQL files need.

### Frontend routing cleanup
`main.tsx` stays the single router setup and `router.tsx` is deleted. The root route renders `<AppShell><Outlet/></AppShell>` and sets `notFoundComponent`. Routes in this change: `/` (landing) and `/login` (placeholder text "coming soon", filled in by `magic-link-auth`). The TanStack devtools render only when `import.meta.env.DEV` is true.

Files that get removed: `src/router.tsx`, `.cta.json`, `frontend/README.md` (content moves to the root README), `tsr.config.json` and the `@tanstack/router-cli` dev dependency plus its `generate-routes` script (the Vite plugin already generates `routeTree.gen.ts`). Files that stay: `biome.json`, `.vscode/settings.json`, `tsconfig.json`, `pnpm-workspace.yaml`, `routeTree.gen.ts` (committed, so type checks work without a dev run).

### Dev proxy
`vite.config.ts` sets `server.proxy = { '/api': 'http://localhost:8080' }`. The browser only ever talks to `:3000` in dev, which matters for cookies and WebAuthn origins in later changes.

### UI primitives: Base UI (`@base-ui/react`) + lucide-react + Tailwind
Base UI's `Menu` gives positioning, focus management, Escape and outside-click handling, and ARIA roles. The user menu and the language switcher both use `Menu` (the switcher's items are radio items). Icons come from `lucide-react` (`CircleUserRound` for the avatar, `Languages` for the switcher). Styling uses Tailwind utility classes on Base UI parts, through data attributes such as `data-[highlighted]`.

### i18n: i18next + react-i18next + i18next-browser-languagedetector
Catalogs are `src/locales/en.json` and `de.json`, bundled statically because they're small. Detector order is `localStorage` (key `famodash.lang`) then `navigator`, with `supportedLngs: ['en','de']`, `nonExplicitSupportedLngs: true` (so `de-AT` → `de`), `fallbackLng: 'en'`, and caching to `localStorage` only on explicit change. A `languageChanged` listener sets `document.documentElement.lang`. A Vitest test checks that both catalogs have the same keys. Keys are namespaced by area, for example `shell.userMenu`, `landing.hero.title` and `features.<id>.title`.

### Landing page
It has three blocks: hero (name, tagline, CTA to `/login`), a screenshot placeholder (`aspect-video`, dashed border, "Screenshot coming soon") and a feature grid driven by a `features` array (`{id, icon}`, with text looked up by `features.<id>.*`). The initial placeholder entries are dashboards, favourites and karakeep. The grid is one column below `sm` and three at `lg`. Visually it's restrained, a neutral palette with one accent color. The visual pass happens when real screenshots exist.

### Tests
- Go: `httptest` covers SPA fallback, the 404/405 JSON errors, health, config parsing and migration idempotency (against a temp-dir DB).
- Frontend: Vitest + jsdom + `@testing-library/react` + `@testing-library/user-event`. They cover the user menu (open, Escape, outside click, sign-in navigation), the language switch (texts and `lang` attribute), catalog parity and landing rendering.
- `make test` runs both, and `make check` runs `biome check`, `tsc --noEmit` and `go vet`.

## Risks / Trade-offs

- [The Vite build writes into the backend tree, so `emptyOutDir` could delete the wrong files] → The out dir is a dedicated `dist/` that holds only build output and `.gitkeep`. A Vite plugin hook (or a Makefile step) recreates `.gitkeep` after the build.
- [Go's built-in 405 response is plain text] → The mux wrapper rewrites it. A test covers this.
- [Hand-rolled migrations lack down migrations] → That's acceptable for a self-hosted single-node app. Rollback means restoring the DB file, and the README says so.
- [Base UI v1 is young and its API may still shift] → It's pinned to an exact version and used through two small wrapper components.

## Migration Plan

Nothing is deployed yet. The rename is a plain search-and-replace. The user renames the folder and the GitHub repo afterwards.
