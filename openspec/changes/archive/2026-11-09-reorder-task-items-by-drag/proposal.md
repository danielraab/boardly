# Proposal

## Why

Changing the order of a task list in a note today means cutting and pasting lines, which is awkward on a desktop and close to unusable on a phone. Boardly already lets people reorder other lists by dragging a grip, so task lists should work the same way.

## What Changes

- While a note's content can be edited, every task item shows a faint drag handle to the left of its checkbox. This applies to every editor: the note page, the dashboard note dialog and inline editing on cards.
- Dragging a handle with a mouse or by touch moves the item among the items of its own list level. A line shows where it will land. A sublist moves along with its item.
- Alt+↑ and Alt+↓ move the task item that holds the cursor, as the keyboard way to do the same.
- A move is one undo step and is saved by the normal autosave. The item keeps its ID and checked state.
- Moving an item to another nesting level stays with Tab and Shift+Tab. Bullet and numbered lists, and the read-only checkboxes on cards, are not part of this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `note-document`: task items in the editor can be reordered with a drag handle and with the keyboard.

## Impact

- Frontend only: a new editor plugin and move helper under `frontend/src/features/notes/editor/`, registered in `NoteEditor`, and a new `notes.editor.moveTask` string in `en.json` and `de.json`. It uses the existing `@dnd-kit/core`; no new dependency.
- Export and import are unaffected: the document format doesn't change, only the order of items in it.
- The OpenAPI documents are unaffected: no route or request or response shape changes.
