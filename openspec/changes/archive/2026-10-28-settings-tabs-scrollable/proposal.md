# Proposal

## Why

On a phone the settings tab bar ("Profile", "Passkeys", "Connected apps", "Import / Export") is wider than the screen. The tabs either squeeze and wrap their labels or push the page wider than the viewport, which causes horizontal page scrolling.

## What Changes

- The settings tab bar scrolls horizontally on its own when its tabs don't fit, instead of wrapping labels or widening the page.
- Tab labels stay on one line.
- The selected tab is scrolled into view, so opening `/settings?tab=data` on a phone shows the selected tab.

## Capabilities

### New Capabilities

### Modified Capabilities
- `user-profile`: the "Settings tabs" requirement gains behaviour for narrow screens.

## Impact

- `frontend/src/features/settings/SettingsPage.tsx` (tab list and tab styles).
- No API or backend changes.
