# Tasks

## 1. Store query

- [x] 1.1 Add `RecentLinks` to the favourites store (subtree CTE, links only, optional `since`, `ORDER BY created_at DESC, id DESC LIMIT`, reports whether the folder exists) and verify with Go tests: newest first across subfolders, limit, `since` filter, ID tie-break, another user's folder and a missing folder
- [x] 1.2 Verify an imported link keeps its source date and is ordered by it, with a Go test using `ImportLink.CreatedAt`
- [x] 1.3 Set `createdAt` on the nodes of the owner's tree (`Store.Tree`, omitted elsewhere) for the edit-mode preview and verify with a Go test that the tree has it and `Subtree` doesn't

## 2. Widget kind (backend)

- [x] 2.1 Add the `recent_favourites` kind in `backend/internal/widgets/recent.go` with config validation (folder, limit 1–100, days nil or 1–365, title, size) and verify Normalize tests reject each invalid value with `invalid_layout`
- [x] 2.2 Implement Resolve (owner/visitor visibility, `since = now − days×24h`, favicons, missing folder for the owner, left out for visitors) and register it in `widgets.Register`; verify with tests for flag off/on, missing folder and that the visitor response has only id/title/url/icon/createdAt
- [x] 2.3 Add an httptest-level check that a dashboard lookup returns the widget's links, and verify `make test` passes for the backend

## 3. Frontend widget

- [x] 3.1 Add API types for the widget's config and data and a relative-time helper (`Intl.RelativeTimeFormat`, `numeric: "auto"`, largest fitting unit), and verify unit tests for seconds/hours/yesterday/days/months/years in en and de
- [x] 3.2 Add `widgets/recentFavourites` with the settings form (folder picker, number of links, optional days, title, size, public flag with hint; can't confirm without folder or with out-of-range numbers) and add it to `registry.ts`; verify with a Testing Library test of the defaults and validation
- [x] 3.3 Add the view: rows with icon/globe, title or URL, full URL cut off with "…", relative date with full-date tooltip, sizes, title heading, info button for the owner outside edit mode, no navigation in edit mode, empty states and missing-folder message (sharing the link list's row markup where practical); verify with Testing Library tests for each
- [x] 3.4 Add all en + de strings (kind name and description, setting labels, public hint, empty states) and verify the i18n test passes

## 4. Check

- [x] 4.1 Run `make check` and `make test` and verify both pass
- [x] 4.2 Run the app, add a recent favourites widget with a day limit, and verify the list, the empty state and the public dashboard view by hand
