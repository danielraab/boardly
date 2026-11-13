## 1. Row

- [x] 1.1 In `FeedItemRow.tsx`, replace the `showSummary` prop with `summary: "preview" | "onDemand"`; update `FeedsPage` to pass `"preview"`
- [x] 1.2 For `"onDemand"`, outside edit mode and for an item with a summary, add an `Info` icon button first in the actions group (sized like the flag buttons, above the news link overlay) with `aria-label` "Show summary" / "Hide summary", `aria-expanded` and `aria-controls`; render the actions group when the row has audio, flags or this button
- [x] 1.3 When opened, show the whole summary in the summary grid area without the two-line clamp and without the "Show more" toggle; the open state is per row and not remembered
- [x] 1.4 Add `feeds.showSummary` / `feeds.hideSummary` to `en.json` ("Show summary" / "Hide summary") and `de.json` ("Zusammenfassung zeigen" / "Zusammenfassung verbergen")

## 2. Widget

- [x] 2.1 In `widgets/feed/index.tsx`, pass `summary={config.showSummary ? "preview" : "onDemand"}` to the rows

## 3. Tests

- [x] 3.1 Widget tests (`Feed.test.tsx`): with "Show summary" off, an owner's row with a summary has the info button and no summary; activating it shows the whole summary, sets `aria-expanded` and the label to "Hide summary", and doesn't open the link; activating it again hides it
- [x] 3.2 Widget tests: no info button for an item without a summary, in edit mode, or for a visitor's rows; with "Show summary" on, the preview and "Show more" behave as before and there is no info button
- [x] 3.3 Run the frontend checks and tests (`cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test`)

Export, import, backend and OpenAPI documents are unaffected; no tasks needed for them.
