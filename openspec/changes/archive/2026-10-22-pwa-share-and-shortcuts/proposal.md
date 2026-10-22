# Proposal

## Why

Saving a link is the most frequent thing people do with a bookmark manager, and on a phone it starts in another app's share sheet. Once Famodash is installed (change `pwa-install`), it can appear in that share sheet and offer shortcuts on its launcher icon. This makes the installed app clearly more useful than a browser tab.

## What Changes

- **Share target**: the installed app registers as a share target for links and text. Sharing a page to Famodash opens a new "Save link" page (`/share`) that is prefilled with the shared URL and title. There the user picks a folder and adds tags, then saves the link into their favourites tree.
- The "Save link" page also works without a share, as an empty form. It is the destination of the "Add link" shortcut.
- **App shortcuts** on the launcher icon (long-press on Android, right-click on desktop): "Add link" (`/share`), "Favourites" (`/favourites`) and "Dashboards" (`/dashboards`), translated like the manifest.
- Signed-out visitors who open `/share` are sent to `/login` and come back to the same share after signing in on the login page. To support this, the login page gains a `redirect` parameter that it honours for local paths.
- Still no offline support: sharing while offline shows the browser's usual offline error.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `pwa`: adds the share target, the "Save link" page and the app shortcuts. Added by `pwa-install`, which must be archived first.
- `auth-session`: the login page returns to a requested local page after sign-in.

## Impact

- Frontend: both manifests (`share_target`, `shortcuts`), new route `/share` with a "Save link" form that reuses the favourites form parts (URL and title fields, folder picker, tag input), a `redirect` search parameter on `/login`, and new en/de strings.
- Backend: none. `share` is already a reserved user slug, and saving uses the existing favourites API.
- Depends on `pwa-install`.
