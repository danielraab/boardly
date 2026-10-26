# Tasks

## 1. Backend

- [x] 1.1 Add a `skipDuplicates` argument to `favourites.Store.Import`; when false, skip loading and checking URLs. Verify with store tests for "duplicates allowed" (tree and within-file) next to the existing `TestImportSkipsDuplicates`.
- [x] 1.2 Parse `skipDuplicates` in `importFavourites` (missing → true, `true`/`false`, else 400 `invalid_request`) and pass it on. Verify with handler tests for the invalid value and for `false` importing a duplicate.

## 2. Frontend

- [x] 2.1 Add `skipDuplicates` to `importBookmarks` in `import/api.ts` and send it in the query. Verify with the form test that checks the request URL.
- [x] 2.2 Add the checkbox "Skip links that are already saved" / "Bereits gespeicherte Links überspringen" (checked by default) to `ImportExportTab.tsx` with en/de translations. Verify with form tests for the default and unchecked cases.

## 3. Check

- [x] 3.1 Run `make check` and `make test` and verify both pass.
