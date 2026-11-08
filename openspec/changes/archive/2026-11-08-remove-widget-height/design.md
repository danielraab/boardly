# Design

## Context

See proposal.md for the motivation. Layouts are stored in `dashboard_widgets`, one row per section or content widget. A section's `config` column holds `{title, border, height}`, and a content widget's holds `{height, config}`. The export copies that column as it is, and `dashboards.StoredWidget` turns it back into a widget on import.

`SectionFrame` makes a fixed-height section scroll vertically. `WidgetFrame` has its own height and an `overflow-auto` area. `min(width, parentColumns)` already caps a widget's width at its parent's width on every viewport.

## Goals / Non-Goals

**Goals:**

- One place for height, and one vertical scroller per section.
- Unchanged export format version, with old exports still importable.

**Non-Goals:**

- Removing or moving a widget's width. Side-by-side layouts stay as they are.
- Adding sections during migration or import.
- Changing the width of existing widgets. Those already as wide as their section keep their number and don't follow the section. A later change could set them to 12 without any visible difference.
- Filling a section with several children, or with a nested section as its only child.

## Decisions

### Keep the stored envelope, without `height`

A content widget's `config` column stays an object with a nested `config`, now `{config}`. `rewriteIDs` and the export keep working on the same shape.

Unwrapping it to the bare kind config was considered. It would be tidier, but it would make old and new exports differ in structure and force a format version 2. A missing `height` decodes as `auto` in old and new code, so version 1 remains correct.

### Ignore a height sent for a content widget

`normalizeContent` always clears the height, which replaces the `FitContent` branch. The range check moves to sections only. Tabs running the old frontend or a cached PWA still send `height` and can save without an error. Rejecting it would be stricter, but it would fail exactly those saves. A malformed value still fails JSON decoding of `Height`, as it does today.

### Migration in SQL

Migration `0026` needs no new rows and no new IDs, so it fits the SQL-only migration runner:

1. For sections with `$.height` equal to `auto` or missing (sections stored before heights existed) and exactly one child, where that child isn't a section and its `$.height` has the JSON type `integer`, copy the child's height to the section.
2. Remove `$.height` from every row whose `kind` isn't `section`.

No earlier migration uses `json_set` or `json_remove`, but the schema already relies on SQLite's JSON support through `json_valid`. The migration test proves the functions work on the shipped driver.

### Import applies the same rule in Go

`StoredWidget` reads an old `height` from a content widget's config and keeps it on the in-memory `Widget`. When it builds a section with height `auto` and exactly one child, and that child is a content widget with a height, the section takes it over. `normalizeLayout` clears content heights before saving. Children are converted before their parent, so the parent always sees its final children.

The rule exists twice, in SQL and in Go. That is acceptable: it is short, the migration runs once per installation, and both have tests against the same scenarios.

### Fill with the grid's first row

When a fixed-height section isn't collapsed and has exactly one content widget, its children's grid becomes a flexible item of the section (`flex-1`), with `grid-template-rows: minmax(min-content, 1fr)` and automatic further rows. The first row then takes up the free height but never shrinks below the content, so a too-tall widget makes the section scroll instead of being squeezed. The automatic rows keep the "Add section" and "Add widget" row of edit mode at its natural height.

Giving the widget a fixed percentage height was considered. It would clip content and bring back a second vertical scroller.

`WidgetFrame`'s content area keeps horizontal scrolling. It must not constrain the height, so no vertical scrollbar can appear inside a widget. This depends on browser layout, which Vitest's DOM doesn't compute, so it needs a check in a real browser.

The link widget and list widgets fill the section as well, but their content simply starts at the top. Only the note widget's card has a surface that visibly stretches (`min-h-full`), so no per-kind exception is needed.

### "Full width" is the stored value 12

`MaxColumns` is already valid and already capped when rendering, so no new field or API value is needed. A separate `"full"` value was considered, as the section's `height` has `"auto"`. It would change the API, the column type and the export for no behaviour gain.

Only the widget dialog shows the "Full width" option. Section dialogs keep their column counts. On a dashboard with 12 desktop columns, "12 columns" and "Full width" are the same, so the dialog lists only "Full width".

`useSizeState` and `SizeFields` keep the height for the section dialog. `WidgetSettingsForm` uses the width part only and hands `{ width }` to `onApply`, which the kinds already spread into the widget.

## Risks / Trade-offs

- [A tall list loses its own scroll area and makes the page longer] → Accepted. The owner can set a height on its section.
- [A fixed height moved to a section includes the heading and padding, so the widget gets slightly less room] → Accepted. Its content scrolls with the section.
- [The fill relies on a chain of layout rules that unit tests can't see] → Check it in a browser, including the note widget, a too-long note and edit mode.
- [A heading now scrolls away in a section whose only widget used to have the fixed height] → Accepted. Fixed-height sections already behave like this.

## Migration Plan

The migration runs on startup like all others. Rolling back to an older binary after it has run is safe for reading, since a missing height means `auto`, but the dropped widget heights aren't restored.
