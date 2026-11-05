# Design

## Context

The footer of `FavouriteDetails.tsx` is `flex justify-end gap-3` with two `secondaryButton`s and one `primaryButton`, each with `px-4` and a text label.

## Goals / Non-Goals

**Goals:** a footer that fits narrow dialogs in both languages.

**Non-Goals:** changing the offline copy dialog, or the order or behaviour of the actions.

## Decisions

- **Icon-only "Offline copy".** It is the least-used of the three actions. "Edit" and "Open" keep their labels, because they are the main actions and "Open" is the primary button. The app already uses `secondaryIconButton` with `aria-label` and `title` for secondary actions, for example the dashboard's actions menu and the layout editor's cancel button, so this one matches them.
- **`flex-wrap` on the footer** is a fallback for very narrow widths or long labels, so the buttons never overflow the dialog.
- **Alternative considered:** making all three buttons smaller (`px-3`, smaller text). That was rejected: they would be inconsistent with the other dialogs, and German labels can still overflow.

## Risks / Trade-offs

- An icon alone is less discoverable than a label. The tooltip and the accessible name help; the archive icon already appears next to the label today.
