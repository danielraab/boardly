# Tasks

## 1. Saving and editor building blocks

- [x] 1.1 Add `saveNow(fields)` to `useAutosave` (record fields and flush without the delay); verify with a unit test that two quick `saveNow` calls send two PATCHes in order, the second with the version returned by the first
- [x] 1.2 Let `useAutosave` adopt a newer `note` prop version while nothing is pending or running; verify with a unit test that a save after an external version bump sends the new version
- [x] 1.3 Add `toolbar` (default `true`) and `autoFocus` props to `NoteEditor`; verify with a test that the inline editor renders no toolbar while Ctrl/Cmd+B still applies bold and the cursor starts at the end, and that the editor page tests still pass
- [x] 1.4 Move `SaveIndicator` out of `NoteEditorPage.tsx` into its own module and use it from there; verify the existing editor page tests still pass

## 2. Card checkboxes

- [x] 2.1 Add an optional `onToggleTask(id, checked)` to `NoteRenderer`; with it, task checkboxes are enabled, focusable and stop click propagation, without it they stay inert; verify with a `NoteRenderer` test for both cases
- [x] 2.2 Give `NoteCard` its saver (`useAutosave`) and a local draft; toggles update the draft and call `saveNow({content})`; on failure reset the draft, invalidate `notes` and report via a callback to `NotesPage`'s toast (changed elsewhere vs. couldn't save); verify with tests for a successful toggle (PATCH body has `checked: true`, still on `/notes`) and a failed one (checkbox reverts, toast shown)

## 3. Inline editing

- [x] 3.1 Replace the covering `<Link>` with an "Open note" icon link and a covering edit `<button>` labelled "Edit <title>"; route the card's category badge through the card's saver; add `notes.openNote`, `notes.edit` and keep `notes.open` for the link's label, in `en.json` and `de.json`; verify with tests that "Open note" navigates to `/notes/<id>` and the i18n key-parity test passes
- [x] 3.2 Add `InlineNoteEditor` (title input with placeholder, `NoteEditor` without toolbar, `SaveIndicator`) rendered by the card in edit mode, with the card growing to its full content; focus the title when the title was activated, else the end of the content; verify with a test that clicking the content shows the editor without a toolbar and typing "eggs" is saved after the autosave delay
- [x] 3.3 Hold the editing note ID in `NotesPage`; end edit mode on Escape, on activating another card, and on focus leaving the card, flushing pending changes with `saveNow`; verify with tests that Escape shows the card with the new text and that only one card is in edit mode at a time
- [x] 3.4 On `version_conflict`, end edit mode, reset the card's saver, invalidate `notes` and show the `notes.changedElsewhere` toast; verify with a test using a 409 response

## 4. Grid order

- [x] 4.1 Keep the order of note IDs in `NotesPage` from the first load; sort later data by it, with unknown IDs first in server order; verify with a test that a card edited inline keeps its position after the list refetches, and a remount uses the server order again

## 5. Verification

- [x] 5.1 Update the existing `Notes.test.tsx` tests that relied on the card opening the note, then run `make check` and `make test` and verify both pass
- [x] 5.2 Run the app and check by hand: tick a todo on a card, edit a card's title and content inline, Escape, "Open note", at desktop and at 375 px width
