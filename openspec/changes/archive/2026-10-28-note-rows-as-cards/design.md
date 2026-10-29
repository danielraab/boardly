# Design

## Context

All three note widgets render their rows with the shared `widgets/NoteRows.tsx`. The server rows come from `recentNote` (`recentnotes.go`) and `listedNote` (`notelist.go`, also used by selected notes); neither has a colour. Note cards use `rounded-xl border border-line` with a small shadow and `noteBackground(color)` from `features/notes/colours.ts`, or `bg-surface` without a colour.

## Goals / Non-Goals

**Goals:**
- One change in `NoteRows` covers all three widgets.
- Rows read as notes but stay compact: the size still sets the padding.

**Non-Goals:**
- Showing content, the category badge's colour or icon, or any other card element.
- A setting to turn the card look off.

## Decisions

**Row style.** Each row gets `rounded-lg border border-line` with `px-2` and the size's vertical padding, `noteBackground(color)` as inline background or `bg-surface`, and the list a `gap-1` (S) to `gap-1.5` (L) flex column. `rounded-lg` rather than the cards' `rounded-xl`, because rows are much smaller. No shadow, so a long list doesn't get busy. The hover state of the owner's button rows becomes a slightly darker ring/border (`hover:border-ink/30`) instead of `hover:bg-paper`, which would hide the colour.

**Colour in the data.** `Color *string` with `json:"color"` on `recentNote` and `listedNote`, filled from `n.Color`. The frontend `NoteRow` type gets `color: NoteColour | null`. Visitors get it too, since it's now shown; it reveals nothing beyond what the page displays.

## Risks / Trade-offs

- [Text contrast on coloured backgrounds] → The note colours are theme tokens already used behind note text on cards, so `ink`/`muted` stay readable.
- [Existing tests assert visitor data without `color`] → Updated with the new field.
