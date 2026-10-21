# Design

## Context

Famodash has users (UUID ids, `email`, optional `name`), sessions with the `requireUser` middleware, the `{"error": {"code", "message"}}` format, a CSRF check on state-changing requests, and migrations up to `0005_favourites.sql`. SQLite runs with `foreign_keys(1)`, WAL and `_txlock=immediate`. `GET /api/me` and `PATCH /api/me` (name only) live in `internal/httpapi/auth.go`. The Go server sends every non-`/api/` path to the SPA with an `index.html` fallback. In the frontend, `/` switches between `Landing` and `Home`, which is a placeholder. `AppShell` renders the header, and protected routes guard themselves in `beforeLoad` with `meQueryOptions`.

Requirements are in `dashboards`, `app-shell`, `home-page` and `user-profile` of this change. The next change adds the grid and widget editor to the dashboard page that this change introduces as a placeholder.

## Goals / Non-Goals

**Goals:**
- Dashboard entities and user slugs with database-level uniqueness and "at most one default".
- Root-level dashboard addresses `/<user-slug>/<dashboard-slug>` that never shadow the app's own routes.
- One public lookup endpoint that enforces visibility on the server, so private dashboards never reach a visitor's browser.
- Slug rules defined once per side (Go and TypeScript) and shared by user and dashboard slugs.

**Non-Goals:**
- Grid, widgets, editor and dashboard content (next change).
- Redirects from old slugs, and manual ordering of dashboards.
- Dashboard switcher on non-dashboard pages.

## Decisions

### Schema (migration `0006_dashboards.sql`)
```
ALTER TABLE users ADD COLUMN slug TEXT;
UPDATE users SET slug = lower(hex(randomblob(5)));        -- 10 chars [0-9a-f]
CREATE UNIQUE INDEX users_slug ON users (slug);

dashboards
  id          TEXT PK (uuid)
  user_id     TEXT NOT NULL → users (id) ON DELETE CASCADE
  slug        TEXT NOT NULL CHECK (slug = lower(slug))
  title       TEXT NOT NULL
  is_default  INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1))
  is_public   INTEGER NOT NULL DEFAULT 0 CHECK (is_public IN (0, 1))
  created_at  TEXT NOT NULL
  updated_at  TEXT NOT NULL
  UNIQUE (user_id, slug)
  UNIQUE INDEX dashboards_default ON (user_id) WHERE is_default = 1
```
- `users.slug` is added with `ADD COLUMN`, so it stays nullable at the database level. Making it `NOT NULL` would mean rebuilding `users`, which `sessions`, `favourite_nodes` and others reference while `foreign_keys` is on. The store always writes a slug, and a test checks that no user has a NULL slug after migrating. *Alternative:* a table rebuild with `PRAGMA foreign_keys=OFF` in the migration runner. That adds a special case to the runner for one column.
- The backfill uses `hex(randomblob(5))` in plain SQL. The result is a valid slug (hex digits are a subset of `[a-z0-9]`) and can't be a reserved word, because no reserved word is made only of hex digits and has 10 characters. A collision between existing users (1 in 2⁴⁰ per pair) would make the migration fail at the unique index. That is visible and harmless, since the migration transaction is rolled back and simply runs again on the next start.
- The partial unique index guarantees at most one default. Zero defaults is allowed, so nothing else is needed.
- Slugs are always stored lowercase, so plain `=` comparisons and unique indexes are case-insensitive in effect. Lookups lowercase their input first.
- The next change adds the dashboard content (a layout column or a widgets table). This change keeps no speculative columns.

### Package `internal/slugs`
- `Normalize(s) (string, error)`: trims, lowercases and checks `^[a-z0-9-]{1,32}$` (hyphens are allowed at the start and end too), returning `ErrInvalid` on a mismatch.
- `IsReserved(s) bool` and the exported `Reserved` list: `about, account, admin, api, app, assets, auth, dashboards, docs, favicon, favourites, health, help, karakeep, login, logout, me, new, public, register, search, settings, share, signup, static, tags, u, user, users, widgets`. The list is generous on purpose, because adding a word later can strand an existing user. The TypeScript copy in `features/dashboards/slug.ts` has a test that compares it against a JSON fixture, which a Go test checks against the same fixture, so the two can't drift.
- `Generate() string`: 10 characters from `crypto/rand` over `a-z0-9`.

User and dashboard slugs share `Normalize`. Only user slugs are checked against `Reserved`, because dashboard slugs only ever appear as the second path segment.

### User slug generation in `users.EnsureByEmail`
Inside the existing insert path, a candidate is generated with `slugs.Generate` (and generated again if it is reserved). If the candidate is taken, `-2`, `-3`, … is appended until the `SELECT` finds it free. The `INSERT … ON CONFLICT (email) DO NOTHING` then includes the slug. The calling transaction already holds the write lock (`_txlock=immediate`), so the check and the insert can't race. `SetSlug(id, slug)` normalizes the slug, checks it against `Reserved`, and maps a unique-constraint failure to `ErrSlugTaken`. `User` gains `Slug string \`json:"slug"\``.

### Store `internal/dashboards`
- `List(userID)` sorts with `ORDER BY title COLLATE NOCASE, slug`. NOCASE only folds ASCII, which is acceptable for a sort.
- `Create(userID, in)` and `Update(userID, id, patch)` run in one transaction. If `isDefault` becomes true, `UPDATE dashboards SET is_default = 0 WHERE user_id = ? AND is_default = 1 AND id <> ?` runs first, then the write. A violation of `UNIQUE (user_id, slug)` becomes `ErrSlugTaken`.
- `Delete(userID, id)` deletes the row and needs no further logic, because no default is fine.
- `Lookup(userSlug, dashboardSlug, viewerID)` loads the owner by slug and the dashboard by `(user_id, slug)`. If the user or dashboard doesn't exist, or the viewer isn't the owner and the dashboard isn't public, it returns `ErrNotFound`. It also returns the switcher entries: all of the owner's dashboards, or `WHERE is_public = 1` for others. Every case of "not visible" maps to the same error, so a response never reveals that a private dashboard exists.
- Titles are trimmed, and 1–100 characters is checked as runes.

### HTTP API
| Method & path | Auth | Notes |
|---|---|---|
| `GET /api/dashboards` | required | own list |
| `POST /api/dashboards` | required | 201 |
| `PATCH /api/dashboards/{id}` | required | partial; unknown/foreign id → 404 |
| `DELETE /api/dashboards/{id}` | required | 204 |
| `GET /api/u/{userSlug}/{dashboardSlug}` | optional | `{dashboard:{slug,title}, ownerSlug, isOwner, dashboards:[{slug,title}]}` |
| `PATCH /api/me` | required | now `name` and/or `slug` |

Error codes: `invalid_slug`, `slug_reserved` (400), `slug_taken` (409), `invalid_title` (400), `not_found` (404). The lookup uses a new `optionalUser` middleware. It shares the session lookup with `requireUser`, but if there is no valid session it continues as anonymous instead of responding 401. It never clears or renews cookies differently from `requireUser`. `PATCH /api/me` with both fields validates both before writing either, and writes them in one `UPDATE`, so a taken slug doesn't half-apply a name change.

### Routing in the web app
- The new file routes are `routes/$userSlug/index.tsx`, which redirects to `/` in `beforeLoad`, and `routes/$userSlug/$dashboardSlug.tsx`. TanStack Router ranks static segments above dynamic ones, so `/login`, `/settings`, `/favourites`, `/dashboards` and `/login/verify` keep winning. Paths with three or more segments fall through to the not-found page as before.
- The dashboard route loads `['dashboard-view', userSlug, dashboardSlug]` (lowercased params) through the lookup endpoint, and renders `NotFound` on a 404.
- The Go side needs no change. The SPA fallback already serves `index.html` for these paths. `/api/` and `/assets/…` are matched before the fallback, and both are reserved anyway.

### Shared `DashboardView`, and `/` without a redirect
`DashboardView` renders the title and the empty placeholder from a lookup result. The dashboard route uses it, and so does `Home`: when `GET /api/dashboards` contains a default, `Home` calls the same lookup with `me.slug` and the default's slug, and renders the same component, so `/` and `/<user>/<slug>` can't diverge. *Alternative:* redirect `/` to the canonical address. That was rejected in exploration, because `/` should stay a stable start-page bookmark.

### Header switcher placement
`AppShell` renders `<DashboardSwitcher />` after the product name. The switcher reads the active dashboard through a small React context, `CurrentDashboardContext`, which `DashboardView` provides (with the `ownerSlug`, the current slug and the entries). `AppShell` wraps the outlet in the provider, and `DashboardView` sets the value in an effect and clears it on unmount. With no value or fewer than two entries, the switcher renders nothing. The switcher is a Base UI `Menu` with `Menu.LinkItem`s rendered as `<Link to="/$userSlug/$dashboardSlug">`, so the URL updates and middle-click works. *Alternative:* parse the route matches in `AppShell` and refetch. That duplicates the lookup and doesn't cover `/`.

### Query cache coherence
Mutations on `/dashboards` invalidate `['dashboards']` and the `['dashboard-view']` prefix. A user-slug change sets the new `me` and also invalidates `['dashboard-view']`. Picking a default on `/` is a `PATCH` with `isDefault: true`, followed by invalidating `['dashboards']`.

### "Create your first dashboard" opens the dialog
`/dashboards` takes a validated search param `create` (boolean). When it is set, the page opens the create dialog on mount and removes the param with `navigate({ replace: true })`, so a reload doesn't reopen it.

### Slug inputs and confirmation
`features/dashboards/slug.ts` exports `normalizeSlugInput` (lowercases while typing), `slugError(value, {reserved})` that returns `invalid | reserved | null`, and `slugFromTitle`, which does the transliteration `ä→ae, ö→oe, ü→ue, ß→ss`, then NFD and strips combining marks, replaces runs of `[^a-z0-9]` with `-`, trims hyphens and cuts to 32 characters, trimming a trailing hyphen again after the cut. A shared `SlugField` component shows the prefix (`/` for the user, `/<user-slug>/` for a dashboard), the validation hint and the "old links stop working" hint when the value differs from the saved one. A shared `ConfirmSlugChangeDialog` shows old and new addresses. Both the Profile tab and the dashboard dialog use them.

## Risks / Trade-offs

- [A future fixed route shadows an existing user slug] → The reserved list is broad from the start. A route added later that hits a taken slug strands that user's addresses until they rename, which the user accepted. The release notes of such a change must say so.
- [Renaming breaks shared links] → The hint and the confirmation make this explicit. Slug history and redirects are deliberately out of scope.
- [Enumerating user slugs] → The lookup responds with the same 404 for an unknown user, an unknown dashboard and a private dashboard, so existence can't be probed beyond what is public. Random 10-character default slugs aren't guessable.
- [`users.slug` nullable in the schema] → The store always sets it, and a migration test asserts no NULLs. A rebuild to `NOT NULL` can come with a future migration that rebuilds `users` anyway.
- [Context-based switcher shows stale entries after edits on another tab] → Mutations invalidate `['dashboard-view']`. Cross-tab staleness is fixed by the normal refetch on window focus.

## Migration Plan

`0006` runs on start like every migration: it adds the column, backfills it, adds the index and creates `dashboards`. Existing users get random slugs and no dashboards. They see the "Create your first dashboard" state at `/`. Rollback means restoring the database backup taken before the upgrade, since there are no down migrations.

## Open Questions

None. The scope decisions were made in exploration: zero or one default, `/` keeps its URL, and the switcher appears only with more than one choice.
