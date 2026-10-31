# Tasks

## 1. Linkwarden collections

- [ ] 1.1 Add `GET /api/v1/collections/{id}` (service `Collection` + route); verify with httptest: 200 shape for a folder in scope, 404 `{"response": "Collection not found."}` out of scope
- [ ] 1.2 Add `EditCollection` + `PUT /api/v1/collections/{id}` (rename, move to end of new parent, `null` = token root, cycle → 400, out-of-scope parent → 400, unsupported echoed fields → 400, root rename/move → 403, root no-op → 200); verify each case with service and httptest tests, including a PUT body that is exactly the GET response with a new name
- [ ] 1.3 Switch `DELETE /api/v1/collections/{id}` to delete contents (`DeleteWithChildren`); update the existing delete tests so links and subfolders are gone, unused tags removed and tokens rooted in a deleted subfolder revoked; root still 403

## 2. Karakeep REST layer

- [ ] 2.1 Add the REST router (`httpapi/karakeep_rest.go`) with token check, `{"error"}` bodies and the error→status mapping (400/401/404/500); verify 401 without a token and 404 for an unnamed `/api/v1/…` path
- [ ] 2.2 Bookmark endpoints: `POST` (201 / 200 `alreadyExists`), `GET` (ignores `includeContent`), `PATCH`, `DELETE` (204), `GET …/lists`; verify each with httptest against the matching procedure's behaviour
- [ ] 2.3 Tag endpoints `POST`/`DELETE /bookmarks/{id}/tags` with `{"tags": [{tagName}|{tagId}]}`; verify `attached`/`detached` IDs and the resulting link tags
- [ ] 2.4 List endpoints: `GET`, `POST` (201), `PATCH`, `DELETE` (204), `PUT`/`DELETE …/bookmarks/{bookmarkId}` (204); verify with httptest, including create with `icon` + `type: "manual"` and a nested `parentId`
- [ ] 2.5 `GET /lists/{id}/bookmarks` with base64url cursor, `limit`, `sortOrder`; verify 25 links page as 20 + 5 with a URL-safe string cursor, a trailing `&` in the query works, and an invalid cursor → 400

## 3. Floccus call sequences

- [ ] 3.1 Add a Linkwarden integration test that replays the Floccus adapter: initial tree load (search pages + collections), create folder, create bookmark, update bookmark (GET parent collection + PUT link), rename and move folder (GET + PUT collection), remove bookmark, remove folder; verify the resulting favourites tree after each step
- [ ] 3.2 Add the same for the Karakeep REST adapter: create root list, create folder, create bookmark + add to list + attach tags, update (PATCH + tag diff + list move), remove bookmark (lists → remove from list → delete), remove folder (children first, then list); verify the tree after each step
- [ ] 3.3 Manually sync a real Floccus extension (Linkwarden mode and Karakeep mode) against a local build; verify a folder rename, a bookmark move and a folder delete round-trip to the browser without duplicates

## 4. Copy and docs

- [ ] 4.1 Add Floccus to `features.apps.description` in `en.json` and `de.json` (design §6 copy rules); verify matching keys and that the landing test still passes
- [ ] 4.2 Mention Floccus in `settings.apps.intro` (EN/DE), recommending a Linkwarden token; verify i18n tests pass
- [ ] 4.3 Add a README "Floccus" section (server URLs for both modes, token as password, Linkwarden mode recommended, duplicate-URL limitation in Karakeep mode) and note the Linkwarden delete change; verify links/anchors render

## 5. Wrap-up

- [ ] 5.1 Run `make check` and `make test` and verify both pass
