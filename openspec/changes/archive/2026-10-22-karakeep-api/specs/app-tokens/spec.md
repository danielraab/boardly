# Spec Delta

## MODIFIED Requirements

### Requirement: Karakeep tRPC transport
The Karakeep browser extension and mobile app talk tRPC (version 11, `httpBatchLink`, `superjson` transformer) to `<server address>/api/trpc`. Famodash SHALL answer these calls at `/api/karakeep/api/trpc/<procedures>?batch=1`, where `<procedures>` is a comma-separated list of procedure names such as `apiKeys.validate`. Queries are sent with `GET` and their inputs in the `input` query parameter; mutations are sent with `POST` and their inputs in the JSON body. In both cases the inputs are a JSON object keyed by the call's index (`"0"`, `"1"`, …), each value a superjson envelope `{"json": <value>, "meta"?: …}`; a missing key means no input. Dates in an input arrive as ISO 8601 strings in `json`, and Famodash SHALL accept them as such; `meta` is not needed to read an input. The response SHALL be a JSON array with one entry per procedure, in the request's order: `{"result": {"data": {"json": <output>}}}` on success, or `{"error": {"json": {"message", "code", "data": {"code", "httpStatus", "path"}}}}` on failure, where `code` is the JSON-RPC number of the tRPC error code (for example `-32001` for `UNAUTHORIZED`, `-32004` for `NOT_FOUND`). When an output contains dates or maps, the result SHALL also carry superjson's `meta` (`{"json": …, "meta": {"values": {<path>: ["Date"] | ["map"]}}}`), with dates as ISO 8601 strings and maps as arrays of `[key, value]` pairs in `json`, so the clients get real `Date` and `Map` values. The HTTP status SHALL be the common status of all entries, or 207 when they differ. Only the procedures named in this and later changes are implemented; any other procedure answers with a `NOT_FOUND` error entry. A query sent with `POST`, or a mutation sent with `GET`, answers with a `METHOD_NOT_SUPPORTED` error entry. A request without `batch=1`, or whose `input` isn't valid JSON, answers 400 with a single `BAD_REQUEST` error entry.

#### Scenario: Batched call
- **WHEN** a client sends `GET /api/karakeep/api/trpc/users.whoami,config.clientConfig?batch=1&input={}` with a valid token
- **THEN** the server responds 200 with an array of two `result` entries, the user first and the configuration second

#### Scenario: Unknown procedure
- **WHEN** a client calls `bookmarks.summarizeBookmark`
- **THEN** its entry is an error with `data.code` `NOT_FOUND`, `code` `-32004` and `data.path` `bookmarks.summarizeBookmark`

#### Scenario: Mixed results
- **WHEN** one call of a batch succeeds and another fails with `NOT_FOUND`
- **THEN** the server responds 207 and each entry carries its own result or error

#### Scenario: Date in an output
- **WHEN** a client calls `bookmarks.getBookmark` for a link created on 1 March 2026 at 12:00 UTC
- **THEN** the result's `json.createdAt` is `"2026-03-01T12:00:00.000Z"` and `meta.values` maps `createdAt` to `["Date"]`

