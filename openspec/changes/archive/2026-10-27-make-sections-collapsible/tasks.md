# Tasks

## 1. Collapse state

- [x] 1.1 Add a collapsed-sections context in `layout/SectionGrid.tsx`, filled in `DashboardGrid` from `useStoredIdSet("boardly.collapsedSections")`, with a toggle function; verify with a unit test that toggling writes the ID set to localStorage

## 2. Section rendering

- [x] 2.1 In `SectionFrame`, outside edit mode and for titled sections, render a chevron toggle button before the title with `aria-expanded`, `aria-controls` and a translated label; verify the button appears only for titled sections in view mode (Vitest)
- [x] 2.2 When collapsed, don't render the section's children and drop the fixed height and scroll; verify with a test that children disappear and reappear on toggle, and a fixed-height section loses its height style
- [x] 2.3 Ignore the collapsed state in edit mode; verify with a test that a stored collapsed section renders expanded without a toggle in the editor
- [x] 2.4 Add `dashboards.section.expand` / `dashboards.section.collapse` to `en.json` and `de.json`; verify the i18n key-parity test passes

## 3. Verification

- [x] 3.1 Add a test that the collapsed state is restored from localStorage on a fresh render (reload), then run `make check` and `make test` and verify both pass
