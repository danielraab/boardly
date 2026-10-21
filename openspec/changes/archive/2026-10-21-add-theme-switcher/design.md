## Context

All UI colors already come from 6 Tailwind v4 tokens in `frontend/src/styles.css` (`paper`, `surface`, `ink`, `muted`, `line`, `accent`), used through utilities such as `bg-surface`, `text-muted`, `border-line` and `bg-ink/85`. A few places bypass them: `text-red-700` error texts (Login, Verify, Settings, Passkeys), the `bg-red-700 text-white` delete button in `PasskeysTab.tsx`, and the literal shadow color `rgb(29 39 51/0.35)` in `menuStyles.ts` and `PasskeysTab.tsx`.

The language switcher (`LanguageSwitcher.tsx`, Base UI `Menu.RadioGroup`, `headerButton`/`menuPopup`/`menuItem` styles) and its persistence (`famodash.lang` in localStorage, only explicit choices stored) set the pattern this change follows. The Go server sends no Content-Security-Policy, so an inline script in `index.html` is allowed.

## Goals / Non-Goals

**Goals:**
- 7 themes + System, switchable without reload, remembered per browser.
- No component has to know which theme is active: themes are pure token swaps.
- No flash of the wrong theme on load.
- Every theme meets the contrast minimums from the spec.

**Non-Goals:**
- Storing the theme per user on the server or syncing it across devices.
- User-defined themes or custom colors.
- Section widget colors (direction recorded below, implemented with the widget).
- Updating `<meta name="theme-color">` for mobile browser chrome (can follow later).

## Decisions

### 1. A theme is a `[data-theme]` block that redefines the CSS variables

`@theme` keeps the White values as defaults, so the generated utilities keep pointing at `var(--color-…)`. Each other theme is a plain CSS rule:

```css
[data-theme="dusk"] {
	color-scheme: dark;
	--color-paper: #1e1a2a;
	/* … all 9 tokens … */
}
```

The resolved theme id is set as `data-theme` on `<html>`. Opacity variants like `bg-ink/85` keep working because Tailwind v4 builds them with `color-mix()` on the variable.

The selector is `[data-theme="…"]`, not `:root[data-theme="…"]`, on purpose: the menu swatches set `data-theme` on their own element and so render with that theme's real tokens (see 6). This keeps a single source of truth for the colors.

*Alternatives:* Tailwind `dark:` variants (only two modes and touches every component); one CSS file per theme loaded on demand (extra request, and causes a flash on switch).

### 2. Palettes

White replaces the current blue-grey palette. Values (all checked against the contrast requirement; lowest ratio per pair is ≥ 4.4:1 for accent, ≥ 5.5:1 for muted text):

| Theme | Mode | paper | surface | ink | muted | line | accent | danger | on-danger |
|---|---|---|---|---|---|---|---|---|---|
| white | light | `#f4f4f5` | `#ffffff` | `#18181b` | `#5c5c66` | `#d4d4d8` | `#2563eb` | `#b91c1c` | `#ffffff` |
| black | dark | `#0a0a0a` | `#18181b` | `#f4f4f5` | `#a1a1aa` | `#2e2e33` | `#60a5fa` | `#f87171` | `#0a0a0a` |
| sand | light | `#f4efe6` | `#fffaf2` | `#2b2620` | `#685e52` | `#ddd3c4` | `#b4532a` | `#a61b1b` | `#ffffff` |
| sage | light | `#edf2ec` | `#fafcf8` | `#1f2a22` | `#536357` | `#cbd7ca` | `#2f7a4a` | `#b42318` | `#ffffff` |
| fjord | dark | `#242933` | `#2e3440` | `#eceff4` | `#a3adbf` | `#434c5e` | `#88c0d0` | `#f0939b` | `#242933` |
| dusk | dark | `#1e1a2a` | `#29233a` | `#ece6f5` | `#ab9fc2` | `#3c3350` | `#e58fb4` | `#ff8f8f` | `#1e1a2a` |
| ember | dark | `#1f1b18` | `#2a2420` | `#f2e9df` | `#b0a291` | `#40372f` | `#e9a23b` | `#f28b82` | `#1f1b18` |

`shadow` is `rgb(0 0 0 / 0.18)` for light themes and `rgb(0 0 0 / 0.6)` for dark themes. Popups use `shadow-[0_8px_24px_-12px_var(--color-shadow)]`.

The existing `bg-ink text-paper` primary buttons need no change: they invert automatically in every theme.

### 3. Choice vs. resolved theme

Two values are kept apart:

```
choice   = "system" | ThemeId        ← stored in localStorage, shown as selected in the menu
resolved = ThemeId                   ← written to <html data-theme>
resolve("system") = matchMedia("(prefers-color-scheme: dark)").matches ? "black" : "white"
```

System is resolved in JavaScript, not with a CSS `@media` block, so the Black values aren't duplicated and there is one code path. A `matchMedia` change listener re-applies the theme while the choice is `system`.

### 4. Theme module and React binding

`frontend/src/theme/index.ts` owns the theme list (`{ id, mode }`, in menu order), `THEME_STORAGE_KEY = "famodash.theme"`, `readChoice()`, `resolveTheme()`, `applyTheme()`, `setTheme(choice)` and a tiny subscribe/notify store. `initTheme()` runs once in `main.tsx`, applies the theme (idempotent with the inline script) and installs the `matchMedia` listener. `ThemeSwitcher` reads the choice with `useSyncExternalStore`. Storage access is wrapped in try/catch like `setLanguage`.

Unlike language, choosing System *does* store `system`. This is equivalent to no stored value, but it makes the explicit choice visible when debugging and keeps `setTheme` free of special cases.

Besides `useTheme()` (the choice, for the menu) the module offers `useThemeMode()`, which returns `"light"` or `"dark"` for the resolved theme. Its store also notifies on OS color scheme changes, so components that pick light or dark images (passkey provider icons) re-render when System flips.

*Alternative:* a React context provider. Rejected: the state lives outside React anyway (the inline script and `matchMedia`), and a module store is simpler to test.

### 5. Flash-free start: inline script in `index.html`

A small synchronous script in `<head>`, before the module script, reads `famodash.theme`, validates it against the theme ids, resolves System and sets `data-theme` on `document.documentElement`. Since CSS for `[data-theme]` lives in the bundle's stylesheet, the first paint after the stylesheet loads already uses the correct theme.

The script duplicates the storage key and the list of ids. A Vitest test parses the script from `index.html` and asserts that its id list and key equal the theme module's, so they can't drift apart.

*Alternative:* inject the script via a Vite `transformIndexHtml` plugin generated from the theme module. More machinery than a ~10-line script plus a test.

### 6. Theme switcher UI

`ThemeSwitcher.tsx` mirrors `LanguageSwitcher.tsx`: Base UI `Menu` with a `RadioGroup`, `headerButton` trigger with the lucide `Palette` icon and `aria-label={t("shell.theme")}`, `align="end"`. Order: System, a separator, then the 7 themes. Each item shows the check indicator, a swatch and the name.

The swatch is a small circle with `data-theme={id}`, `bg-paper`, a `border-line` ring and an inner `bg-accent` dot, so it shows the theme's own colors. System's swatch is a half White / half Black circle.

`AppShell` renders `<ThemeSwitcher />` before `<LanguageSwitcher />`.

### 7. i18n keys

`shell.theme` ("Theme" / "Farbschema") and `theme.system`, `theme.white`, `theme.black`, `theme.sand`, `theme.sage`, `theme.fjord`, `theme.dusk`, `theme.ember` (de: System, Weiß, Schwarz, Sand, Salbei, Fjord, Dämmerung, Glut). The existing key-parity test covers both catalogs.

### 8. Direction for future section colors (not implemented here)

Recorded so the Section widget change can build on it: sections will store a **named color** (e.g. `"blue"`), not a hex value. Each name maps to an OKLCH hue, and each theme defines one lightness/chroma pair (`--section-l`, `--section-c`), so the color is `oklch(var(--section-l) var(--section-c) <hue>)`. This keeps every section color readable in every theme. A "custom hex" escape hatch can be added later without migrating stored names. These variables are added with the Section widget, not now.

## Risks / Trade-offs

- [The inline script and the theme module drift apart] → test that compares ids and key (decision 5).
- [A future component uses a raw Tailwind color like `text-red-700` again] → spec requirement "No theme-independent colors"; a review note in tasks to grep for raw palette classes. A lint rule can follow if this recurs.
- [Contrast regressions when a palette is tweaked] → a unit test computes the spec's contrast pairs from the token values and fails below the minimums. This needs the values in one machine-readable place: the test parses the `[data-theme]` blocks from `styles.css`.
- [jsdom has no `matchMedia`] → tests stub `window.matchMedia` in `test/setup.ts`, with a controllable `matches` and change dispatch for the System scenarios.
- [Changing the default palette is visible to existing users] → intended: White replaces the current look. No migration needed since nothing is stored yet.

## Migration Plan

Frontend-only and stateless; deploy as usual. Rollback is reverting the change: a leftover `famodash.theme` key is ignored by the old version.
