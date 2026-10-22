# Design

## Context

- `AppTokenItem` in `ConnectedAppsTab.tsx` renders each token as `<li className="flex items-start gap-4 …">` with a details column and a revoke/delete button. The app appears only as `appNames[token.app]` in the second line.
- `AppTokenApp` (`appTokens/api.ts`) is `"karakeep" | "linkwarden"`, and `appNames` is a `Record<AppTokenApp, string>`.
- The official assets upstream (both AGPL-3.0):
  - Karakeep: `apps/web/public/icons/logo-icon.svg` (a black rounded-square mark, ~1.4 KB) and `logo-icon-dark.svg` (the same mark in white).
  - Linkwarden: no SVG. `apps/web/public/icon.png` and `icon-small-150x150.png` (a dark rounded tile with a blue lightning bolt).
- The theming spec requires images with light/dark variants to follow the theme mode. `ProviderIcon` in `PasskeysTab.tsx` already does this with `useThemeMode()` from `#/theme` and a fallback to the other variant.
- The name `AppIcon` is already used by `#/components/AppIcon` (Boardly's own icon).

## Goals / Non-Goals

**Goals:**
- A logo per app, in its official colors and following the theme mode, from a single place keyed by `AppTokenApp`, so adding a third app fails type-checking until it has a logo.

**Non-Goals:**
- Logos in the create dialog's app `<select>`.
- Redrawing or recoloring the logos.

## Decisions

### Bundle the files as Vite assets in the feature folder
Put the files in `frontend/src/features/settings/appTokens/logos/` (`karakeep.svg`, `karakeep-dark.svg`, `linkwarden.png`) and import them, so Vite fingerprints them (or inlines the small ones) and they end up in the embedded `dist`.

*Alternatives*: `public/` gives unhashed URLs and keeps the files away from the feature. Hotlinking the upstream URLs sends requests to third parties, breaks offline/PWA use, and breaks if upstream moves the files. Inline JSX SVG doesn't work for Linkwarden (PNG only) and would mean two rendering paths.

### `<img>` over inline SVG
Render both logos with `<img src alt="" />`, like `ProviderIcon`. That's one code path for SVG and PNG, the upstream SVGs are used unchanged, and an `<img>` never runs scripts inside an SVG.

### Light/dark variants as `{ light, dark? }`
```ts
const appLogos: Record<AppTokenApp, { light: string; dark?: string }> = {
  karakeep: { light: karakeep, dark: karakeepDark },
  linkwarden: { light: linkwarden },
};
```
`AppLogo` picks the variant with `useThemeMode()`, with the same fallback as `ProviderIcon`. The Karakeep logo would also look right with `fill="currentColor"`, but that would be a recolored copy, not the official dark asset, and it wouldn't fit a `<img>`.

### Size and placement
The `AppLogo` component lives in `appTokens/AppLogo.tsx` (since `AppIcon` is already taken). It renders at `size-8` (32 px) with `shrink-0`, as the first child of the `<li>`. With `items-start`, it lines up with the token name. The Linkwarden PNG is downscaled to 64×64 (2× for high-DPI screens) and optimized, so it stays a few KB. The 242 KB `icon.png` is not used.

### Theme tokens
The theming spec's "No theme-independent colors" applies to UI colors. Third-party logos are images, like passkey provider icons, so their own colors are expected.

### Provenance
Record each file's upstream repository, path and license in `logos/README.md`, so a later update knows where to get them.

### Exclude `logos/` from Biome
Biome lints SVGs under `src/` and flags the upstream SVGs for having no `<title>` (`a11y/noSvgWithoutTitle`). The files are vendored unchanged and shown as decorative `<img alt="">`, so `biome.json` excludes `src/features/settings/appTokens/logos`. Adding a `<title>` would edit the official asset for no gain, because a title inside an `<img>` isn't exposed anyway.

## Risks / Trade-offs

- [The Linkwarden tile is dark and filled, while the Karakeep mark is a bare glyph, so they look a bit different side by side] → Both are rounded squares at the same 32 px size, which keeps the row aligned. Accepted, since the user wants the official look.
- [Karakeep's black mark on dark surfaces] → Handled by the dark variant and `useThemeMode()`.
- [Upstream rebrands] → The bundled copies stay as they are until someone updates them using `logos/README.md`.
