# Design

## Context

This builds on the `add-recent-notes-widget` and `add-note-list-widget` changes, whose code is already on this branch. `widgets/notelist.go` has the visitor and owner row structs without a time (`listedNote`, `ownerListedNote`), `widgets/note.go` has `categoryOf` and resolves a missing note per kind (`missingNote` for the owner, left out for visitors), and `widgets/recentnotes.go` has `maxRecentLimit` (100). `notes.Store` has `Note` (one by ID) and `Notes` (all of a user), but nothing for a list of IDs.

On the frontend, `widgets/NoteRows.tsx` renders the rows (icon, display title, optional second line, optional time), `note/NoteDialog.tsx` is the editor modal, `notes/categories/tree.ts` has `categoryPath`, and `noteList/list.ts` builds the edit-mode preview from the notes queries. `favourites/TagInput.tsx` uses the Base UI `Combobox` for a searchable choice. `@dnd-kit/core` and `@dnd-kit/sortable` are already used by the layout editor and the favourites page.

## Goals / Non-Goals

**Goals:**
- Reuse the note list's rows, row structs, modal handling and server-side display title rather than copying them.
- The same rows and order on the server and in the edit-mode preview.
- Reordering that works by pointer, touch and keyboard.

**Non-Goals:**
- Automatic sorts, a count limit separate from the list, or filtering.
- Removing a deleted note's ID from saved layouts when the note is deleted.
- Placeholders for deleted notes in the widget itself.

## Decisions

**Config shape.** `{noteIds, title, size, showIfPublic}`, decoded strictly. `noteIds` is a non-empty array of at most `maxRecentLimit` strings, each non-empty, at most `maxIDLength` and unique. Duplicates are rejected rather than removed, so the server never changes the order the owner saw. Alternative: an array of objects (`[{noteId}]`), which would leave room for per-note options. Nothing needs those yet, and plain IDs match `noteId` in the note widget.

**Store query.** Add `notes.Store.NotesByID(ctx, userID string, ids []string) ([]Note, error)` with `WHERE user_id = ? AND id IN (?, …)`. It returns the found notes in any order. The widget puts them in config order via a map and skips IDs that weren't found. At most 100 placeholders is far below SQLite's limit. Alternative: call `Store.Note` per ID, which is up to 100 queries per widget and lookup.

**Resolve.** Flag off for a visitor → left out. Load the notes; if none is found, the owner gets `{"missing": true}` (the same `missingNote` shape the note widget uses) and visitors get the widget left out. Otherwise load `Store.Categories` once, only when a found note has a category, and build `listedNote` rows with `notes.DisplayTitle` and `categoryOf` for every note with a category (path for the owner only). The owner gets `ownerNoteListData` with IDs, and visitors `noteListData`. The response shape is the same as the note list's, so the frontend type and `NoteRows` can be shared.

**Second line.** Unlike the note list, there is no chosen category, so every note with a category gets it, the same rule as the note list's "All notes".

**IDs in the config.** The dashboard lookup sends every widget's config to visitors, so a public widget carries its note IDs, as the note widget carries `noteId` and the note list its `categoryId`. The rows never contain IDs. Stripping configs for visitors would be a dashboard-wide change and is out of scope.

**Edit-mode preview.** A pure `selectedNotes(notes, categories, config)` in `widgets/selectedNotes/selected.ts` maps the IDs to the loaded notes in order, skips missing ones, adds the second line with `categoryPath`, and returns `{missing: true}` when nothing is found. It is tested with the same cases as the Go side, like `noteList` / `recentNotes`.

**Adding notes.** A Base UI `Combobox` as in `TagInput`, single choice, that clears its input after each choice. The items are the owner's notes from the notes query, minus those already in the list, sorted with `Intl.Collator(undefined, {sensitivity: "base"})` on the display title, untitled last. The filter matches the display title and the joined category path, case- and accent-insensitive (`localeCompare`-style normalization via `normalize("NFD")` and removing combining marks). It is disabled at 100 notes. Alternative: a plain text input with a filtered list below. The combobox already gives keyboard navigation and ARIA roles.

**Ordering.** A `DndContext` + `SortableContext` (vertical list strategy) inside the settings dialog, with `PointerSensor`, `TouchSensor` and `KeyboardSensor` with `sortableKeyboardCoordinates`. Only the handle has the drag listeners, so the buttons inside a row stay clickable. "Move up"/"Move down" buttons call the same `arrayMove`. This is a separate context from the layout editor's: the dialog renders in a portal and dnd-kit scopes draggables to the nearest `DndContext`. Alternative: buttons only. They are enough for accessibility, but dragging is what people expect for a list of up to 100 items.

**Missing notes in the dialog.** The dialog keeps the config's IDs as its state. Any ID not in the notes query is rendered as a warning row "This note no longer exists" with only the remove button and the handle, so the owner sees where it was and can remove it.

**Frontend reuse.** The widget renders the note list's `NoteListView` (title heading, `NoteRows` without times), which gets an optional `missingText` for "None of the selected notes exist anymore". Its empty state can't occur, since data without notes is missing. The `NoteDialog` handling for the owner outside edit mode is the same as in the note list's `View`.

## Risks / Trade-offs

- [Saved layouts keep IDs of deleted notes] → They're skipped, and the settings dialog shows them for removal. Same as the note widget's missing note.
- [Two `DndContext`s on the page while editing] → The dialog's context is separate and portaled; tested by dragging inside the dialog while the layout editor is open.
- [Server collation and `Intl.Collator` don't matter for order here] → The order is the list's, so the preview and the server always agree. Only the search results use the collator.
- [All three note widget changes modify "Add widgets" in `dashboard-editing`] → This delta already includes "recent notes" and "note list"; archive `add-recent-notes-widget`, then `add-note-list-widget`, then this one.
- [Visitors see note titles once the flag is on] → Off by default, with a hint next to the flag, as in the other note widgets.
