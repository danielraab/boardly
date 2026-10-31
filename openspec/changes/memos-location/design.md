# Design

## Context

- `add-note-locations` adds `notes.BlockLocation` (`Lat`, `Lng`, `Label`) and its `geo:` Markdown.
  This change builds on it and can't be applied before it.
- `internal/memos` maps a note to a memo with `notes.SplitTrailingAttachments` (the memo's
  `content` is the body, `attachments` the split-off IDs). On write, it calls
  `notes.JoinTrailingAttachments(FromMarkdown(content), ids)`.
- `UpdateFromBody` reads masked fields with a `use` helper that treats `null` as absent. That is
  right for every current field, but `location: null` has to mean "remove".

## Goals / Non-Goals

**Goals:**
- One rule for what a memo's location is, and a round trip through MoeMemos that keeps the note's
  block structure stable.

**Non-Goals:**
- Mapping several locations to Memos (Memos has one per memo).
- Reverse geocoding an empty placeholder into a place name.

## Decisions

### 1. Note layout: body, trailing location, trailing attachments

```
┌──────────────── note content ─────────────────┐
│ body blocks …   │ [location]?  │ attachment*  │
└──── memo.content ┴ memo.location ┴ memo.attachments
```

A generalised split replaces the separate attachment split:
`notes.SplitMemoTail(d) (body Doc, loc *Block, ids []string)` and
`notes.JoinMemoTail(body, loc, ids) Doc`. `SplitTrailingAttachments` and
`JoinTrailingAttachments` are kept, but implemented on top of these, so existing callers (Karakeep)
don't change behaviour. The order "location before attachments" was chosen because MoeMemos shows
the location under the text and the attachments last, and because the web editor's "Add
attachment" puts files at the end, after any location the user added last.

Alternative: the *first* location block anywhere in the note. Rejected: it would have to be kept
in `content` too (or removed from the middle of the text), and editing it from MoeMemos would
move it.

### 2. Tri-state `location` in updates

`UpdateInput.Location` becomes a field with three states: not applied, applied with a value, and
applied as removal (`*MemoLocation` plus a `LocationSet bool`, or a small `optional[T]` type).
It's read directly from the body, not through `use`: the key is present (including `null`), or it
is listed in `updateMask`. Validation (ranges, number type) happens while reading the body, so a
bad location fails before anything is loaded or written.

### 3. Equality for "unchanged"

The existing check compares content Markdown. In addition, the new trailing location is compared
field by field after rounding and label normalisation, so resending the same location doesn't
bump the version (scenario "Unchanged memo").

### 4. Wire type

```go
type MemoLocation struct {
	Placeholder string  `json:"placeholder"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}
```

`Memo.Location *MemoLocation` with `json:",omitempty"`. The input is decoded from
`json.RawMessage`. A missing coordinate is `0`: MoeMemos serialises with kotlinx `encodeDefaults = false` and
`MemoLocation` defaults to `0.0`, so a point on the equator or the prime meridian arrives without that
field (protobuf JSON, which Memos itself uses, does the same). Only a coordinate that isn't a number,
or is out of range, is rejected.

## Risks / Trade-offs

- [A client that always sends `location: null` in an unmasked PATCH removes the trailing location]
  → that matches what the client last read, and MoeMemos sends an `updateMask`. Covered by a test.
- [Content written from Memos that ends with a `geo:` paragraph, plus a `location` field, gives two
  locations] → that's consistent with how trailing attachments work, and the earlier one shows as a
  `geo:` link in the next read.
