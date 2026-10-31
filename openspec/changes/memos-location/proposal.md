# Proposal

## Why

Memos clients such as MoeMemos can attach a location to a memo, but Boardly's Memos API ignores
the `location` field, so a place set on the phone is lost. Now that notes have location blocks
(change `add-note-locations`), a memo's location can be kept as one.

## What Changes

- A memo's **`location`** (`{"placeholder", "latitude", "longitude"}`) SHALL map to the note's
  **trailing location**: a location block at the end of the content, right before the trailing
  attachments. Its label is the placeholder. This follows the existing "trailing attachments are
  the memo's attachments" rule.
- Listing and getting memos returns `location` when the note has a trailing location, and the
  memo's `content` leaves that block out. Other location blocks stay in `content` as `geo:` links.
- Creating a memo with `location` appends a location block after the content and before the
  attachments.
- Updating a memo with `location` replaces the trailing location, and `location` set to `null`
  (or listed in `updateMask` without a value) removes it. Updating only `content` keeps it.
- Invalid coordinates respond 400 with code `3`.
- Depends on `add-note-locations` (the location block and its `geo:` Markdown), which must be
  applied first.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `memos-api`: the memo shape gains `location`; create and update apply it instead of ignoring
  it; a new requirement defines the trailing location.

## Impact

- Backend: `internal/notes` gets split/join helpers for the trailing location next to
  `SplitTrailingAttachments`/`JoinTrailingAttachments`; `internal/memos` (`Memo`, `MemoOf`,
  `Content`, `CreateInput`, `UpdateFromBody`, `Update`) reads and writes `location`; tests in
  `memos_test.go` and the httptest suite.
- No frontend, configuration or migration changes.
