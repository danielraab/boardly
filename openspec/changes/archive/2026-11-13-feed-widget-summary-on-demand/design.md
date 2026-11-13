## Context

`FeedItemRow` renders an item on `/feeds` and in the feed widget. It takes `showSummary: boolean`; when true and the item has a summary, a `Summary` component shows it clamped to two lines with a "Show more" / "Show less" toggle. The widget passes `config.showSummary`; `/feeds` always passes true.

The data is already there: the owner's widget data contains full items, summary included, whatever the setting. For visitors, `backend/internal/widgets/feed.go` copies the summary only with `ShowSummary`, so visitors' items have `summary` missing when the setting is off.

The row is a CSS grid with areas `icon / meta / title / summary / actions`. Narrow (< 480 px) puts the actions on their own line below the text; wide puts them in a column on the right. In the widget the owner always has flag buttons, so an owner's narrow row always has that actions line.

## Goals / Non-Goals

**Goals:**
- Let the owner read an item's summary from a widget with "Show summary" off, without making the rows taller until they ask for it.

**Non-Goals:**
- A new widget setting or a change to the saved config.
- Summaries for visitors on widgets with "Show summary" off.
- Changing `/feeds`, or moving the row's buttons into an overflow menu (a possible later change).

## Decisions

**The summary mode is a three-way prop.** Replace `FeedItemRow`'s `showSummary: boolean` with `summary: "preview" | "onDemand"`. `FeedsPage` passes `"preview"`; the widget passes `"preview"` with the setting on and `"onDemand"` with it off. The widget config keeps its `showSummary` field. An enum avoids a meaningless combination of two booleans (preview and on-demand), and the type checker finds every caller. Alternative: derive on-demand inside the row from `!showSummary` — rejected, because `/feeds` and other callers of `FeedItemRow` would suddenly gain the button.

**The info button lives in the actions group, first.** It's a `lucide-react` `Info` icon button sized like the flag buttons (`style.icon`), placed before the play button, with `aria-label` "Show summary" / "Hide summary", `aria-expanded` and `aria-controls` pointing at the summary. Being in the actions group keeps it on the existing buttons line when narrow and to the right when wide, so it adds no line. The actions group is rendered when the row has audio, flags or the info button. Like the other buttons it sits above the news link's overlay (`relative z-10`), so it never opens the link.

**The opened summary reuses the summary grid area, unclamped.** The row keeps an `open` state; when open, the summary text renders in `[grid-area:summary]` with the same muted style as the preview but without `line-clamp-2` and without the "Show more" toggle. Not remembered: the state is per mounted row.

**No info button without a summary, in edit mode, or for visitors.** The button is shown only when `!editing` and `item.summary` is non-empty. Visitors' items carry no summary in this mode, so no backend or visitor-specific check is needed.

**Strings.** New keys `feeds.showSummary` / `feeds.hideSummary` (en: "Show summary" / "Hide summary"; de: "Zusammenfassung zeigen" / "Zusammenfassung verbergen"), next to the existing `feeds.showMore` / `feeds.showLess`.

## Risks / Trade-offs

- [A visitor's row has no button where the owner's has one] → Intended: the owner chose not to publish summaries. Documented in the spec.
- [A visitor's narrow news row has no actions line, so if a future change sent summaries to visitors, the button would add a line there] → Out of scope; the overflow-menu idea would address it.
- [Prop rename touches every `FeedItemRow` caller] → Few callers (`FeedsPage`, the widget, tests); the type checker finds them all.
