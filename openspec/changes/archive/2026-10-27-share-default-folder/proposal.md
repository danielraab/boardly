# Proposal

## Why

The "Save link" page (`/share`, used by the share target and the "Add link" shortcut) starts in the folder last saved into in the current browser. That is per browser, so every new device or installed app starts at the root, and one save into an unusual folder changes where the next shared link goes. Users want to pick one folder, for example "Inbox", that new links land in by default.

## What Changes

- Users can choose a **default folder for new links** in Settings → Profile, using the existing folder picker. It can be reset to the root.
- The choice is stored on the user account on the server, so it applies on every device and in the installed app.
- `GET /api/me` returns it as `shareFolderId` (`null` = root), and `PATCH /api/me` accepts `{"shareFolderId": ...}`. The server checks that the ID is a folder of the user (400 `invalid_folder`).
- `/share` preselects the default folder; if none is set or it was deleted, the root. Picking another folder on the page only affects that one save.
- **BREAKING** (behaviour): `/share` no longer remembers the last used folder in the browser. The `boardly.share.folder` localStorage entry is no longer read or written.
- Deleting the default folder resets the setting to the root.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `pwa`: "Save link page" preselects the user's default folder instead of the folder last used in this browser.
- `user-profile`: new requirement for choosing the default folder in the Profile tab, and `PATCH /api/me` accepts `shareFolderId` alongside `name` and `slug`.

## Impact

- Backend: migration adding `users.share_folder_id` (FK to `favourite_nodes`, `ON DELETE SET NULL`), `internal/users` (field, patch, validation), `PATCH /api/me` handler, tests.
- Frontend: `User` type, Profile tab (new folder picker form), `SharePage` (drop localStorage, use `me.shareFolderId`), translations (en + de), tests.
- README: Save link / settings docs.
