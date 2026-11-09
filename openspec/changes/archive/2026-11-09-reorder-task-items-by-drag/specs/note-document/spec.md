# Spec Delta

## ADDED Requirements

### Requirement: Reordering task items in the editor
While a note's content can be edited, in every editor of a note's content (the note page, the dashboard note dialog and inline editing on cards), each task item SHALL show a drag handle to its left, always visible, in `muted`, with the label "Move task" / "Aufgabe verschieben". Bullet and numbered list items SHALL NOT have a handle. Content that can't be edited, such as a note card that isn't being edited or an editor locked after a version conflict, SHALL NOT show handles.

Dragging a handle with a mouse or by touch SHALL move the item, while a line shows where it would land. An item SHALL only be movable among the items of its own list at its own nesting level, and SHALL take its sublist with it. No drop position SHALL exist between an item and its sublist, nor above a sublist that has no item of its own at the start of a list. A short tap or click on a handle SHALL change nothing and SHALL NOT toggle the checkbox. Pressing Escape during a drag SHALL cancel it.

Alt+↑ and Alt+↓ SHALL move the task item that holds the cursor one place up or down among the items of its own list, with its sublist, and the cursor SHALL stay in the item. At the start or end of the list the keys SHALL do nothing. Operating a handle SHALL NOT move the cursor or the focus out of the editor.

A move SHALL count as a change that is saved automatically and SHALL be one undo step. The item SHALL keep its ID and its checked state. Ending a drag at the item's current position SHALL NOT count as a change.

#### Scenario: Drag a task to another place
- **WHEN** Ann drags the handle of "eggs" in the task list "milk, bread, eggs" above "bread" and releases it
- **THEN** the list reads "milk, eggs, bread", and after the autosave the note's task items are in that order, with "eggs" under its previous ID and in its previous checked state

#### Scenario: Drag by touch
- **WHEN** Ann drags the handle of "eggs" with a finger on a phone and lifts it above "bread"
- **THEN** "eggs" is above "bread", and the page hasn't scrolled because of the drag gesture on the handle

#### Scenario: Sublist moves along
- **WHEN** "bread" has the sublist "rye", and Ann drags "bread" below "eggs"
- **THEN** "rye" is still the sublist of "bread", below "eggs"

#### Scenario: Own level only
- **WHEN** Ann drags the top-level item "milk" over the sublist item "rye" and releases it
- **THEN** "milk" lands in a position of the top-level list, and "rye" stays in its sublist

#### Scenario: Sublist item stays in its sublist
- **WHEN** Ann drags the sublist item "rye" far above its parent list
- **THEN** "rye" becomes the first item of its sublist and doesn't leave it

#### Scenario: Keyboard
- **WHEN** the cursor is in "eggs" in the list "milk, bread, eggs", and Ann presses Alt+↑
- **THEN** the list reads "milk, eggs, bread" and the cursor is still in "eggs"

#### Scenario: Keyboard at the end of the list
- **WHEN** the cursor is in the first task item of a list, and Ann presses Alt+↑
- **THEN** nothing changes

#### Scenario: Undo a move
- **WHEN** Ann has moved "eggs" above "bread" and presses Ctrl+Z
- **THEN** "eggs" is below "bread" again

#### Scenario: Drag cancelled
- **WHEN** Ann presses Escape while dragging "eggs"
- **THEN** the order is unchanged and no save happens

#### Scenario: Dropped in place
- **WHEN** Ann picks up "bread" and releases it at its current position
- **THEN** the order is unchanged and no save happens

#### Scenario: Tap on a handle
- **WHEN** Ann taps the handle of the unchecked task "milk" without dragging
- **THEN** "milk" stays unchecked and in its place

#### Scenario: Inline editing continues
- **WHEN** Ann edits a note inline on its card and drags a task item by its handle
- **THEN** the item moves and the card stays in inline editing

#### Scenario: No handles without editing
- **WHEN** a note card with a task list isn't being edited, or the editor is locked after a version conflict
- **THEN** no drag handles are shown

#### Scenario: Other lists
- **WHEN** a note has a bullet list and a numbered list
- **THEN** their items have no drag handle, and Alt+↑ and Alt+↓ don't move them
