# Tasks

## 1. Category dialog callback

- [x] 1.1 Add an optional `onSaved?: (category: NoteCategory) => void` to `CategoryDialog`, called with the saved category before it closes; verify the existing notes tests (`pnpm test` in `frontend`) still pass without changes

## 2. Share page

- [x] 2.1 Add a "New folder" button right after the folder choice (via a new optional `folder.action` of `LinkFields`) in "Link" mode that opens `NodeDialog` for a folder inside the chosen folder (root → top level) and sets `chosenFolder` from `onSaved`; verify by the tests in 3.1
- [x] 2.2 Add a "New category" button below `CategoryPicker` in "Note" mode that opens `CategoryDialog` with `parentId` set to the chosen category (or `null`) and sets `chosenCategory` from `onSaved`; verify by the tests in 3.2
- [x] 2.3 Add or reuse the translations for both buttons in `en.json` and `de.json` ("New folder" / "Neuer Ordner", "New category" / "Neue Kategorie"); verify `make check` passes, including the i18n key checks

## 3. Tests

- [x] 3.1 In `SharePage.test.tsx`, cover creating a subfolder and a top-level folder, saving a link into the new folder, cancelling the dialog keeping URL, title, tags and folder, and the default folder staying unchanged; verify the tests pass
- [x] 3.2 In `SharePage.test.tsx`, cover creating a subcategory with the preselected parent, a top-level category from "No category", saving a note into the new category, cancelling keeping title, text and category, and the default category staying unchanged; verify the tests pass

## 4. Wrap-up

- [x] 4.1 Confirm that export and import (`data-export`, `data-import`) and the OpenAPI documents need no change, since no data shape or route changes; verify `make check` and `make test` pass
