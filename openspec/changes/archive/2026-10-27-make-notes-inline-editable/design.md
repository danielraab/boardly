# Design

## Context

`NoteCard` renders a note read-only with `NoteRenderer` and covers the whole card with a `<Link>` to `/notes/<id>`; only the category badge sits above it. Checkboxes and links in `NoteRenderer` are inert. The editor page combines `NoteEditor` (Lexical, with `Toolbar`) and `useAutosave`, which serialises PATCHes, tracks the note's version in a ref, and stops on `version_conflict`. `NotesPage` sorts nothing itself: the server returns notes by time of last change, and every save invalidates the `notes` query.

## Goals / Non-Goals

**Goals:**
- Edit title and content of one card at a time in place, with the same document format and autosave rules as the editor page.
- Toggle task items on a card without entering edit mode.

**Non-Goals:**
- Icon, colour, delete or toolbar on the card; those stay on the editor page.
- Creating notes inline; "New note" still opens `/notes/new`.
- Adding task items through a dedicated input.

## Decisions

- **The card has two modes, view and edit, with the editing card's ID held in `NotesPage`.** Holding it in the page makes "only one card at a time" trivial and lets activating another card end the current one. Alternative: state per card with a blur handler only — two cards could briefly be in edit mode and race their saves.
- **One saver per card: `NoteCard` always runs `useAutosave` for its note**, for both checkbox toggles and inline editing, so all saves of a note are serialised and share one version. `useAutosave` gets a `saveNow(fields)` that records the fields and flushes without the delay, used for toggles and when edit mode ends. Alternative: a `useMutation` per toggle with `note.version` from the list — the second of two quick toggles would carry a stale version and get 409.
- **Edit mode renders an `InlineNoteEditor`** with a title `<input>`, `NoteEditor` with a new `toolbar={false}` prop, and the shared `SaveIndicator` (moved out of `NoteEditorPage.tsx`). It reports changes to the card's saver. Lexical is only mounted while a card is in edit mode: with dozens of notes, one Lexical instance per card would be slow.
- **The card shows a local draft** (title and content) on top of the note from the list: changes from edit mode or toggles update it at once, and it is reset from the list whenever the saver has nothing pending or running. On a failed toggle the card resets the draft to the list's note, invalidates the `notes` query and shows a toast.
- **`NoteEditor` gets an `autoFocus` prop** (Lexical's `AutoFocusPlugin` with `defaultSelection="rootEnd"`), used when the content was activated; when the title was activated, the title input gets `autoFocus` instead.
- **Ending edit mode**: Escape (keydown on the card), and `focusout` on the card whose `relatedTarget` is outside it. The inline editor has no toolbar or popups, so focus never legitimately moves into a portal.
- **Activation without the covering link**: the covering `<Link>` becomes a covering `<button>` labelled "Edit <title>", so click, Enter and Space work natively and nothing interactive is nested in it. The content sits above it with `pointer-events: none`, except for the checkboxes, so clicks on the text still reach the button. Whether the title was activated is told from the click position against the title's box; keyboard activation (`detail === 0`) edits the content. An "Open note" icon link (`Maximize2`) in the top-right corner, always visible, opens the editor page. Alternative: a `role="button"` element around title and content — it would contain the checkboxes, which a button must not.
- **The card's category badge saves through the card's saver too** (`saveNow({categoryId})`) instead of a separate mutation in `NotesPage`, so there is only one writer per note on the page.
- **`useAutosave` adopts a newer note version when it is idle.** When the `note` prop's version is higher than the saved one and nothing is pending or running, the hook takes the prop, e.g. after the list refetched a change made in another tab. Small change, keeps the editor page's behaviour.
- **Stable grid order while `/notes` is open**: `NotesPage` keeps an ordered list of note IDs in state, set from the server order the first time notes arrive. Later data is sorted by that list; IDs not in it (notes created elsewhere) go first in server order and are added to it. Remounting the page (navigating back to `/notes`) starts from the server order again. Alternative: pause refetching while editing — doesn't cover checkbox toggles and leaves stale data around.
- **Conflict**: on status `conflict` the card calls `onConflict`; `NotesPage` ends edit mode, invalidates the `notes` query and shows the existing `notes.changedElsewhere` toast. Unlike the editor page there is no read-only state with "Reload": the card's saver is reset (remounted by keying it on the conflict) and the card shows the current note, which is what "Reload" would do.

## Risks / Trade-offs

- [Clicking a link on a card used to open the note; now it starts editing] → Matches how text is edited elsewhere; Ctrl/Cmd+click inside the inline editor opens the link.
- [Typed text is lost on a conflict] → Same as the editor page's "Reload"; conflicts need two tabs editing the same note.
- [Editing card grows and shifts the cards below it in the grid] → Expected with a CSS grid; only one card grows at a time, and it shrinks back when editing ends.
- [`pagehide`/`beforeunload` listeners per card] → A few dozen listeners are negligible, and they make toggles that are still saving survive a tab close too.
