# Tasks

## 1. Data model

- [x] 1.1 Add migration `0006_dashboards.sql`: `users.slug` with a hex backfill and the unique index `users_slug`; a `dashboards` table with `UNIQUE (user_id, slug)`, the lowercase CHECK, the 0/1 CHECKs and the partial unique index `dashboards_default`. Verify migration tests that existing users get distinct 10-character slugs and none is NULL, that a second default per user and a duplicate `(user_id, slug)` are rejected, that an uppercase slug is rejected, and that deleting a user removes their dashboards

## 2. Slugs (`internal/slugs`) and user slugs

- [x] 2.1 Implement `Normalize`, `IsReserved` with the `Reserved` list, and `Generate`, and add the shared fixture `reserved-slugs.json` that a Go test compares against `Reserved`. Verify table tests for valid and invalid slugs (`-a`, `a-`, `a_b`, 32 vs 33 characters, uppercase lowercased) and that `Generate` returns 10 characters from `[a-z0-9]`
- [x] 2.2 Generate a slug in `users.EnsureByEmail` (skip reserved words, append `-2`, `-3`, … on collision), add `Slug` to `User`, and add `SetSlug` with `ErrInvalidSlug`, `ErrSlugReserved` and `ErrSlugTaken`. Verify tests with an injectable generator: the collision path produces `x-2` and then `x-3`, and `SetSlug` covers taken, reserved and uppercase input
- [x] 2.3 Extend `PATCH /api/me` to accept `name` and/or `slug` (both validated before one write; neither → `invalid_request`) and return `slug` from `GET /api/me`. Verify `httptest` tests for `invalid_slug`, `slug_reserved` (`"API"`), 409 `slug_taken`, and that a taken slug leaves the name unchanged when both are sent

## 3. Dashboards store and API

- [x] 3.1 Implement `internal/dashboards` with `List`, `Create`, `Update` (default switch in one transaction) and `Delete`, plus title validation. Verify tests for sorting by title ignoring case, switching the default, clearing the default, deleting the default (others remain, no default), `slug_taken` per user only, and a foreign ID → not found
- [x] 3.2 Implement `Lookup(userSlug, dashboardSlug, viewerID)` with the visibility rules and the switcher entries. Verify tests for owner/private → found with all entries, visitor/public → found with only public entries, visitor/private, unknown user and unknown dashboard → the same `ErrNotFound`, and mixed-case input
- [x] 3.3 Add the `optionalUser` middleware and the handlers in `internal/httpapi/dashboards.go` for `GET/POST /api/dashboards`, `PATCH/DELETE /api/dashboards/{id}` and `GET /api/u/{userSlug}/{dashboardSlug}`. Verify `httptest` tests for each endpoint, 401 on the management endpoints without a session, the lookup with and without a session, and an invalid session cookie on the lookup → treated as anonymous

## 4. Shared frontend pieces

- [x] 4.1 Add `features/dashboards/api.ts` (types, query options `['dashboards']` and `['dashboard-view', …]`, mutations with invalidation) and `slug.ts` (`normalizeSlugInput`, `slugError`, `slugFromTitle`, reserved list), and add `slug` to the `User` type. Verify unit tests for `slugFromTitle` ("Küche & Haus" → `kueche-haus`, accents, a cut at 32 without a trailing hyphen), `slugError`, and that the reserved list equals the shared fixture
- [x] 4.2 Add `SlugField` (prefix, lowercasing while typing, format/reserved hint, rename hint when it differs from the saved value) and `ConfirmSlugChangeDialog` (old and new address). Verify component tests for lowercasing, the reserved hint blocking submit and the rename hint appearing only when the value changes

## 5. Dashboards page

- [x] 5.1 Add the route `/dashboards` with the signed-in guard and the validated `create` search param, plus the list (title, address link, "Default" and "Public" badges) and an empty state. Verify tests for the redirect to `/login` when signed out, the badges, the empty state, and that `?create=true` opens the dialog and is removed from the URL
- [x] 5.2 Add the create/edit dialog (title, slug following the title until edited, default and public switches, server errors at the field, rename confirmation on edit) and the delete confirmation. Verify tests for create, a manual slug being kept, `slug_taken` shown at the field, a confirmed and a cancelled rename, and delete

## 6. Dashboard view, routing and switcher

- [x] 6.1 Add `DashboardView` (title and empty placeholder) and `CurrentDashboardContext`, the routes `/$userSlug/$dashboardSlug` (lookup and `NotFound` on 404) and `/$userSlug` (redirect to `/`). Verify routing tests: owner and visitor see a public dashboard, a private one shows not found for others, `/Ann/Work` works, `/ann` ends on `/`, and `/login/verify` and `/settings` still render their own pages
- [x] 6.2 Add `DashboardSwitcher` to `AppShell` after "Famodash /" (Base UI `Menu` with `Link` items, current entry marked, hidden with fewer than two entries or off dashboard pages). Verify tests that picking an entry changes the URL and the title, that a visitor only sees public entries, that there's no switcher with one entry, and none on `/settings`

## 7. Home page, user menu and profile

- [x] 7.1 Rewrite `Home` for the three states: default (`DashboardView` through the lookup, URL stays `/`), no dashboards (greeting and "Create your first dashboard" → `/dashboards?create=true`), and no default (greeting and "Select your default dashboard" dropdown that sets the default). Verify updated home tests for all three states, including the name/email greeting fallback and that picking a default shows that dashboard
- [x] 7.2 Add the "Dashboards" item (`LayoutGrid` icon, `shell.dashboards`) above "Favourites" in `UserMenu.tsx`. Verify the updated app-shell tests for item order with and without a name, and that the link closes the menu and shows `/dashboards`
- [x] 7.3 Add the slug field with the rename confirmation to the Settings Profile tab, saved through `PATCH /api/me`. Verify tests for a confirmed change (the `me` cache is updated), a cancelled confirmation, the reserved hint, and `slug_taken` shown at the field
- [x] 7.4 Add all new `dashboards.*`, `home.*`, `settings.*` and `shell.*` translation keys in English and German. Verify that the i18n key-parity check passes

## 8. Integration and docs

- [x] 8.1 Update the README's dashboards section (slugs, addresses `/<user-slug>/<dashboard-slug>`, default and public, the reserved user slugs, renaming breaks links). Verify the flow in dev: change the user slug, create two dashboards, set a default, switch through the header, open a public dashboard in a private window, and confirm a private one shows not found there
- [x] 8.2 Run `make check` and `make test` and verify both pass

## Workflow follow-up

- Archive with `/opsx:archive` after review.
