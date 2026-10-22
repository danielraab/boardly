# Tasks

## 1. Integer IDs

- [x] 1.1 Add migration `0012_int_ids.sql`: `num` columns on `favourite_nodes` and `tags` filled in creation order, unique indexes, the `id_counters` table set to the highest number, and `AFTER INSERT` triggers that assign the next number; verify with a db test that migrates a database with existing nodes and tags and checks distinct numbers and the counters
- [x] 1.2 Add `Num` to `favourites.ScopedNode` and `favourites.Tag` (`json:"-"`), and the tag's creation time to the subtree tags, selected by `SubtreeNodes`; verify with favourites tests that a node keeps its number after a move and rename, a new node after deleting the highest gets a higher number, and the web API's JSON has no `num`

## 2. Shared pieces

- [x] 2.1 Move Karakeep's URL comparison to `favourites.SameURL` and use it from `internal/karakeep`; verify that the existing Karakeep tests still pass and add a unit test for fragments and trailing slashes
- [x] 2.2 Add `images.Favicons.LookupImages(ctx, origins)` returning image ID and content type per origin; verify with an images test for a known PNG, a known ICO and an unknown origin
- [x] 2.3 Add the app `linkwarden` to `apptokens` (create accepts it, `Authenticate` binds it) and allow `GET, POST, PUT, DELETE` in `appAPI`'s preflight; verify with apptokens and httpapi tests for creating a Linkwarden token, a Karakeep token rejected under the Linkwarden app, and a `PUT` preflight

## 3. Linkwarden service (`internal/linkwarden`)

- [x] 3.1 Add the shapes (link, short collection, collection, tag, user, config, dashboard section) and the mapping from the subtree: integer IDs, user `1`, root collection named after the folder or "Unorganized", `lastPreserved`, `"unavailable"` formats, `preview` from the favicon content type; verify with unit tests of the full JSON of one link, one collection and one tag
- [x] 3.2 Implement `Search`: sort orders, `collectionId`, `tagId`, `pinnedOnly`, the query grammar (`url:`, `name:`, `tag:`, `collection:`, quotes, plain terms) and the ID cursor with 50 per page; verify with service tests for paging over 60 links, collection and tag filters, `url:` with a fragment and a trailing slash, and unknown qualifiers
- [x] 3.3 Implement `Link`, `CreateLink` (collection by ID, by name or root; first position; tags by name; unsupported fields), `UpdateLink` (title, URL, tag replacement, move, unsupported fields, nothing changed on rejection) and `DeleteLink`; verify with service tests for each scenario in the spec, including a new collection by name, a PDF type, and a pin request
- [x] 3.4 Implement `Collections` (tree order, parents, link counts), `CreateCollection` and `DeleteCollection` (contents move up, root refused); verify with service tests for nested folders, link counts, and deleting a collection with links and a subfolder
- [x] 3.5 Implement `Tags` (visibility, counts, search, six sort orders, cursor), `Tag` and `DeleteTag` (subtree only); verify with service tests for a tag used inside and outside the subtree and for the name sort with search
- [x] 3.6 Implement `Me`, `Config`, `Dashboard` (16 newest links, tag count, fixed sections) and `Preview` (favicon image for format 0/1 with `preview=true`, else not found); verify with service tests using a fixed clock

## 4. HTTP endpoints

- [x] 4.1 Add `httpapi/linkwarden.go` with every route from the spec, the token check with Linkwarden's 401, lenient query parsing, envelopes, error mapping, the `POST /api/v1/session`, `POST /api/v1/tags` and archive upload refusals, and the catch-all 404; mount it in `router.go`; verify with httptest requests shaped like the clients': extension login (`config`, collections, tags with `sort=2&cursor=0&search=`), duplicate check and save with `{"name": "Unorganized"}`, mobile login (`users/me`, `config`, `logins`), dashboard, link list with cursor, edit and move, delete collection, preview image, a request outside the subtree, a Karakeep token, and an unknown path

## 5. Frontend

- [x] 5.1 Add "Linkwarden" to `AppTokenApp`, `appNames` and the create dialog, and build the server address in the secret view from the chosen app; verify with Vitest tests that the dialog offers both apps with Karakeep preselected and shows `/api/linkwarden` after creating a Linkwarden token
- [x] 5.2 Update the texts that only mention Karakeep (connected apps intro and empty state, landing feature) in en and de; verify that the i18n key check passes and both languages mention Linkwarden

## 6. Docs and checks

- [x] 6.1 Add a "Linkwarden-compatible server" section to the README: server address, logging in with an access token, what works in the mobile app and extension, how collections and tags map to folders and tags (root collection, new links first, delete keeps links), favicon previews, what isn't supported (including turning off screenshot upload in the extension), and the Linkwarden version the shapes come from; verify by reading it against the spec
- [ ] 6.2 Run `make check` and `make test`, then log in with a Linkwarden token in the mobile app and the browser extension against a local server: save a page, see the "already saved" badge, browse the dashboard, collections and tags, edit, move and delete a link, and check that favicon previews show on Android and iOS; verify that both commands pass and the clients work without errors on the supported screens
