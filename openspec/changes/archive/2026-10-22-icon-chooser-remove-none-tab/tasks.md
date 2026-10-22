# Tasks

## 1. Icon chooser

- [x] 1.1 In `IconChooser.tsx`, render only the "Website icon", "Own image" and "Symbol" tabs and remove the "None" panel; verify the chooser shows three tabs
- [x] 1.2 Open the chooser on "Website icon" when the current kind is "None"; verify with a test that opens the chooser of a widget without icon
- [x] 1.3 Make the tab list `flex-nowrap` with `whitespace-nowrap`, `px-2` and `min-w-0 truncate` tabs; verify in the browser at 320px and 360px width in German that the tabs stay on one line
- [x] 1.4 Add the "Remove icon" button to the footer left of "Done", pressed with a check and "No icon" when the kind is "None"; verify with a test that it sets the kind to "None" and keeps focus

## 2. Strings and tests

- [x] 2.1 Add `iconChooser.remove` and `iconChooser.removed` and drop `iconChooser.none.*` in `en.json` and `de.json`; verify `pnpm exec tsc --noEmit` passes
- [x] 2.2 Update `Link.test.tsx` to remove the icon via "Remove icon" instead of the "None" tab; verify `pnpm exec vitest run` and `pnpm exec biome check` pass
