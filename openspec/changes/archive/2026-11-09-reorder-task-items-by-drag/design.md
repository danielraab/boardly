# Design

## Context

See `proposal.md` for why, and the `note-document` delta for the required behaviour.

The note editor is one Lexical component (`editor/NoteEditor.tsx`) used by the note page, the dashboard dialog and inline card editing. Task items are `ListItemNode`s in a `check` list. Their checkbox and tick are the item's `::before` and `::after`.

Lexical stores a sublist as a separate wrapper item directly after its parent item:

```
ul(check)
  li  "bread"            <- item
  li  > ul > li "rye"    <- wrapper, belongs to the item before it
  li  "eggs"
```

`listBlock` in `document/lexical.ts` turns a wrapper into the `children` of the item before it, unless that item already has children; then, or when there is no item before it, the wrapper becomes an item of its own with empty content (an orphan). A first item that is indented produces such an orphan.

Other sortable lists (`ChosenNotes`, `ChosenSources`) use dnd-kit with a `GripVertical` handle.

## Goals / Non-Goals

**Goals:**
- One implementation that works in every editor instance.
- A move never changes which item owns which sublist.

**Non-Goals:**
- Moving items between lists or between nesting levels.
- A focusable handle for each item.
- Animating items out of the way during a drag.

## Decisions

### Handles are an overlay next to the content, not part of it

A new plugin renders one handle per task item, absolutely positioned inside the existing `relative` wrapper around the content, as a sibling of the contenteditable. It computes positions from `editor.getElementByKey(key).getBoundingClientRect()` relative to that wrapper. It recalculates on every editor update and from a `ResizeObserver` on the root element, which covers rewrapping and images finishing loading. It renders nothing while the editor isn't editable, following `registerEditableListener`.

Alternatives: an inline decorator node in each item would land in the document, selection and `$lexicalToDoc`. CSS has no pseudo-element left on the item. Lexical's `DraggableBlockPlugin` handles top-level blocks with mouse events only.

The handle sits in the gutter to the left of the item. The theme gives a task list both `pl-6` (from `ul`) and `pl-0` (from `checklist`), and nested lists need that indentation too. `pl-6` wins by CSS order, confirmed in a browser, which leaves a 24 px gutter at every level, so no layout change is needed.

### Drags use dnd-kit's sensors and auto-scroll, with our own drop calculation

A `DndContext` with the same pointer and touch sensors as `ChosenNotes` and `useDraggable` on each handle provides activation, Escape to cancel and scrolling near the edge. The handle has `touch-none`.

Droppables aren't used. On each move, the plugin finds the drop gap from the pointer's `clientY` and the current DOM rects of the dragged item's sibling blocks. It tracks `clientY` itself with a `pointermove` listener during the drag, because dnd-kit's `delta` is adjusted for scrolling and so isn't a viewport position. Limiting the gaps to the item's own siblings is what enforces "own level". A drop line and a faint highlight on the dragged block are drawn in the overlay; the Lexical-owned DOM stays untouched until the drop.

Alternative: `useSortable` with droppables on the rows. It would need the rows' transforms ignored and a collision filter for levels, and gives nothing the gap calculation doesn't.

### Escape cancels only the drag

During a drag, a capturing `keydown` listener on `window` stops Escape from reaching anything else, since inline card editing would end and the dashboard dialog would close. dnd-kit's own Escape handling is on `document` and is stopped as well, so the listener ends the gesture by dispatching `pointercancel` and `touchcancel` on `document`, which the sensors treat as a cancelled pointer.

### A move works on blocks

A helper module holds the node logic, independent of React:

- A block is a task item plus every wrapper directly after it. Including all wrappers, not only the first, keeps an orphan wrapper from being adopted by a different item when the block in front of it moves away.
- Gaps exist only at block boundaries, so a drop between an item and its sublist can't be expressed.
- A wrapper at the very start of a list is a block with no handle, and the gap before it isn't offered, since an item placed there would adopt it.
- The move relocates the existing nodes in a single `editor.update`, so task IDs, checked state and the cursor stay with them, `TaskIdPlugin` has nothing to do, and history records one step. A gap equal to the block's current position is a no-op without an update.
- On drop, the update looks the nodes up by key again and does nothing if one is gone or the editor is no longer editable.

### Keyboard is Alt+↑/↓ on the current item, and handles aren't focusable

Inside a list Tab already means indent, and a tab stop per item would make getting past a long list tedious. The handle has `tabIndex={-1}`. A command handler on `KEY_ARROW_UP_COMMAND` and `KEY_ARROW_DOWN_COMMAND` acts when Alt is held and the selection lies in one task item. It is registered above the priority of `CheckListPlugin`'s arrow handling, and calls `preventDefault`, also when the item is already at the boundary.

### Pressing a handle doesn't take focus

The handle calls `preventDefault` on `mousedown`, so the caret stays and the phone keyboard doesn't change. The handles live inside the editor's DOM, so inline card editing, which ends on a blur leaving the card, isn't affected.

## Risks / Trade-offs

- [Handles lag behind layout changes the editor doesn't report, such as a font finishing loading] → The `ResizeObserver` catches anything that changes the root's size, and the next update corrects the rest.
- [Recalculating positions on every update in a long note] → Only task items are measured, and state is only set when a value changed.
- [Lexical's `ListItemNode` has special handling on removal that merges neighbouring sublists] → Blocks keep wrappers with their items, so two wrappers shouldn't become neighbours through a move. Tests cover nested lists, and the document is compared after each move.
- [24 px beside the checkbox is a small touch target] → A tap on the handle does nothing, so a miss can't cause a wrong action. Larger targets stay out of scope, as for the toolbar in `improve-mobile-note-editor`.
- [Alt+↑/↓ replaces Option+↑/↓ caret movement on macOS inside task items] → Limited to task items; elsewhere the keys keep their behaviour.
