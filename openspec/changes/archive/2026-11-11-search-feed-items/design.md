# Design

## Context

`Store.Items` (`backend/internal/feeds/feeds.go`) builds one SQL query per view from `itemFrom`, adds `WHERE` conditions for subscriptions (`s.id IN (...)`), audio and the cursor, and pages with a keyset cursor on the view's sort key. The handler already maps `feedId` to `SubscriptionIDs`. SQLite's `LIKE` and `lower()` only fold ASCII, so "Ärger" wouldn't match "ärger". The `tags` table works around this with a stored `name_key`. Items are shared between users and there are at most `BOARDLY_FEED_LATEST_ITEMS` per feed plus flagged ones, a few thousand rows for a typical user.

## Goals / Non-Goals

**Goals:**
- Unicode case-insensitive substring search over title and summary, combined with the existing views, feed filter and paging.
- No change to how items are stored.

**Non-Goals:**
- Ranking by relevance, stemming, phrase search or highlighting.
- Searching items the server no longer keeps.
- Search in the feed widget.

## Decisions

### A registered SQL function instead of FTS5 or a stored key
The `db` package registers a deterministic scalar function `boardly_fold(text)` with `modernc.org/sqlite` (`RegisterDeterministicScalarFunction`), which returns `strings.ToLower` of its argument, or `NULL` for `NULL`. The query folds each word in Go the same way and adds, per word:

```sql
AND (instr(boardly_fold(i.title), ?) > 0 OR instr(boardly_fold(COALESCE(i.summary, '')), ?) > 0)
```

`instr` avoids escaping `%` and `_` as `LIKE` would need.

Alternatives:
- **FTS5**: relevance ranking and speed that the data size doesn't need, plus a virtual table and triggers that have to be kept in step with item merges and cleanup. It also only does token matching by default, so "produc" wouldn't find "production".
- **Stored folded column** (like `tags.name_key`): needs a migration and filling existing rows, which SQL alone can't fold, so it needs a backfill in Go. It also doubles the text per item.
- **`LIKE`**: ASCII-only case folding.

The scan stays bounded by the joins of the view (the user's subscriptions or flags), so it reads only the user's items.

### Words
`q` is trimmed and split with `strings.Fields`, and empty means no filter. The length check (at most 200 characters) runs on the raw parameter in the handler and maps to `ErrInvalidQuery`, so it gives 400 `invalid_request`. At most 20 words are used, so a pathological query can't build a huge `WHERE`. A longer list is cut, which can't happen within 200 characters of real words anyway.

### Paging
The cursor encodes the sort key and the ID and doesn't depend on the filters, so it stays valid with `q` and `feedId`. The frontend includes `q` and `feedId` in the query key, so changing them starts a new list instead of mixing pages.

### Frontend
- `FeedsPage` keeps `search` (the input value), a debounced `query` (300 ms) and `feedId` in local state next to `view`. `feedItemsQueryOptions(view, { q, feedId })` puts them in the query key. `placeholderData: keepPreviousData` keeps the list visible while typing.
- The search field is `type="search"` with a visible label, and the clear button is a separate `<button>`. The feed filter is a native `<select>` with the subscription icons left out, so it works well on phones and with the keyboard.
- When the chosen subscription disappears from `GET /api/feeds`, the filter resets to "All feeds".
- "No matching items" replaces the empty-view texts whenever `query` or `feedId` is set.

## Risks / Trade-offs

- [The fold function runs on every row the view joins, for each search] → It runs over a few thousand short strings per request, which is fast. If it ever gets slow, a stored key can be added without changing the API.
- [`strings.ToLower` isn't full case folding ("ß" vs "SS")] → That's acceptable for search. `golang.org/x/text/cases.Fold` is already a dependency and can replace it if needed.
- [Users expect older items to be found] → The page says nothing about it, but the spec states it. If it's often confusing, a hint can be added later.
