# Design

## Context

See `proposal.md` for the motivation and `specs/` for the behaviour. The `feed` kind resolves its items from the database through `feeds.Store.Items`, with the item count as the query's limit.

## Goals / Non-Goals

**Goals:**

- Filter per item, so that feeds mixing articles and episodes show only their episodes.

**Non-Goals:**

- The same filter on the `/feeds` page.
- Filtering by audio type or length.

## Decisions

### Filter in the query, not after it

`ItemQuery` gets an `AudioOnly` field that adds `i.audio_url IS NOT NULL` to the query. Filtering the returned rows instead would apply the limit first, leaving a widget with few or no rows when the newest items are articles.

### A plain optional config field

`audioOnly` is a boolean that decodes to false when missing, so stored widgets need no migration. The edit-mode preview filters its 100 preview items on the client, like it already does for the chosen feeds.

## Risks / Trade-offs

- [The edit-mode preview only sees the newest 100 items of a view] → It can show fewer rows than the saved widget when audio items are old. It is only a preview, and the saved widget is queried with the filter.
