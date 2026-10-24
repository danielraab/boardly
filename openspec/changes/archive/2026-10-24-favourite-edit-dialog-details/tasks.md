# Tasks

## 1. Backend

- [x] 1.1 Add `Description` and `PreviewImageID` (omitempty) to `favourites.Node`; select and fill them in `Store.Tree`, `Create` and `Update`; verify with store tests that a link with description and preview returns both and a folder returns neither
- [x] 1.2 Add `RemovePreview` to `favourites.Patch`: clears `preview_image_id` without enqueuing a fetch, rejected with `ErrInvalidField` on folders; verify with store tests (removed, folder rejected, no job enqueued)
- [x] 1.3 Accept `description` and `removePreview` in `PATCH /api/favourites/{id}`; verify with httpapi tests for set, clear, too long (`invalid_description`), remove preview and folder (`invalid_field`)
- [x] 1.4 Verify the favourite explorer and link widgets on a public dashboard don't return `description` or `previewImageId` (widget/httpapi test)
- [x] 1.5 Run `go test ./...` in `backend/` and verify it passes

## 2. Frontend: link form and edit dialog

- [x] 2.1 Add `description` and `previewImageId` to `FavouriteNode`, and `description` / `removePreview` to `updateFavourite`'s patch in `api.ts`; add `checkDescription` (max 2048) to `validate.ts` with a unit test
- [x] 2.2 Reorder `LinkFields` to title, URL, tags, folder and add the background-fetch hint for new links; verify the dialog and share page tests find the title field before the URL field and the hint text
- [x] 2.3 In `NodeDialog`, for editing a link: add the description textarea, the preview with staged "Remove preview" / "No preview image", and send them on save; verify with tests (edit description, too long, remove preview and save, cancel keeps preview, folder dialog unchanged)
- [x] 2.4 Move the "Refresh icon" action and `refreshErrorKey` from `TreeRow` into `NodeDialog`, showing the current icon (cache-busted after refresh) and errors inside the dialog; verify tests: refresh success updates icon and keeps dialog open, not-found message, no refresh button in tree rows
- [x] 2.5 Add en/de translations for description, preview, remove preview, no preview, hint and header "Add link"; remove unused keys; verify the i18n key parity test passes

## 3. Frontend: header

- [x] 3.1 Add the "Add link" header button to `AppShell` for signed-in users, before the theme switcher, linking to `/share`; verify with `AppShell` tests (shown and navigates for signed-in users, hidden when signed out)

## 4. Verification

- [x] 4.1 Run frontend lint, typecheck and `vitest` and verify they pass
- [x] 4.2 Run the app and check the edit dialog (description, preview remove, icon refresh), the reordered share page with hint, and the header button in light and dark themes
