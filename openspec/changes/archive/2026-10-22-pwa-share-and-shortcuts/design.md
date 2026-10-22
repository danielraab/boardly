# Design

## Context

`pwa-install` adds two static manifests (`manifest.en.webmanifest`, `manifest.de.webmanifest`) and no service worker. Favourites are created with `createFavourite` (`POST /api/favourites`). The favourites dialog (`NodeDialog`) already combines URL and title fields, `FavouritePicker` (folder mode) and `TagInput`, with validation in `validate.ts`. `share` is already in the reserved slugs list, so `/share` can't clash with a user's dashboard address. Protected routes redirect signed-out visitors to `/login`, and after a successful sign-in the login page always navigates to `/`.

## Goals / Non-Goals

**Goals:**
- Save a shared link in two taps: share → (optionally choose a folder) → Save.
- Shortcuts for the three most frequent destinations.

**Non-Goals:**
- Sharing files or images (`share_target` with POST and `enctype="multipart/form-data"`, which would need a service worker).
- Detecting duplicates of an already saved URL.
- Shortcut icons other than the app icon.
- Remembering the share across a magic-link sign-in that completes in another browser tab or app.

## Decisions

### `share_target` with GET
```json
"share_target": { "action": "/share", "method": "GET",
  "params": { "title": "title", "text": "text", "url": "url" } }
```
GET needs no service worker and lands directly on a client route. Server-side POST handling would need a backend endpoint and CSRF considerations.

### Finding the URL
Android apps often put the link into `text` instead of `url` (for example "Look at this https://…"). The page uses `url` if it is a valid http(s) URL; otherwise it uses the first http(s) URL found in `text`. The title is `title`, or else the remaining text of `text` without the URL, trimmed and cut to the 1000-character limit. Without a URL, the URL field stays empty for the user to fill in. This logic is a pure function with unit tests.

### One "Save link" page for share and shortcut
`/share` renders the link form as a page, not as a dialog. The form part of `NodeDialog` (`NodeForm`) is extracted so the dialog and the page share it. Folder: the folder used last on this page, stored in localStorage (`famodash.share.folder`), falling back to the root when that folder no longer exists. After saving, the page shows a confirmation with "Open favourites" and "Save another link" (on the "Add link" shortcut); "Cancel" goes to `/`.

### Login `redirect` parameter
`/share` redirects signed-out visitors to `/login?redirect=<path+search>`. After a sign-in completed on the login page (passkey, or the one-time code from `magic-link-code`), the page navigates to `redirect` when it is a local path (starts with a single `/`, not `//`, and is not a `/login` path), and to `/` otherwise. The magic-link verify page keeps going to `/`, as it usually opens in another tab.
*Alternative*: keep the share in sessionStorage: rejected, because a URL parameter is visible, testable and survives a reload just as well.

### Shortcuts
`shortcuts` with `name`/`short_name` translated per manifest, `url`s `/share`, `/favourites`, `/dashboards`, and no icons (the browser uses the app icon).

## Risks / Trade-offs

- [The share target only exists on Android/ChromeOS installs; iOS and desktop have no share target for web apps] → Accepted. The "Add link" shortcut and page still help there.
- [Chrome caches the manifest, so the share target and shortcuts appear only after it refreshes it or the app is reinstalled] → Mention in the release notes.
- [Open redirect via `redirect`] → Only local paths are accepted (see above), and this is covered by tests.
