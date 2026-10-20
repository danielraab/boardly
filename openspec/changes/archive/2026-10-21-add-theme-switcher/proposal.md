## Why

Famodash is a start page that people keep open all day, often on wall-mounted screens or at night, but it ships with a single fixed light palette. Users need a dark theme and a few distinct color themes to make their dashboard feel their own.

## What Changes

- Add 7 color themes: **White** and **Black** (neutral light and dark) plus **Sand**, **Sage** (light) and **Fjord**, **Dusk**, **Ember** (dark). The current blue-grey palette is replaced by White.
- Add a **System** choice, the default, that follows the OS `prefers-color-scheme` and shows White or Black, switching live when the OS setting changes.
- Add a theme switcher to the header: an icon button (palette icon) directly before the language switcher that opens a menu with System and the 7 themes, each with a color swatch.
- Remember an explicit choice in the browser (localStorage `famodash.theme`); without a stored choice, System applies.
- Apply the theme before the first paint so a reload never flashes the wrong theme.
- Set `color-scheme` per theme so scrollbars, form controls and autofill match light or dark themes.
- Replace the remaining hard-coded colors (`text-red-700` error texts, the red delete button, the fixed menu shadow color) with theme tokens (`danger`, `on-danger`, `shadow`) so they stay readable in every theme.
- Translate theme names and the switcher label (en + de).

## Capabilities

### New Capabilities
- `theming`: the set of themes and their color tokens, the System default, the theme menu, persistence in the browser, flash-free application and `color-scheme`.

### Modified Capabilities
- `app-shell`: the global header now shows the theme switcher next to the language switcher.

## Impact

- Frontend only; no backend or API changes.
- `frontend/src/styles.css`: theme token blocks per `[data-theme]`, new `danger`/`on-danger`/`shadow` tokens.
- `frontend/index.html`: small inline script that sets `data-theme` before the app loads.
- New `frontend/src/theme/` module (theme list, resolve/apply/store) and `frontend/src/components/ThemeSwitcher.tsx`; `AppShell.tsx` renders it.
- Components with hard-coded colors: `LoginPage`, `VerifyPage`, `SettingsPage`, `PasskeysTab`, `menuStyles.ts`.
- `frontend/src/locales/en.json`, `de.json`: new `shell.theme` and `theme.*` keys.
- Not in scope: user-chosen section colors for the future Section widget (see design.md for the chosen direction), and storing the theme per user on the server.
