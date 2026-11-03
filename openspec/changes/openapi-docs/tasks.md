## 1. Route recording

- [x] 1.1 Add `routeMux` (an embedded `*http.ServeMux` plus the recorded patterns; `Handle` and `HandleFunc` record, then delegate) and `newRouteMux()` in `backend/internal/httpapi`
- [x] 1.2 Move the inline `/api` route block from `New` into `func (a *api) mainAPI() *routeMux`, and have `New` use it
- [x] 1.3 Switch `karakeepAPI`, `karakeepRESTv1`, `linkwardenAPI`, `memosAPI` and `appAPI` to `*routeMux`
- [x] 1.4 Run the existing `httpapi` tests unchanged to confirm routing still behaves the same

## 2. Serving the documents

- [x] 2.1 Create `backend/internal/httpapi/openapi/` with placeholder `boardly.yaml`, `karakeep.yaml`, `linkwarden.yaml`, `memos.yaml` and a `README.md` (how to edit them, and that the drift test enforces them)
- [x] 2.2 Embed the directory and add a handler that serves one file with `Content-Type: application/yaml` and `Cache-Control: no-cache`
- [x] 2.3 Register `GET /api/openapi.yaml` on the main mux and `GET <prefix>/openapi.yaml` on each app mux
- [x] 2.4 Test the four routes: 200, content type, body equals the embedded file, works without a session or token, and the app ones carry `Access-Control-Allow-Origin: *`

## 3. Drift test

- [x] 3.1 Add `go.yaml.in/yaml/v3` to `backend/go.mod`, used by tests only
- [x] 3.2 Write `openapi_test.go`: collect normalized `method path` pairs from the four route muxes (skip method-less catch-alls, map the tRPC prefix to `/api/trpc/{procedure}`, strip the prefix, lower-case the method)
- [x] 3.3 Parse each YAML document into `method path` pairs, apply the commented alias table (Memos `users/{id}:getStats` → `users/{id}`), and report both directions of the difference per document
- [x] 3.4 Check that the tRPC `procedure` enum equals the keys of `karakeepProcedures()`
- [x] 3.5 Check that each document declares `openapi: 3.1.x` and that every local `$ref` resolves
- [x] 3.6 Confirm the test fails with the placeholder documents and lists every route, which checks the test itself

## 4. Boardly document (`boardly.yaml`)

- [x] 4.1 Write `info` (with links to the three app documents), `servers: /api`, the session-cookie security scheme, the error schema and the shared error responses, and the "Connected apps" tag with `externalDocs`
- [x] 4.2 Document health, auth (magic link, passkey login, logout), `me` (get, patch, delete, restore, export), import, and passkeys
- [x] 4.3 Document favourites (including bulk, move, import), tags and favicons
- [x] 4.4 Document dashboards, layout and the public dashboard view `/u/{userSlug}/{dashboardSlug}`
- [x] 4.5 Document notes and note categories
- [x] 4.6 Document media (including order, station search and search logo), images and attachments
- [x] 4.7 Document app tokens and `/openapi.yaml` itself
- [x] 4.8 Cross-check request and response schemas against the handlers and the existing specs, and mark the public operations with `security: []`

## 5. Connected-app documents

- [x] 5.1 `karakeep.yaml`: `servers: /api/karakeep`, the Bearer scheme, the error format and an upstream link; health, version, assets, the REST v1 bookmarks, lists and users endpoints; `/api/trpc/{procedure}` (GET and POST) with the procedure enum and a description of `input`, superjson and batching; `/openapi.yaml`
- [x] 5.2 `linkwarden.yaml`: `servers: /api/linkwarden`, the Bearer scheme, the `{"response"}` and `{"data"}` envelopes and an upstream link; config, logins, session, users/me, v2 dashboard, search, links, collections, tags, archives; `/openapi.yaml`
- [x] 5.3 `memos.yaml`: `servers: /api/memos`, the Bearer scheme, the gRPC-style error format and an upstream link; instance profile, auth/me, users (including `:getStats` and settings), memos, attachments; `/openapi.yaml`

## 6. Wrap-up

- [x] 6.1 Run the drift test until it passes with no alias entries beyond the documented ones
- [ ] 6.2 Validate all four documents once with an OpenAPI linter, run locally and not added to the project
- [x] 6.3 Mention the documents and their URLs in `README.md` and add a `CHANGELOG.md` entry
- [x] 6.4 Add a note to `CLAUDE.md` that route changes have to update the matching OpenAPI document
- [x] 6.5 `make check` and `make test` pass
