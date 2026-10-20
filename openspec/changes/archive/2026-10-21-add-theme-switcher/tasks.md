# Tasks

## 1. Theme tokens

- [x] 1.1 In `styles.css`, replace the `@theme` defaults with the White palette and add the `danger`, `on-danger` and `shadow` tokens. Verify `pnpm build` succeeds and the app shows the neutral White look
- [x] 1.2 Add `[data-theme="…"]` blocks for all 7 themes with the palettes from design.md and `color-scheme: light|dark`. Verify by setting `data-theme` by hand in the browser devtools that each theme recolors the page
- [x] 1.3 Add a Vitest test that parses the `[data-theme]` blocks from `styles.css`, checks that every theme defines all 9 tokens and that the spec's contrast pairs meet their minimums. Verify it passes, and fails when a muted value is changed to a low-contrast one

## 2. Remove hard-coded colors

- [x] 2.1 Replace `text-red-700` (also `hover:text-red-700`) with `text-danger` in `LoginPage`, `VerifyPage`, `SettingsPage`, `PasskeysTab` and the favourites page and dialogs, and the `dangerButton` in `dialogStyles.ts`: `bg-red-700`/`bg-red-800`/`text-white` with `bg-danger`/`text-on-danger` (hover via opacity, e.g. `hover:bg-danger/90`)
- [x] 2.2 Replace the literal `rgb(29_39_51/0.35)` shadows in `menuStyles.ts` and `dialogStyles.ts` with `var(--color-shadow)`. Verify `grep -rnE "(bg|text|border)-(red|white|black|gray|slate|zinc)-?|rgb\(" frontend/src --include=*.tsx --include=*.ts` returns no hits outside tests

## 3. Theme module

- [x] 3.1 Create `src/theme/index.ts` with the theme list (`id`, `mode`), `THEME_STORAGE_KEY`, `readChoice`, `resolveTheme`, `applyTheme`, `setTheme`, subscribe/notify and `initTheme` (applies the theme and listens to `prefers-color-scheme` changes). Wrap storage access in try/catch
- [x] 3.2 Add a controllable `window.matchMedia` stub to `test/setup.ts` and reset `data-theme` between tests
- [x] 3.3 Add tests for: no stored value + light/dark OS → white/black; OS change while `system` switches live; OS change with an explicit theme does nothing; stored `neon` → system; `setTheme` stores the choice and sets `data-theme`; storage throwing still applies the theme
- [x] 3.4 Call `initTheme()` in `main.tsx` before rendering
- [x] 3.5 Add `useThemeMode()` (notifies on theme and OS changes) and use it in `PasskeysTab`'s `ProviderIcon` to prefer `iconDark` in dark themes. Verify tests for the dark icon in `fjord`, the light fallback in `black`, and the light icon in `white`

## 4. Flash-free start

- [x] 4.1 Add the inline `<head>` script to `index.html` that reads `famodash.theme`, validates it, resolves System and sets `data-theme` on `<html>`, before the module script
- [x] 4.2 Add a test that extracts the script from `index.html`, runs it against a stubbed localStorage/matchMedia for a stored theme, `system` and an invalid value, and asserts that its id list and storage key equal the theme module's
- [x] 4.3 Verify manually: with `ember` stored and CPU throttling in devtools, a reload shows no White flash

## 5. Theme switcher

- [x] 5.1 Add i18n keys `shell.theme` and `theme.system|white|black|sand|sage|fjord|dusk|ember` to `en.json` and `de.json`. Verify the key-parity test passes
- [x] 5.2 Build `components/ThemeSwitcher.tsx` (Base UI Menu, `Palette` icon trigger with `headerButton` and accessible name, RadioGroup with System, separator, then the 7 themes, each with check indicator, swatch via `data-theme={id}` and translated name; System swatch half White / half Black)
- [x] 5.3 Render `<ThemeSwitcher />` before `<LanguageSwitcher />` in `AppShell`
- [x] 5.4 Add tests: header shows the theme button before the language button; menu lists System + 7 themes in order with System selected by default; choosing Sage closes the menu, sets `data-theme="sage"` and stores it; Enter then Escape returns focus to the button; names switch to German with the language

## 6. Wrap-up

- [x] 6.1 Check every page (landing, login, verify, home, settings incl. passkey dialog, not-found) in all 7 themes for leftover unreadable or hard-coded colors
- [x] 6.2 Mention theme switching in README's feature list
- [x] 6.3 Run `make check` and `make test` and verify both pass
