# OpenAPI documents

These OpenAPI 3.1 documents describe every route the server registers. They are embedded in the
binary and served without sign-in:

| File              | Served at                      | Describes                                  |
| ----------------- | ------------------------------ | ------------------------------------------ |
| `boardly.yaml`    | `/api/openapi.yaml`            | Boardly's own API (session cookie)         |
| `karakeep.yaml`   | `/api/karakeep/openapi.yaml`   | The Karakeep API Boardly implements        |
| `linkwarden.yaml` | `/api/linkwarden/openapi.yaml` | The Linkwarden API Boardly implements      |
| `memos.yaml`      | `/api/memos/openapi.yaml`      | The Memos API Boardly implements           |

Each document's server URL is its prefix (`/api`, `/api/karakeep`, …), so paths are written
without it. `boardly.yaml` links to the app documents instead of including them.

When you add, remove or rename a route, update the matching document. `openapi_test.go` checks
that the documents list exactly the registered routes (and, for Karakeep, the tRPC procedures),
that they are OpenAPI 3.1, and that every local `$ref` resolves. It doesn't check request and
response schemas against the handlers, so update those by hand when a shape changes.

`boardly.yaml` describes request and response bodies in full. The app documents describe each
endpoint briefly and refer to the upstream project's API for the shapes.
