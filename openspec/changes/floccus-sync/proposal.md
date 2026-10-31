# Proposal

## Why

[Floccus](https://floccus.org) syncs browser bookmarks across browsers and devices (browser extensions and mobile apps), and can use a Linkwarden or a Karakeep server as its sync backend. Boardly speaks both APIs, but not the parts Floccus calls: through the Linkwarden adapter, editing a synced bookmark or renaming/moving a folder fails, and the Karakeep adapter uses Karakeep's REST API, which Boardly doesn't serve at all (Boardly only implements Karakeep's tRPC procedures). Supporting Floccus turns Boardly's favourites tree into a home for the browser's own bookmarks.

## What Changes

- **Linkwarden API**: add `GET /api/v1/collections/{id}` and `PUT /api/v1/collections/{id}` (rename and move a collection).
- **Linkwarden API**: **BREAKING** `DELETE /api/v1/collections/{id}` deletes the folder together with its links and subfolders, as Linkwarden does, instead of moving the contents to the parent. Otherwise deleted browser folders would come back as loose links on the next Floccus sync. This also applies to the Linkwarden extension and mobile app.
- **Karakeep API**: add the REST endpoints under `/api/karakeep/api/v1/` that the Floccus Karakeep adapter calls: bookmarks (create, get, update, delete, the lists of a bookmark, attach/detach tags) and lists (list, create, edit, delete, the bookmarks of a list with paging, add/remove a bookmark). They follow the existing tRPC procedures' rules on the token's subtree. Other Karakeep REST paths stay unimplemented.
- **Landing page**: mention Floccus on the connected-apps feature card (EN/DE).
- **Docs and settings copy**: README sections for Floccus setup (Linkwarden or Karakeep mode); the Connected apps intro mentions Floccus.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `linkwarden-api`: new requirements for getting and editing a single collection; "Delete collections" deletes the contents too.
- `karakeep-api`: new requirements for the Karakeep REST endpoints that sync clients such as Floccus use.

## Impact

- Backend: `backend/internal/linkwarden` (collection get/edit, delete mode), `backend/internal/karakeep` (cursor encoding for REST paging, reuse of list/bookmark operations), `backend/internal/httpapi/linkwarden.go` and a new Karakeep REST router next to `karakeep.go`; Go tests with `httptest`.
- Frontend: `features.apps.description` and `settings.apps.intro` in `en.json` / `de.json`; landing test if it asserts the apps text.
- README: Floccus setup section.
- No database migration. Existing Linkwarden clients see the new delete behaviour.
- The landing copy change assumes `landing-everyday-helper` is applied (the five-card list with the `apps` card); it doesn't change landing requirements, only the card's wording.
