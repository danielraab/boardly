## Context

`backend/internal/httpapi/router.go` builds one `http.ServeMux` inline in `New` for `/api/*` (about 70 routes, session cookie plus the Origin check). Three more muxes come from `a.karakeepAPI()`, `a.linkwardenAPI()` and `a.memosAPI()`, mounted under their prefixes through `appAPI` (Bearer app token, CORS `*`). Those builders register routes through local helpers (`handle`, `respond`, `public`, `karakeepRESTv1(mux)`), and each app mux ends in a method-less catch-all that answers 404 in the app's error format. Karakeep also mounts a tRPC server on the prefix `…/api/trpc/` with 29 procedures from `karakeepProcedures()`.

There's no API description today. The routes use the standard library mux without annotations, and `http.ServeMux` can't list the patterns registered on it.

## Goals / Non-Goals

**Goals:**
- One accurate OpenAPI document for Boardly's API and one per connected app, served from the binary.
- A test that fails as soon as the documents and the registered routes disagree.

**Non-Goals:**
- Checking request and response bodies against the schemas. That's a possible follow-up, for example with kin-openapi's `openapi3filter` in the handler tests.
- A documentation UI (Swagger UI, Scalar, Redoc).
- Generating the frontend client or Go code from the documents.
- Full schemas for the connected apps. Their upstream projects document the shapes, and Boardly's specs describe how Boardly differs.

## Decisions

### Hand-written YAML, not generated
Generating the documents would need a framework that knows the handlers' types (huma, ogen, swaggo annotations), which means rewriting or annotating every handler. Hand-written YAML plus the drift test keeps the router as it is. YAML is chosen over JSON because it's far easier to write and review by hand, at the cost of a test-only dependency.

### Four separate documents, linked rather than `$ref`ed
Each connected app is a different API: it has its own base URL that clients are configured with, its own auth and its own error format. Separate documents can declare their own `servers` and `securitySchemes` and keep the upstream path names. Pulling the app paths into the main document through `$ref` would mean rewriting every path with the prefix, mixing three auth schemes and error formats, and keeping about 80 cross-file references in step. The main document links to the app documents in its `info.description` and through a "Connected apps" tag with `externalDocs`. Viewers that take several URLs (for example Swagger UI's `urls`) show them as one set.

### Files live in `backend/internal/httpapi/openapi/`
`go:embed` can only reach files inside the package directory, and `httpapi` serves them. The files are `boardly.yaml`, `karakeep.yaml`, `linkwarden.yaml` and `memos.yaml`, plus a short `README.md` that says how to edit them and that the drift test enforces them.

### Serving
- `GET /api/openapi.yaml` is registered on the main mux.
- `GET /api/{app}/openapi.yaml` is registered on each app's mux. A pattern on the main mux would never be reached, because the root mux sends `/api/karakeep/…` to the app handler. As a result, these responses get the app's CORS header for free, and a token is not needed because the handler doesn't ask for one.
- The responses use `Content-Type: application/yaml` and `Cache-Control: no-cache`. The bytes change only with a new binary.
- The main document links to the app documents with relative URLs (`karakeep/openapi.yaml`), so the links work behind any host.

### Route recording: `routeMux`
A small type in `httpapi`:

```go
type routeMux struct {
	*http.ServeMux
	patterns []string
}
// Handle and HandleFunc record the pattern, then delegate.
```

All four muxes are created with `newRouteMux()`. The app builders and `appAPI` take or return `*routeMux` instead of `*http.ServeMux`. The local helpers already go through `mux.Handle`, so every route gets recorded. The inline main-mux block in `New` moves into `func (a *api) mainAPI(db Pinger) *routeMux`. `New` stays the only production entry point, and the recorded patterns are read only by the test.

Alternatives considered:
- **Parsing the Go source for patterns:** this breaks on concatenation such as `"GET "+v1+"/links"` and on helpers.
- **Probing the mux with `mux.Handler(req)` for each documented path:** this only checks one direction and can't find routes that aren't documented.

### Drift test (`openapi_test.go`)
1. Build an `api` the way the existing router tests do, and call `mainAPI(fakeDB{})`, `karakeepAPI()`, `linkwardenAPI()` and `memosAPI()`.
2. Normalize each recorded pattern:
   - Skip patterns without a method (the catch-alls).
   - Map Karakeep's tRPC prefix to `post /api/trpc/{procedure}` and `get /api/trpc/{procedure}`.
   - Remove the mux's prefix: `/api` for main, the app prefix for the apps.
   - Lower-case the method.
3. Parse each YAML file with `go.yaml.in/yaml/v3` and collect `method path` for every operation under `paths`. A small alias table maps paths that are handled inside another route to the pattern that handles them, for example Memos `get /api/v1/users/{id}:getStats` → `get /api/v1/users/{id}`.
4. Compare the two sets in both directions, per document, and report the sorted differences with `t.Errorf`.
5. Compare the tRPC `procedure` enum with the keys of `karakeepProcedures()`.
6. Light document checks: `openapi` is `3.1.x`, and every local `$ref` (`#/components/...`) resolves.

Go's wildcards (`{id}`) use the same syntax as OpenAPI's path templates, so they compare as plain strings, and a renamed wildcard counts as a mismatch. The test does not compare `HEAD` or `OPTIONS`.

A second, ordinary handler test checks the four `GET …/openapi.yaml` routes (status, content type, body equal to the embedded file, no auth needed).

### Dependency
`go.yaml.in/yaml/v3` is the maintained successor of `gopkg.in/yaml.v3`. It's imported only from `_test.go` files, so it isn't linked into the binary.

## Risks / Trade-offs

- [Schemas in `boardly.yaml` can drift from handler output without failing the test] → Reviewers check schema edits along with handler changes. Validating responses is the noted follow-up.
- [Writing about 70 fully described operations is a lot of up-front work] → Shared shapes and errors go in `components`. Operations go in by area in the same order as `router.go`, so a reviewer can walk both side by side.
- [The alias table in the test can hide a real mismatch] → It stays tiny, each entry has a comment, and it maps to a specific pattern, not a wildcard.
- [Exposing the documents publicly reveals the API surface] → The routes are already discoverable from the open-source code and the frontend bundle, and the documents contain no instance data.
