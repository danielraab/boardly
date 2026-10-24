# Design

## Context

The `favourite_nodes` table already has `description` and `preview_image_id` (see the archived description and preview change). `favourites.Store.Tree` doesn't select them, `favourites.Node` has no JSON fields for them, and `Patch` already supports `Description` but the HTTP handler doesn't pass it. `favourites.Node` is also serialized by the favourite explorer widget, which public dashboards serve to anyone.

The "Refresh icon" action lives in `TreeRow` in `FavouritesPage.tsx`, calling `POST /api/favicons/refresh`. `LinkFields` is shared by `NodeDialog` and `SharePage`.

## Goals / Non-Goals

**Goals:**
- Show and edit the description, show and remove the preview, and refresh the icon from the link edit dialog.
- Keep description and preview private to the owner.

**Non-Goals:**
- Uploading a preview from Boardly's UI.
- Setting a description when creating a link through `/api/favourites`; the page fetch fills it.
- Showing description or preview in dashboard widgets.

## Decisions

### Owner-only fields on `Node`
Add `Description *string` (`json:"description,omitempty"`) and `PreviewImageID *string` (`json:"previewImageId,omitempty"`) to `favourites.Node`. Only `Tree`, `Create` and `Update` fill them; the subtree loaders used by widgets don't, so `omitempty` keeps them out of public dashboard responses. A test on the explorer widget's public response guards this.
*Alternative:* a separate response type for `/api/favourites`. More code for the same result; rejected.

### Removing the preview through PATCH
`Patch` gets `RemovePreview bool`, sent as `"removePreview": true`. The update sets `preview_image_id = NULL` when the URL changed or `RemovePreview` is set. No job is enqueued, so the removed preview doesn't come back. The image cleanup already deletes images no node references.
*Alternative:* `"previewImageId": null`. Distinguishing an explicit `null` from a missing field needs a custom JSON type in Go; a boolean flag is clearer.

### Removal is staged in the dialog
"Remove preview" only sets local state; the flag goes with the dialog's save. This matches how the other fields behave and makes "Cancel" keep the preview.

### Icon refresh in the dialog works on the saved URL
The refresh uses the link's stored URL, not the URL field, because the result is shown for the saved link and a changed URL triggers its own favicon fetch on save. On success the dialog invalidates the favourites query; the dialog reads the icon from the fresh tree data for its node ID so it shows the new icon. Since a refresh keeps the image ID, the dialog's `ImageIcon` gets a `version` (the refresh's submit time), appended as `?v=`, so the browser reloads it. The `refreshErrorKey` helper moves with the action into `NodeDialog`.

### Header button reuses the header button style
The "Add link" button is a TanStack `Link` to `/share` styled like `ThemeSwitcher`'s button, rendered only when `useMe()` returns a user.

### Background-fetch hint lives in `LinkFields`
`LinkFields` gets a `hint` flag (set when creating), so the dialog and the share page show the same text. `LinkFields` reorders to title, URL, tags, folder. Description and preview are rendered by `NodeDialog` only for editing, not by `LinkFields`.

## Risks / Trade-offs

- [Description or preview leaks into public widget responses if a loader starts filling them] → widget test asserting the fields are absent.
- [Browser shows the stale favicon after refresh because the URL is unchanged] → cache-busting query on the dialog's icon; the tree already re-renders after invalidation and serves favicons with a one-day cache, which is the existing behaviour.
- Tests that locate fields by order (`getAllByRole("textbox")[0]`) may break with the reorder → update them to use labels.
