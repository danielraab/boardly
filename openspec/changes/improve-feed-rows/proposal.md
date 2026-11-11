# Proposal

## Why

On a phone, a feed item row puts the play button and the two flag buttons to the right of the text. The title and the feed's title get squeezed into a narrow column, and the feed's title, which shares its line with the time, is cut off first, while the space under the buttons stays empty. Summaries are cut after two lines with no way to read the rest, and they are stored with only 500 characters, which ends most podcast descriptions mid-sentence.

## What Changes

- Item rows on `/feeds` and in the feed widget get a layout for narrow rows: the feed's icon, title and time on their own first line, the item's title over the full width on up to two lines, the summary below, and the buttons on a line under the text, in the space that is empty today. Wide rows keep today's layout. The switch depends on the row's width, not the viewport's, so a narrow widget column on a desktop gets the narrow layout as well.
- A summary that doesn't fit in two lines gets a "Show more" / "Mehr anzeigen" toggle that expands it in place, and "Show less" / "Weniger anzeigen" collapses it again. This works on `/feeds` and in feed widgets that show summaries, also for visitors of public dashboards, and not in edit mode.
- Summaries are stored with up to 1000 characters instead of 500, for fetched and for imported items. Titles stay at 500.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `feeds`: "Item rows" gets the narrow layout and the summary toggle. "Item content" stores summaries of up to 1000 characters.
- `feed-widget`: "Feed widget rendering" gets the narrow layout and the summary toggle.

## Impact

- Frontend: `frontend/src/features/feeds/FeedItemRow.tsx` (layout, toggle), translations (en, de), tests in `Feeds.test.tsx` and `widgets/feed/Feed.test.tsx`.
- Backend: `backend/internal/feeds/parse.go` and `refresh.go` (separate limits for title and summary), with parse tests.
- API: no change in shape, `summary` can only be longer. `boardly.yaml` states no maximum length for it, so the OpenAPI documents stay as they are.
- Export and import are unaffected: the `summary` field keeps its shape and only gets longer. An import accepts summaries of up to 1000 characters, and an older server that imports a newer export cuts them to 500. The format version stays at 2.
- Existing items keep their 500-character summaries until a refresh of their feed delivers them again.
