# Tasks

## 1. Move logic

- [x] 1.1 Confirm in a browser which of `pl-6` and `pl-0` applies to a task list in the editor; if `pl-0` wins, remove it from the `checklist` theme entry in `editor/theme.ts`, and verify that top-level and nested task items have a 24 px gutter on their left
- [x] 1.2 Add a helper in `editor/` that derives the blocks and allowed gaps of a task item's list and moves a block to a gap by relocating the existing nodes; verify with unit tests through `$lexicalToDoc` that a sublist moves with its item, no gap lies between an item and its sublist, a wrapper at the start of a list keeps its place and has no gap before it, an orphan wrapper isn't adopted by another item, and IDs and checked states are kept
- [x] 1.3 Add the Alt+↑ and Alt+↓ handling to the plugin; verify with tests in `NoteEditor.test.tsx` that the item moves, the cursor stays in it, the list boundaries do nothing, undo restores the order in one step, the autosave `onChange` fires, and bullet and numbered lists are ignored

## 2. Handles and dragging

- [x] 2.1 Add the handle overlay plugin and register it in `NoteEditor`, with the `notes.editor.moveTask` string ("Move task" / "Aufgabe verschieben") in `en.json` and `de.json`; verify with tests that each task item gets one handle, bullet and numbered items get none, and no handles show when the editor isn't editable
- [x] 2.2 Add dragging with dnd-kit: the drop gap from the pointer position, the drop line, the source highlight, and the drop calling the helper from 1.2; verify with tests of the gap calculation against given rects, including the first and last gap and that a drop at the current position is a no-op
- [x] 2.3 Verify in a browser with a mouse: dragging in a flat and a nested list, Escape cancels, auto-scroll in a long note, a click on a handle neither toggles nor moves the caret, and a note reload shows the new order
- [x] 2.4 Verify in a browser at 360 px with touch emulation, and on a phone if available: dragging a handle moves the item without scrolling the page, and a tap does nothing
- [x] 2.5 Verify in a browser that dragging works in the dashboard note dialog and in inline card editing, and that inline editing doesn't end when a handle is used

## 3. Docs and checks

- [x] 3.1 Add a `CHANGELOG.md` entry, confirm that export, import and the OpenAPI documents need no change, and verify with `git diff --name-only` that only `frontend/`, `CHANGELOG.md` and OpenSpec paths changed
- [x] 3.2 Run `cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test` and verify it passes
- [x] 3.3 Run `openspec validate reorder-task-items-by-drag --strict` and verify it passes
