# Proposal

## Why

Height is managed twice on a dashboard: sections and content widgets both offer "auto or N rows, then scroll", which allows scroll areas nested inside each other and already needed an exception for link widgets. Sections alone should decide height, and widgets should use the height they are given. Width is different, because a widget's width places it inside its section, so it stays. Its default is improved: today a new widget stores its section's current width as a number and stays narrow when the section is widened later.

## What Changes

- **BREAKING** Content widgets no longer have a height. It is removed from the layout API responses, the stored widget and the widget settings dialogs. A `height` sent for a content widget is ignored, so tabs still running the old frontend can save.
- The link widget's special rule for height is removed, because no widget has a height any more.
- A content widget that is the only child of a fixed-height section fills that section. With several children, each child keeps its natural height, and the section scrolls vertically when they need more room than it has. The section is the only vertical scroller. A widget's content that is too wide still scrolls horizontally inside the widget.
- A new widget starts with "Full width" / "Volle Breite". It stores the maximum width of 12, which rendering already caps at the parent's width, so the widget follows its section. The widget dialog shows that value as "Full width" instead of a column count. Widgets keep their optional width in columns, so side-by-side layouts stay possible.
- A migration carries over one case of existing data: an `auto`-height section whose only child is a content widget with a fixed height takes over that height. All other widget heights are dropped. No sections are added.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dashboard-layout`: content widgets have no height, an only-child widget fills a fixed-height section, the section is the only vertical scroller, and existing heights are migrated.
- `dashboard-editing`: new widgets start at full width, and widget dialogs no longer offer a height.
- `data-import`: exports with widget heights follow the migration's rule.
- `link-widget`: the link widget height requirement is removed.
- `media-widget`, `note-widget`, `favourite-explorer-widget`, `link-list-widget`, `note-list-widget`, `recent-favourites-widget`, `recent-notes-widget`, `selected-notes-widget`: their settings and rendering no longer mention a widget height.

## Impact

- **Backend:** `backend/internal/dashboards` (layout validation, JSON, stored config, `FitContent`, `StoredWidget`), `backend/internal/widgets/link.go`, and a new SQL migration in `backend/internal/db/migrations`.
- **Frontend:** `features/dashboards/layout` (`SectionGrid`, `SizeFields`, `LayoutEditor`), `features/dashboards/api.ts`, `WidgetSettingsForm`, `LinkSettings`, and the `en` and `de` locales.
- **API docs:** the content widget schema in `backend/internal/httpapi/openapi/boardly.yaml` loses `height`. No routes change.
- **Export and import:** the stored widget config loses `height`, and the export carries it as stored. The format stays at version 1, because a document without a widget height still reads as `auto` in every version. Import still accepts older exports that contain widget heights and applies the same single-child rule as the migration.
- **User-visible loss, accepted:** a widget with a fixed height next to other children becomes as tall as its content, unless its section has a fixed height.
