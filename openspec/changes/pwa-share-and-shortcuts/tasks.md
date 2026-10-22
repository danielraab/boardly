# Tasks

## 1. Prerequisites

- [ ] 1.1 Archive `pwa-install` and verify `openspec/specs/pwa/spec.md` exists

## 2. Login redirect

- [ ] 2.1 Add a `safeRedirect` helper with unit tests for local paths, `//host`, `/\host`, absolute URLs and `/login` paths
- [ ] 2.2 Add the `redirect` search parameter to `/login` and use it after passkey sign-in and in the signed-in `beforeLoad` redirect; verify with an auth test that a passkey sign-in with `redirect=/share?url=…` ends on that page

## 3. Save link page

- [ ] 3.1 Add a pure `sharedLink(params)` function (URL from `url` or from `text`, title from `title` or the rest of `text`, length limit) with unit tests for the spec scenarios
- [ ] 3.2 Extract the link form from `NodeDialog` into a component used by both the dialog and the new page; verify the existing favourites tests still pass
- [ ] 3.3 Add the `/share` route (signed-out → `/login?redirect=…`), prefilled form, folder picker with the last used folder from localStorage (falling back to the root), Save/Cancel, and the confirmation with "Open favourites" and "Save another link"; verify with Testing Library tests for the spec scenarios
- [ ] 3.4 Add en and de strings under `share.*` and verify the i18n key check passes

## 4. Manifest

- [ ] 4.1 Add `share_target` (GET `/share`, params `title`, `text`, `url`) and the translated `shortcuts` to both manifests; extend the manifest test to check that both declare the same share target and shortcut URLs

## 5. Verification

- [ ] 5.1 Install the app on an Android device, share a page from Chrome or another app to Famodash and verify the "Save link" page is prefilled and saves into the chosen folder; long-press the icon and verify the three shortcuts; run `make check` and `make test`
