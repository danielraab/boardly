# Tasks

## 1. Backend: separate limits

- [x] 1.1 Replace `maxTextLength` in `backend/internal/feeds/parse.go` with `maxTitleLength = 500` and `maxSummaryLength = 1000`, and use them in `parse.go` and in `insertImported` in `refresh.go`; verify with `go build ./...`
- [x] 1.2 Add parse tests for a 1500-character description (summary is 1000 characters) and an 800-character title (title is 500 characters), and a store test that a refresh replaces a 500-character summary with the longer one; verify with `go test ./internal/feeds`
- [x] 1.3 Add an import test that an imported item keeps a 1000-character summary; verify with `go test ./internal/dataimport`

## 2. Frontend: row layout

- [x] 2.1 Make the row in `FeedItemRow.tsx` a `@container` with grid areas for icon, meta, title, summary and actions, switching at `@[480px]` as in design.md, with the title clamped to two lines in the narrow layout and the empty bottom slot for the later progress bar; verify by checking `/feeds` at 375 px and at 1280 px in the dev server
- [x] 2.2 Check the feed widget at sizes S, M and L in a narrow and a wide column, with and without flags (owner and visitor), and adjust spacing so that the S/M/L scale still applies; verify by hand in the dev server
- [x] 2.3 Keep the existing tests in `Feeds.test.tsx` and `widgets/feed/Feed.test.tsx` passing unchanged (same accessible names and order); verify with `pnpm exec vitest run src/features/feeds src/features/dashboards/widgets/feed`

## 3. Frontend: summary toggle

- [x] 3.1 Add the "Show more" / "Show less" toggle with `aria-expanded` and `aria-controls`, shown only when the summary is cut off (measured on mount and on resize), above the link layer, and not in edit mode; add translations `feeds.showMore` / `feeds.showLess` in en and de; verify with the tests of 3.2
- [x] 3.2 Add tests: the toggle appears for a cut-off summary (stubbed `scrollHeight`) and not for a short one, expanding doesn't open a link, the toggle works for a visitor's widget, and it is absent in edit mode; verify with `pnpm exec vitest run src/features/feeds src/features/dashboards/widgets/feed`

## 4. Finish

- [x] 4.1 Run the full checks of both sides once (`make check` and `make test`) and fix what fails
