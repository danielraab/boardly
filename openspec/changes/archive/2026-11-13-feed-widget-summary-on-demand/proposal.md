## Why

A feed widget with "Show summary" off is compact, but its items' summaries can't be read from the dashboard at all; the only way is to switch the whole widget to two-line previews, which costs height on every row. The owner should be able to keep the compact rows and still open an item's summary when it's of interest.

## What Changes

- With "Show summary" off, each row of an item with a summary gets an info button among its buttons. Activating it shows the whole summary in the row, unclamped; activating it again hides it. It doesn't open the item's link or start playback, isn't shown in edit mode, and the open state isn't remembered.
- The button sits with the row's other buttons (to the right when wide, on the buttons line when narrow), so it adds no line of its own to an owner's row.
- With "Show summary" on, nothing changes (two-line preview with "Show more" / "Show less").
- Visitors are unaffected: their items carry no summary when "Show summary" is off, so they get no info button.
- No change to the widget's settings, its saved config, the API or the backend.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `feed-widget`: the rendering requirement gains the on-demand summary for widgets with "Show summary" off.

## Impact

- Frontend: `frontend/src/features/feeds/FeedItemRow.tsx` (info button and on-demand summary for the widget), `frontend/src/features/dashboards/widgets/feed/index.tsx` (passes the mode), `frontend/src/locales/en.json` and `de.json`, and the widget tests.
- `/feeds` is unchanged; it always shows the summary preview.
- Backend, OpenAPI documents: unaffected.
- Export and import are unaffected: no user data or widget config changes.
