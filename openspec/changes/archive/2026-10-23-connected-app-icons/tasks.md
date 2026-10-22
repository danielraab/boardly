# Tasks

## 1. Logo assets

- [x] 1.1 Copy Karakeep's `apps/web/public/icons/logo-icon.svg` and `logo-icon-dark.svg` unchanged to `frontend/src/features/settings/appTokens/logos/karakeep.svg` and `karakeep-dark.svg`. Verify both files open as the black and the white mark
- [x] 1.2 Download Linkwarden's `apps/web/public/icon.png`, downscale it to 64×64 and optimize it as `logos/linkwarden.png`. Verify that the file is 64×64 and a few KB at most
- [x] 1.3 Add `logos/README.md` with each file's upstream repository, path, license (AGPL-3.0) and any processing. Verify that every file in `logos/` is listed

## 2. Component and list

- [x] 2.1 Add `appTokens/AppLogo.tsx` with an `appLogos: Record<AppTokenApp, { light; dark? }>` map and an `AppLogo` component that picks the variant with `useThemeMode()` (falling back like `ProviderIcon`) and renders `<img alt="" className="size-8 shrink-0" />`. Verify that `pnpm tsc` passes
- [x] 2.2 Render `<AppLogo app={token.app} />` as the first child of the `<li>` in `AppTokenItem` (`ConnectedAppsTab.tsx`), keeping the app name in the text line. Verify in the running app in a light and a dark theme

## 3. Tests and checks

- [x] 3.1 Extend `ConnectedApps.test.tsx`: a Karakeep and a Linkwarden token show different logo `src`s with empty `alt`, the app names are still shown as text, and with theme `black` the Karakeep token uses the dark variant and the Linkwarden token uses its only logo. Verify that `pnpm test` passes
- [x] 3.2 Run `make check` and `make test` and verify that both pass
