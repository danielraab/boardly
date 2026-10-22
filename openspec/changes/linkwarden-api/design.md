# Design

## Context

- `app-tokens` and `karakeep-api` built the parts this change reuses: `httpapi.appAPI(app, mux)` (Bearer token into the context, CORS, cookie ignored), `favourites.Scope` with `CheckCreate`/`CheckChange`/`CheckMove`, `Store.SubtreeNodes`, `Store.Create` with `First`, `Move`, `Delete(DeleteMoveChildren)`, `DetachTag`, and `images.Favicons.Lookup`. `internal/karakeep` shows how a client protocol is mapped onto the subtree.
- Node and tag IDs are UUID strings (`favourite_nodes.id`, `tags.id`, `TEXT PRIMARY KEY`). Favicons are stored as images with `content_type` (ICO, PNG, JPEG, GIF, WebP).
- The Linkwarden clients were read at linkwarden/linkwarden commit `952ac45` (web app `v2.16.3`, monorepo with `apps/mobile`, `apps/extension`, `packages/router` hooks, `packages/types`, `packages/lib/schemaValidation.ts`). Findings that shape this design:
  - The mobile app parses IDs with `Number(id)` (`app/links/[id].tsx`, `tags/[id].tsx`, `collections/[id].tsx`, `lib/offlineSync.ts`), so IDs have to be integers.
  - Token login: the app sends `GET /api/v1/users/me` with `Bearer <token>`, then `/api/v1/config` and `/users/me` to decide on the subscribe screen (`hasInactiveSubscription` needs `STRIPE_ENABLED`). The extension has an "API Key" method and reads `INSTANCE_VERSION` to turn on paginated tags (≥ 2.14.0) and tag search (≥ 2.14.1).
  - The link list polls every few seconds while any link has no `lastPreserved` (`anyPreservationPending`). Previews are only requested when `link.preview` is set and not `"unavailable"` (`formatAvailable`).
  - Envelopes differ per endpoint: search, tags and dashboard read `data.data`, everything else reads `data.response`, and errors show `response`.
  - The extension's duplicate badge searches `url:<page URL>`. Saving with "upload screenshot" posts the screenshot to `/api/v1/archives/{id}?format=0` after creating the link.
  - The mobile app's tag screen creates a tag without a link (`POST /api/v1/tags` with `[{label}]`).

## Goals / Non-Goals

**Goals:**
- The official Linkwarden mobile app and browser extension log in with a Famodash app token and can browse, search, save, edit, move, tag and delete links and manage collections in the token's subtree.
- Same rules as the Karakeep API: every read and write goes through `favourites.Scope` and `favourites.Store`, so validation, tag cleanup, favicon fetching and token revocation behave as in the web app.
- Exact wire shapes, pinned by tests modelled on the clients' requests.

**Non-Goals:**
- Preserved formats, uploads, highlights, pins, sharing, RSS, SSO, billing, bulk actions, and anything else listed as unsupported in the spec.
- The Linkwarden web app, or third-party tools that use other endpoints.
- Performance for very large trees (same trade-off as the Karakeep API).

## Decisions

### Integer IDs: a counter column filled by triggers

Migration `0012_int_ids.sql` adds `num INTEGER` to `favourite_nodes` and `tags`, fills existing rows in creation order (`created_at`, then `id`), and adds a unique index on each. A table `id_counters (name TEXT PRIMARY KEY, last INTEGER NOT NULL)` holds the last number given out for `nodes` and `tags`. An `AFTER INSERT` trigger on each table increments the counter and sets `num` on the new row. Because the counter only goes up, numbers are never reused, also after the highest row is deleted.

- Triggers cover every insert path (`Store.Create`, `EnsureRoot`, tag creation during `Update`) without touching them, and keep `num` out of the web API.
- `ScopedNode` and `favourites.Tag` get a `Num int64` field (`json:"-"` on `Tag`, so the web API is unchanged), and `SubtreeNodes` and the tag queries select it, together with the tag's `created_at`.
- The Linkwarden service resolves a number by indexing the loaded subtree, so no lookup query is needed and out-of-scope numbers are naturally "not found".

Alternatives: `MAX(num)+1` on insert (reuses the number of a deleted highest row, which the mobile app's offline cache could confuse with the old link); SQLite's `rowid` (not stable across `VACUUM` without an `INTEGER PRIMARY KEY`); a separate mapping table only for Linkwarden (a second source of truth to keep in sync on create and delete); switching primary keys to integers (a rewrite of every table and API).

### Package layout

```
httpapi/linkwarden.go   routes under /api/linkwarden, token check, JSON decoding,
        │               envelopes, error mapping, catch-all 404
        │ scope from the token
        ▼
linkwarden (new)        Service: Me, Dashboard, Search, Link, CreateLink, UpdateLink,
        │               DeleteLink, Collections, CreateCollection, DeleteCollection,
        │               Tags, Tag, DeleteTag, Preview
        ▼
favourites.Store        SubtreeNodes (+ Num, tag created_at), Create(First), Update,
                        Move, Delete(DeleteMoveChildren), DetachTag
```

`internal/linkwarden` holds the shapes (Go structs with Linkwarden's JSON names) and the rules from the spec, and is testable without HTTP. `httpapi/linkwarden.go` stays thin, like `httpapi/karakeep.go`. The URL comparison of Karakeep's `checkUrl` (`normalizeURL`) moves to `favourites` (`SameURL`) so both APIs compare URLs the same way.

### HTTP layer

- `router.go` mounts `/api/linkwarden/` with `a.appAPI(apptokens.AppLinkwarden, a.linkwardenAPI())`. `apptokens` gets `AppLinkwarden = "linkwarden"` and accepts it on create.
- `appAPI`'s preflight allows `GET, POST, PUT, DELETE`. This applies to Karakeep too, which is harmless.
- Each route is wrapped by a helper that, unless the route is public, answers 401 `{"response": "You must be logged in."}` when there is no token in the context, then decodes the body and calls the service with `favourites.Scope{UserID, RootID}` from the token.
- A catch-all handler for `/api/linkwarden/` answers 404 `{"response": "Not found."}`. Go's `ServeMux` prefers it over a 405 for a known path with another method, so the clients always get Linkwarden's error shape.
- Error mapping: `favourites.ErrNotFound` → 404 with "Link not found." / "Collection not found." / "Tag not found."; validation errors and `linkwarden.ErrUnsupported` → 400 with the message; root collection delete and `POST /tags` → 403. Server errors go through the usual `HandlerFunc` logging and answer 500 `{"response": "Internal server error."}`.
- Query parameters (`sort`, `cursor`, `collectionId`, `tagId`) are parsed leniently like Linkwarden's `Number(...)`: invalid values fall back to the default.

### Mapping

- The token's root folder is a collection, unlike Karakeep. Every Linkwarden link needs a `collectionId`, and links directly in the root folder must be reachable. The user's top-level folder has no title and is called "Unorganized", so the extension's default collection `{"name": "Unorganized"}` lands there.
- The user is always `1`. Each token sees exactly one user, and the clients only compare `ownerId` with `user.id` (e.g. "can edit"). `uuid` is the real Famodash user ID.
- `lastPreserved` = `createdAt` and all formats `"unavailable"`, so nothing polls and no archive tab is offered. `linksRouteTo` `ORIGINAL` makes a tap open the page, like `open_original_link` in Karakeep.
- Search and tags load the subtree once and filter, sort and page in Go (50 per page, Linkwarden's default `PAGINATION_TAKE_COUNT`). The cursor is the last item's integer ID, and the next page starts after it in the chosen order, as with Prisma's `cursor` + `skip: 1`. `nextCursor` is only set when more items follow, so the clients don't fetch an empty last page.
- `PUT /links/{id}` replaces tags with the given names (`Store.Update(Tags)`) and moves with `CheckMove` + `Move(position 0)` when `collection.id` differs. The clients send the whole link back, so fields Famodash returns as `""`, `null` or `[]` pass, and only values that would set something unsupported fail.
- `DELETE /collections/{id}` uses `DeleteMoveChildren`, as Karakeep's `lists.delete` does, so no link is lost.

### Previews from favicons

`linkwarden.Favicons` needs the content type, so it needs a lookup that returns `{imageID, contentType}` per origin. `images.Favicons` gets `LookupImages(ctx, origins)`, which joins `favicons` with `images`. `preview` is set for `image/png`, `image/jpeg`, `image/gif` and `image/webp`. ICO is left out because React Native's image decoder on Android doesn't reliably show it. `GET /archives/{id}?preview=true` (format 0 or 1) opens the image with `images.Store.Open` and streams it with its content type and `Cache-Control: private, max-age=86400`. The client caches by `updatedAt`, which doesn't change when a favicon changes, so a day keeps it reasonably fresh.

### `POST /api/v1/tags`

Refused with 403, the same decision and reason as Karakeep's `tags.create`: a tag without links contradicts the rule that unused tags are removed.

### Frontend

- `AppTokenApp` becomes `"karakeep" | "linkwarden"`, `appNames` gets "Linkwarden", and the create dialog gets the option. The server address in the secret view is built from the chosen app (`/api/karakeep` or `/api/linkwarden`).
- Texts that only mention Karakeep (settings intro, empty state, landing feature) mention both apps, in en and de.

## Risks / Trade-offs

- [Linkwarden changes shapes or endpoints in a client update] → Shapes are pinned by tests, and the README names the Linkwarden version (`v2.16.3`, commit `952ac45`). `INSTANCE_VERSION` stays at that version until Famodash is checked against a newer one.
- [Clients may need fields that the source read missed] → Every field of Linkwarden's `Link`, `Collection`, `Tag` and user response is present with a neutral value, as for Karakeep. A manual check with both clients is part of the tasks.
- [Triggers are less visible than Go code] → Documented in the migration and covered by favourites tests (create, delete highest, existing data).
- [Search grammar is a small subset of Linkwarden's] → Unknown qualifiers are searched as plain text, so a query never fails, it may just find less.
- [Screenshot upload from the extension fails after the link is saved] → The link is kept, and the 400 message tells the user Famodash doesn't store screenshots. The README advises turning the option off.
- [Favicons as previews are small and blurry] → Accepted for now (user decision). Thumbnails can replace them later without API changes.

## Migration Plan

Migration `0012` adds the columns, counters, indexes and triggers and fills existing rows in one transaction; `0013` rebuilds `app_tokens` with the new app allowed, keeping all rows. Deploy the new binary. Rollback is the previous binary: the old code ignores the new columns, the triggers keep working on inserts, and the old code never creates Linkwarden tokens (existing ones are rejected by its app check when authenticating).

## Open Questions

- None blocking. A device check (Android and iOS) of PNG, GIF and WebP favicons as previews happens during apply.
