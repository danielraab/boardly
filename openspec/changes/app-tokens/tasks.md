# Tasks

## 1. Database

- [ ] 1.1 Add migration `0011_app_tokens.sql` (table, indexes, `app` CHECK, `root_id` FK with `ON DELETE SET NULL`); verify with a `db` test that the migration applies and that deleting a referenced folder nulls `root_id` and keeps the row

## 2. Token store (`internal/apptokens`)

- [ ] 2.1 Implement secret generation (`famo_` + `rand.Text()`), SHA-256 hashing and the 12-character prefix; verify with a unit test of the format and prefix
- [ ] 2.2 Implement `Create` with validation (`invalid_name`, `invalid_app`, `invalid_root` for links, unknown nodes and other users' folders, `invalid_expiry` for past or invalid times); verify with table-driven store tests
- [ ] 2.3 Implement `List` (newest first, derived `status`, never secret or hash) and `Revoke`/`Delete` with `token_inactive`, `token_active` and not-found errors; verify with store tests covering each status
- [ ] 2.4 Implement `Authenticate(secret, app, now)` returning a single `ErrInvalidToken` for unknown, revoked, expired and wrong-app tokens, and touching `last_used_at` at most once per minute; verify with store tests, including the throttle

## 3. Favourites integration

- [ ] 3.1 Add `favourites.Scope` with `Contains` (recursive CTE) and guards for create, move (both ends inside) and delete, rename or move of the scope root; verify with unit tests for root, descendant, outside node, other user's node and whole-tree scope
- [ ] 3.2 Revoke affected active tokens in `favourites.Store.Delete` within its transaction (whole subtree for delete-with-children, the folder only for move-children, and never expired ones); verify with favourites tests for all three delete modes and for an expired token

## 4. Session API

- [ ] 4.1 Add handlers and routes `GET/POST /api/app-tokens`, `POST /api/app-tokens/{id}/revoke`, `DELETE /api/app-tokens/{id}` with `requireUser`; verify with httptest that the secret appears only in the 201 response, that error codes and statuses match the spec, that another user's token gets 404 and that `PATCH` gets 405

## 5. App API (bearer)

- [ ] 5.1 Add the `/api/karakeep/` router branch with CORS (preflight 204 before auth) and bearer middleware that ignores cookies and skips `checkCSRF`, with JSON 404/405; verify with httptest: valid token 200, cookie-only 401, revoked/expired/unknown/wrong-app 401 with identical bodies, preflight from `chrome-extension://` origin 204 with the expected headers, and existing `/api/` CSRF behaviour unchanged
- [ ] 5.2 Add `GET /api/karakeep/api/v1/users/me` returning `id`, `name`, `email`; verify with an httptest that checks the response and the updated `last_used_at`

## 6. Frontend

- [ ] 6.1 Add `features/settings/appTokens/api.ts` (types, query options, create/revoke/delete mutations) and extract `pathsOf` into `features/favourites/tree.ts`; verify that `pnpm check` passes and the existing `FavouritePicker` tests still pass
- [ ] 6.2 Add the "Connected apps" tab (`?tab=apps`) with list, status badges, root path or "Deleted folder", empty state, and revoke and delete confirmations; verify with Vitest tests for the tab after reload, actions by status and cancelled revoke
- [ ] 6.3 Add the create dialog (name, app preselected, folder picker in folder mode, expiry presets with 90 days as default and a future-only custom date) with a secret step (copy button, server address, "shown only once" note); verify with Vitest tests for the default expiry, copying, and the secret gone after closing
- [ ] 6.4 Show the connected-apps warning in the favourites `DeleteDialog`, counting active tokens per delete mode; verify with Vitest tests for a folder used by two tokens, a subfolder root with each delete mode, and no warning when none is affected
- [ ] 6.5 Add all new strings in en and de under `settings.apps.*` and `favourites.delete.*`; verify that the i18n key-parity test passes

## 7. Docs and checks

- [ ] 7.1 Update README ("Karakeep-compatible server", "Concept overview", settings tabs) to describe connected apps, the server address `<base>/api/karakeep` and the token lifecycle; verify by reading it against the specs
- [ ] 7.2 Run `make check` and `make test`; verify that both pass
