# Design

## Context

The Delete button in `NodeForm` uses `secondaryButton` with an icon and a label. On narrow screens the row of Delete, Cancel and Save gets too wide.

## Goals / Non-Goals

**Goals:**
- A compact, clearly destructive Delete action that keeps its accessible name.

**Non-Goals:**
- Changing the other buttons or the confirmation.

## Decisions

- **New `dangerIconButton` in `buttonStyles.ts`:** it has the same square shape as `secondaryIconButton` (2.375rem, the height of the text buttons), with a `border-danger/40` border, `text-danger` text and a `hover:bg-danger/10` background. It gets its own constant because adding `text-danger` on top of `secondaryIconButton` would compete with that style's `text-ink`, and with Tailwind the winner depends on CSS order, not on class order.
- **Accessible name:** `aria-label` and `title` use the existing `favourites.dialog.delete` key, so the tests that look for the "Delete" button still work.

## Risks / Trade-offs

- [An icon-only button is less obvious than a labelled one] → The trash icon in red is a common pattern, and the tooltip shows the label.
