# Design

## Context

See proposal.md for motivation. Floccus has two adapters that matter here (checked against `floccusaddon/floccus` `develop`, `src/lib/adapters/Linkwarden.ts` and `Karakeep.ts`):

- **Linkwarden adapter** calls `GET /api/v1/search` (paged, empty query), `GET/POST /api/v1/collections`, `GET/PUT/DELETE /api/v1/collections/{id}`, `POST /api/v1/links`, `GET/PUT/DELETE /api/v1/links/{id}`. Boardly lacks `GET` and `PUT /collections/{id}`. `updateBookmark` first calls `GET /collections/{parentId}`, so today every bookmark edit fails, not only folder edits. `removeFolder` is one `DELETE` that expects the links to go with it.
- **Karakeep adapter** calls the REST API only: bookmarks (`POST`, `GET ?includeContent=false`, `PATCH`, `DELETE`, `GET …/lists`, `POST/DELETE …/tags`) and lists (`GET`, `POST`, `PATCH`, `DELETE`, `GET …/bookmarks?includeContent=false&cursor=…`, `PUT/DELETE …/bookmarks/{id}`). It reads only HTTP status codes on errors, the `id`/`alreadyExists` fields on create, `lists[].{id,name,parentId}` and `bookmarks[].{id,title,tags[].name,content.{type,url,title}}`, plus `nextCursor`. It doesn't URL-encode the cursor.

Boardly today: `internal/linkwarden` serves the Linkwarden REST API. `internal/karakeep.Service` holds all Karakeep behaviour, and `httpapi/karakeep.go` exposes it only as tRPC procedures (plus the asset REST endpoints, with `karakeepREST` for token checks and `{"error": …}` errors). `favourites.Store.Delete` already supports `DeleteWithChildren` and `DeleteMoveChildren`.

## Goals / Non-Goals

**Goals:**
- A Floccus profile in Linkwarden mode or Karakeep mode, pointed at Boardly with a connected-app token, can sync both ways: create, rename, move and delete bookmarks and folders.
- No duplicated business rules: REST handlers are thin adapters over the existing services.

**Non-Goals:**
- The rest of Karakeep's REST API (highlights, assets via REST v1, users, tags endpoints, search).
- Letting a link sit in several folders (Karakeep allows a bookmark in several lists; Boardly doesn't).
- Floccus-specific settings in the Connected apps dialog. Users create a normal Linkwarden or Karakeep token.

## Decisions

### 1. Karakeep REST as a thin router over `karakeep.Service`

Add `httpapi/karakeep_rest.go`, which registers the named routes on the existing Karakeep mux using `karakeepREST` (token check, `{"error"}` body). Each handler decodes path/query/body, calls the same `karakeep.Service` method as the matching procedure, and maps `karakeep.InputError` → 400, `favourites.ErrNotFound` (and the service's not-found errors) → 404. This reuses `karakeepScope` and the error mapping, with a REST variant next to `karakeepError`.

*Alternative:* translate REST calls into internal tRPC calls. Rejected because of superjson envelopes and batch semantics, which would only add noise.

### 2. Cursor encoding

`Bookmarks` pages with a `{createdAt, id}` cursor. REST encodes it as base64url (no padding) of the cursor's JSON, which is URL-safe because Floccus appends it unencoded. Decoding errors → 400. This lives in the REST layer, not the service.

### 3. Status codes

They follow Karakeep: 201 on create (200 with `alreadyExists: true` for a duplicate URL), 204 for delete/add/remove, 200 otherwise.

### 3a. Removing a bookmark from a list it isn't in

Floccus' `updateBookmark` sends `DELETE /lists/{old}/bookmarks/{id}` and `PUT /lists/{new}/bookmarks/{id}` together (`Promise.all`), and its requests fail on any status of 400 or above. In Karakeep a bookmark can be in both lists for a moment. In Boardly the `PUT` moves the link, so a `DELETE` that arrives later finds it no longer in the old list. Over REST that case is a no-op 204 (the service returns a named `ErrNotInList` that the REST handler swallows). The tRPC procedure keeps its `BAD_REQUEST`.

### 4. Linkwarden collection edit

Add `linkwarden.Service.Collection` (public wrapper of `collectionByID`) and `EditCollection`. `EditCollection` diffs `name` and `parentId` against the current folder, validates the echoed unsupported fields (non-empty `description`, `isPublic`, `members` → 400), and uses the favourites store's rename and move operations (move to end of parent, cycle check). The token root may be "edited" only as a no-op (403 otherwise). This mirrors `karakeep.Service.EditList`.

### 5. Linkwarden delete deletes contents

Switch `DeleteCollection` from `DeleteMoveChildren` to `DeleteWithChildren` (decided with the user). Karakeep's `lists.delete` keeps moving contents up, because Karakeep itself keeps bookmarks when a list is deleted, and the Floccus Karakeep adapter deletes children explicitly before the list.

### 6. Copy

- Landing `features.apps.description` (EN/DE) adds Floccus in the same voice as the card. EN direction: "…and keep your browser bookmarks in sync across browsers and phones with Floccus." DE: "…und halte deine Browser-Lesezeichen mit Floccus auf allen Browsern und Handys synchron." The final wording keeps the card at two or three short sentences.
- `settings.apps.intro` names Floccus with which app type to choose (Linkwarden recommended, see Risks).
- README: a "Floccus" section with setup for both modes. In Linkwarden mode the server URL is `<boardly>/api/linkwarden`, in Karakeep mode `<boardly>/api/karakeep`, and the password is the token secret.

## Risks / Trade-offs

- [Same URL in two browser folders, Karakeep mode] Boardly answers `alreadyExists` and Floccus then moves that one link into the second list, so the two folders fight on every sync. → The README and settings copy recommend **Linkwarden mode** for Floccus (Linkwarden creates duplicates, so each folder keeps its own link). Karakeep mode is documented with this limitation.
- [Breaking Linkwarden delete] Users of the Linkwarden app who delete a collection now lose its links, as in real Linkwarden. → Mention it in the release notes and README. It matches upstream Linkwarden, which is what these apps are built for.
- [Floccus API drift] The adapters may call more endpoints in future versions. → Integration tests replay the exact call sequences listed in Context, and unknown routes stay 404, which makes failures visible.
- [Large trees] Floccus pages Karakeep lists at 20 per call by default. → Acceptable. `limit` up to 100 is supported if Floccus passes it.

## Migration Plan

No data migration. Ship as a normal release. Rollback is reverting the change, which also restores the old delete behaviour.
