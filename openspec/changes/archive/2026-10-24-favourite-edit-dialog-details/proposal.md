# Proposal

## Why

Links now have a description and a preview image, but Boardly's own UI can neither see nor change them: only connected apps can. The link form also asks for the URL before the title, the "Refresh icon" button crowds every row of the favourites tree, and the "Save link" page is only reachable through the PWA share target or app shortcut.

## What Changes

- The link form (favourites dialog and "Save link" page) shows the title field before the URL field.
- The edit dialog of an existing link gains a description field and shows the link's preview image with a "Remove preview" action. There is no upload.
- The new-link dialog and the "Save link" page don't get these fields. Instead they show a hint that the description and preview image are fetched from the page in the background and can be edited afterwards.
- The "Refresh icon" action moves from each link row in the favourites tree into the edit dialog of the link, next to the link's current icon. Errors are shown inside the dialog.
- `/api/favourites` returns `description` and `previewImageId` on links, and `PATCH /api/favourites/{id}` accepts `description` and `removePreview`. **BREAKING** for the spec rule that Boardly's UI and `/api/favourites` never show or change these values; no client depends on their absence.
- The global header gets an "Add link" button for signed-in users that opens `/share`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `favourites`: the favourites page loses the per-row "Refresh icon" action; the link edit dialog gains description, preview (show and remove) and icon refresh; the new-link dialog shows the background-fetch hint; title before URL.
- `link-previews`: description and preview are exposed and editable through `/api/favourites` and Boardly's UI; a user can remove a preview.
- `pwa`: the "Save link" page shows title before URL and the background-fetch hint.
- `app-shell`: the global header gets an "Add link" button for signed-in users.

## Impact

- Backend: `backend/internal/favourites` (tree query returns description and preview, patch can remove the preview), `backend/internal/httpapi/favourites.go` (new request and response fields).
- Frontend: `features/favourites` (`LinkFields`, `NodeDialog`, `FavouritesPage`, `api.ts`, `validate.ts`), `features/share/SharePage.tsx`, `components/AppShell.tsx`, locales (en, de), and their tests.
- No migration: the columns already exist.
