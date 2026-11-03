## ADDED Requirements

### Requirement: OpenAPI documents
Boardly SHALL describe its HTTP API in four OpenAPI 3.1 documents written in YAML:

- **Boardly API** describes Boardly's own API under `/api`. Its server URL is `/api` and its paths are relative to it. It SHALL describe for every operation the path and query parameters, the request body, the success responses with their schemas, and the error responses in the shared `{"error": {"code", "message"}}` format. It SHALL declare the session cookie as the security scheme and mark the operations that are public.
- **Karakeep, Linkwarden and Memos API** are one document per connected app. Each one's server URL is the app's prefix (`/api/karakeep`, `/api/linkwarden`, `/api/memos`), so its paths read as in the upstream product (for example `/api/v1/bookmarks`). Each SHALL declare the Bearer app token for that app as its security scheme, mark the public operations, describe the app's error format once, and link to the upstream product's API documentation. Each operation SHALL have a summary and its path parameters. Request and response schemas are optional.

The Boardly document SHALL link to the three app documents by their served URLs, in its description and through a "Connected apps" tag with `externalDocs`. It SHALL NOT include their paths.

#### Scenario: Paths relative to the app prefix
- **WHEN** someone opens the Karakeep document
- **THEN** the bookmark endpoint is listed as `/api/v1/bookmarks` under the server `/api/karakeep`, and its security is a Bearer token

#### Scenario: Find the app documents
- **WHEN** someone reads the Boardly document
- **THEN** its description and the "Connected apps" tag link to `/api/karakeep/openapi.yaml`, `/api/linkwarden/openapi.yaml` and `/api/memos/openapi.yaml`

### Requirement: Karakeep tRPC in the document
The Karakeep document SHALL describe the tRPC API as one path, `/api/trpc/{procedure}`, with `GET` for queries and `POST` for mutations. The `procedure` parameter SHALL list every implemented tRPC procedure as an enum. The description SHALL explain the `input` query parameter, the superjson body and batching.

#### Scenario: Procedure list
- **WHEN** someone opens the tRPC path in the Karakeep document
- **THEN** the `procedure` parameter lists `bookmarks.getBookmarks`, `lists.list` and every other procedure Boardly implements

### Requirement: Serve the documents
The server SHALL serve the documents, embedded in the binary, without authentication and with the content type `application/yaml`:

- `GET /api/openapi.yaml` serves the Boardly document
- `GET /api/karakeep/openapi.yaml` serves the Karakeep document
- `GET /api/linkwarden/openapi.yaml` serves the Linkwarden document
- `GET /api/memos/openapi.yaml` serves the Memos document

The app documents are an exception to the rule in `karakeep-api`, `linkwarden-api` and `memos-api` that paths those specs don't name respond 404. They are served like the other app routes, with `Access-Control-Allow-Origin: *`. Boardly SHALL NOT serve a documentation UI and SHALL NOT load anything from third parties to show the documents.

#### Scenario: Download the main document
- **WHEN** a signed-out client calls `GET /api/openapi.yaml`
- **THEN** the server responds 200 with `application/yaml` and the Boardly document

#### Scenario: Download an app document without a token
- **WHEN** a client without an app token calls `GET /api/memos/openapi.yaml`
- **THEN** the server responds 200 with the Memos document

### Requirement: Documents match the routes
Every route the server registers SHALL appear in its document, and every operation in a document SHALL be a registered route. A route is a method and a path pattern. Its document is chosen by the route's prefix, and the prefix is removed before comparing. Routes without a method (the 404 catch-alls), the `HEAD` requests that `GET` routes answer, and the CORS preflight of the app APIs are not compared. Karakeep's tRPC prefix is compared with `/api/trpc/{procedure}`, and that path's `procedure` enum SHALL equal the set of implemented procedures. Some paths are handled inside another route, such as Memos' `users/{id}:getStats` inside `users/{id}`. Such a path SHALL appear in the document as its own path, and the test maps it to the route that handles it. A Go test SHALL check all of this and name each mismatch.

#### Scenario: Undocumented route
- **WHEN** a developer registers `GET /api/widgets/{id}` without adding it to the Boardly document
- **THEN** the test fails and names `get /widgets/{id}` as missing from the Boardly document

#### Scenario: Stale document
- **WHEN** a route is removed but the Boardly document still lists it
- **THEN** the test fails and names the operation as not registered

#### Scenario: New tRPC procedure
- **WHEN** a developer adds a Karakeep tRPC procedure without adding it to the `procedure` enum
- **THEN** the test fails and names the procedure
