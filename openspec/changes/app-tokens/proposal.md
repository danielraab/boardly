# Proposal

## Why

Famodash promises Karakeep compatibility so the Karakeep browser extension and mobile app can save straight into the favourites tree, and later the same for Linkwarden clients. Those clients authenticate with a bearer API key, not a session cookie, and each connection must only see the part of the tree it was given. Before any Karakeep endpoint can be built, users need a way to create, inspect, revoke and delete such keys, and the server needs a safe way to authenticate them.

## What Changes

- New settings tab **"Connected apps" / "Verbundene Apps"** (`?tab=apps`, after Profile and Passkeys) listing the user's app tokens with name, app (Karakeep), root folder, public prefix, dates and status (active, expired, revoked).
- Creating a token asks for a name, the app (only Karakeep for now; Linkwarden follows later), the root folder in the favourites tree (any folder, including the top-level one) and an optional expiry (30 days, 90 days, 1 year, never or a custom date). The secret is shown **once**, with a copy button and the server address to enter in the client.
- Tokens are **immutable**: nothing can be edited after creation. An active token can be revoked (final). Only revoked or expired tokens can be deleted.
- New session-authenticated API: `GET /api/app-tokens`, `POST /api/app-tokens`, `POST /api/app-tokens/{id}/revoke`, `DELETE /api/app-tokens/{id}`.
- New bearer-authenticated API area under `/api/karakeep/` (and later `/api/linkwarden/`): authenticated only by `Authorization: Bearer`, never by the session cookie, exempt from the cross-site check, with CORS for browser extensions. A token only works for its own app.
- The Karakeep extension and mobile app don't use Karakeep's REST API but its internal **tRPC** API (`<address>/api/trpc`, batched, superjson). Famodash implements the tRPC wire format at `/api/karakeep/api/trpc` and, in this change, only the procedures needed to log in: `apiKeys.validate`, `apiKeys.exchange` (always refused: Famodash has no passwords), `users.whoami` and `config.clientConfig`, plus the mobile app's status endpoints `api/health` and `api/version`. Only what these two clients call is implemented, nothing more; the REST API (`/api/v1`) is not.
- Server-side subtree scope for token requests: a token can only read and change nodes in its root folder's subtree. The Karakeep endpoints in the follow-up change `karakeep-api` build on it.
- Deleting a folder that is the root of active tokens, or contains one, asks for confirmation ("used by N connected apps; they will be revoked") and revokes those tokens. The token rows stay and show "Deleted folder" as their root.
- README updated to describe connected apps.

Out of scope: the bookmark, list and tag procedures the clients use for their core features (change `karakeep-api`); Karakeep features Famodash doesn't have (highlights, reading progress, AI summaries, assets, list sharing, account deletion), which answer with a tRPC error; Karakeep's REST API; Linkwarden.

## Capabilities

### New Capabilities
- `app-tokens`: Managing app tokens in the settings (create, list, revoke, delete, lifecycle, immutability, one-time secret) and authenticating app requests with them (bearer auth, app-specific endpoints, subtree scope, CORS, the tRPC transport, login procedures and status endpoints).

### Modified Capabilities
- `user-profile`: The settings page gets the third tab "Connected apps" / "Verbundene Apps" (`?tab=apps`).
- `favourites`: Deleting a folder that is, or contains, the root of an active app token warns about it and revokes those tokens.

## Impact

- **Backend**: new migration (`app_tokens` table), new `apptokens` package (store, secret generation and hashing, lookup), new handlers in `httpapi`, a separate router branch for `/api/karakeep/` with bearer auth and CORS that bypasses the cookie session and `checkCSRF`, a small tRPC wire-format layer (batching, superjson envelope, tRPC errors) with a procedure registry, a subtree-scope helper in `favourites`, and token revocation in `favourites.Store.Delete`.
- **Frontend**: new `ConnectedAppsTab` in `features/settings` with a create dialog (reuses `FavouritePicker` in folder mode), a one-time secret dialog, revoke and delete confirmations, and a warning in the favourites delete dialog. New en and de translations.
- **API**: four new session endpoints; under `/api/karakeep/` four tRPC procedures and two status endpoints. Existing endpoints are unchanged.
- **Docs**: README sections "Karakeep-compatible server" and "Concept overview".
