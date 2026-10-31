# Tasks

## 1. Prerequisite

- [ ] 1.1 Check that `add-note-locations` is applied (`notes.BlockLocation` exists and its Markdown tests pass); verify with `go test ./internal/notes/...`

## 2. Notes helpers

- [ ] 2.1 Add `SplitMemoTail` / `JoinMemoTail` in `internal/notes` (body, optional trailing location, trailing attachment IDs) and reimplement `SplitTrailingAttachments` / `JoinTrailingAttachments` on top of them; verify with doc tests for "Location at the end", "Location in the middle", "Two locations at the end", a note that is only a location, and that existing trailing-attachment tests pass unchanged

## 3. Memos API

- [ ] 3.1 Add `MemoLocation` and `Memo.Location`, and fill them in `MemoOf`/`Content` from the trailing location (placeholder `""` without a label); verify with `memos_test.go` cases for "Location at the end", "Location without label" and "Titled note" (no `location` key)
- [ ] 3.2 Read `location` in `CreateInput` (null is none; validate ranges and number type, normalise the placeholder) and write it with `JoinMemoTail`; verify with httptest cases "Memo with a location" and "Out of range" (400, code `3`, no note)
- [ ] 3.3 Read `location` as a three-state field in `UpdateFromBody` (set, remove via `null` or mask without a value, not applied) and apply it in the update, keeping the location when only `content` changes; verify with httptest cases "Set a location", "Remove the location", "Edit text keeps the location", "Unchanged memo" (version unchanged) and an unmasked body with `location: null`

## 4. Verification

- [ ] 4.1 Run `make check` and `make test`; verify both pass
- [ ] 4.2 With MoeMemos (or `curl` against `/api/memos/api/v1/memos`), create a memo with a location, check the chip in Boardly's editor, move the location in the web editor to the middle of the note, and check that the memo then shows it as a `geo:` link and has no `location`
