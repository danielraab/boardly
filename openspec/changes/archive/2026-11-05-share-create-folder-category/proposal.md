# Proposal

## Why

On the share page (`/share`) a link can only be saved into an existing folder and a note only into an existing category. When the right folder or category is missing, the user has to leave the page, create it on `/favourites` or `/notes` and share again, losing what was shared. Creating it right on the share page removes that detour, which matters most on a phone, where sharing happens.

## What Changes

- With "Link", the share page offers "New folder" / "Neuer Ordner" below the folder choice. It opens the existing folder dialog of the favourites page, creating the folder inside the folder currently chosen (the root when the root is chosen), so subfolders are created by choosing their parent first.
- With "Note", the share page offers "New category" / "Neue Kategorie" below the category choice. It opens the existing category dialog of the notes page, with the currently chosen category preselected as parent ("None" when "No category" is chosen); the parent can still be changed in the dialog.
- After the dialog saves, the new folder or category is chosen in the form, for this save only: like any other choice on the share page it doesn't change the default folder or default category.
- Cancelling the dialog, or a failed save in it, leaves the share form and its values as they were.
- The category dialog gains an optional callback that receives the created or edited category, so the share page can choose it. The notes page keeps working as before.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `pwa`: "Save link page" and "Save note from a share" gain creating a folder or category from the share page.

## Impact

- Frontend only: `frontend/src/features/share/SharePage.tsx`, `frontend/src/features/notes/categories/CategoryDialog.tsx` (optional `onSaved`), translations (en, de), and tests in `SharePage.test.tsx`.
- No backend changes: the existing `POST /api/favourites` (folder) and `POST /api/note-categories` routes are used. The OpenAPI documents are unaffected.
- Export and import are unaffected: no user data is added, removed or reshaped.
