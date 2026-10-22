# Proposal

## Why

The icon chooser of the link widget has four tabs in a popup that is at most 22rem wide. With the German labels, and in both languages on phones, the tab row wraps onto a second line, which looks broken. "None" doesn't need a tab of its own: it has nothing to show or configure, it's a single action.

## What Changes

- The icon chooser shows three tabs: "Website icon", "Own image" and "Symbol". The "None" tab is removed.
- The chooser's footer gets a "Remove icon" / "Icon entfernen" button next to "Done". It sets the icon kind to "None" (the widget shows a globe).
- When the current icon is "None", the chooser opens on the "Website icon" tab.
- The tab row doesn't wrap.

## Capabilities

### New Capabilities

### Modified Capabilities
- `link-widget`: the "Link widget icon" requirement changes from "the four kinds as tabs" to three tabs plus a "Remove icon" action.

## Impact

- `frontend/src/features/dashboards/widgets/link/IconChooser.tsx`: tabs, footer.
- `frontend/src/locales/en.json`, `de.json`: new strings, unused "None" tab strings removed.
- `frontend/src/features/dashboards/widgets/link/Link.test.tsx`: the test that picks "None" via its tab.
- No backend or API changes; the stored icon kind `none` stays as it is.
