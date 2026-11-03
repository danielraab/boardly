## Why

Boardly's HTTP API (about 70 routes under `/api` plus the Karakeep, Linkwarden and Memos compatibility APIs) is only described in prose specs and Go code. Self-hosters who want to script against their instance, and people connecting third-party clients, have no machine-readable description of what exists, how it authenticates and what it returns. An OpenAPI description fixes that, and a test that ties it to the routers keeps it from going stale.

## What Changes

- Add hand-written OpenAPI 3.1 documents in YAML:
  - `boardly.yaml` for Boardly's own API under `/api`: every route with parameters, request and response schemas, the session-cookie auth and the shared error format.
  - `karakeep.yaml`, `linkwarden.yaml` and `memos.yaml`, one per connected app. Each has its own server prefix (`/api/karakeep` and so on), so paths read as in the upstream product, and its own Bearer app-token auth. They list every endpoint Boardly implements, with lighter detail than the main document, and link to the upstream product's API docs.
  - `boardly.yaml` links to the three app documents. They are separate documents, not `$ref`ed into the main one.
  - Karakeep's tRPC API is documented as one generic path, `/api/trpc/{procedure}`, whose `procedure` parameter lists the supported procedures.
- Embed the documents in the binary and serve them unauthenticated: `GET /api/openapi.yaml`, `GET /api/karakeep/openapi.yaml`, `GET /api/linkwarden/openapi.yaml` and `GET /api/memos/openapi.yaml`. There's no bundled docs UI and nothing is loaded from a third-party CDN.
- Add a drift test that fails when a registered route is missing from its document, or when a document lists a route that isn't registered. The test also checks the tRPC procedure list against the implemented procedures. To make this possible, the API's muxes record the patterns registered on them.
- Add `go.yaml.in/yaml/v3`, used only by the test to parse the documents.

## Capabilities

### New Capabilities
- `api-docs`: the OpenAPI documents for Boardly's API and the connected-app APIs, how they are split and linked, where they are served, and the requirement that they list exactly the routes the server registers.

### Modified Capabilities
None. The four documentation routes under the app prefixes are an exception to the app specs' "any other path responds 404" rules, and that exception is stated in `api-docs`.

## Impact

- Code: `backend/internal/httpapi` (route-recording mux, the main mux moved into its own builder, the four spec routes, the drift test), plus a new embedded `openapi/` directory with the YAML files.
- APIs: four new public `GET` routes. No existing route changes.
- Dependencies: `go.yaml.in/yaml/v3` (used by tests only).
- Process: every later change that adds, removes or renames a route has to update the matching document, or the drift test fails.
- Export and import are unaffected, because no user data changes.
