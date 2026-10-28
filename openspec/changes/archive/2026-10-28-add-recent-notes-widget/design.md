# Design

## Context

Content widget kinds live in `backend/internal/widgets` as a `dashboards.Kind` with `Normalize` (validates the config when a layout is saved) and `Resolve` (loads data per viewer on lookup). `recent.go` (recent favourites) already does count limit, optional `days` window and a strict config; `note.go` already resolves a note's category with `categoryOf(categories, id, withPath)` and builds separate owner and visitor structs. `widgets.Register` already gets the notes store. `notes.created_at` / `updated_at` are fixed-width RFC 3339 UTC text, so string comparison in SQL is chronological, and there is an index `notes_user_updated (user_id, updated_at DESC, id)`.

On the frontend, the display title (`displayTitle` / `firstLine` in `features/notes/document/text.ts`) is only computed in the browser. Recent favourites has `relativeTime` and `fullDate` in `widgets/recentFavourites/recent.ts`. The note widget opens `NoteEditorPage` in a dialog (`widgets/note/NoteDialog.tsx`) and invalidates the dashboard view after saves and on deletion. `WidgetKind` has a `group`; the "notes" group currently holds only `note`.

## Goals / Non-Goals

**Goals:**
- One SQL query per widget that returns only the notes to show, sorted and limited.
- Visitors never receive note content, IDs or anything not shown.
- Reuse the note widget's editor modal and the recent favourites' time helpers and row styling.

**Non-Goals:**
- A category scope (all notes are always considered).
- Showing the note's colour, content previews or toggling tasks from the list.
- Live updating of relative times or of notes changed in another tab.

## Decisions

**Config shape.** `{sortBy, limit, days, title, size, showIfPublic}` with `sortBy` `"created" | "updated"`, decoded strictly like the other kinds. `limit` is required (1–100) and `days` is `*int` (nil or 1–365), the same bounds and error messages as recent favourites; share the constants rather than copying them. `sortBy` is required on save; the frontend default is `created`. No ID in the config, so there is nothing that can go missing and no "missing" state.

**Store query.** Add `notes.Store.RecentNotes(ctx, userID string, byUpdated bool, since time.Time, limit int) ([]Note, error)`: `SELECT noteColumns FROM notes WHERE user_id = ? [AND <col> >= ?] ORDER BY <col> DESC, id DESC LIMIT ?`, where `<col>` is picked from two fixed column names (never from input). The `updated` order uses the existing index; `created` sorts the user's notes, which are few. The ID tie-break matches recent favourites. Alternative: `Store.Notes` and sort in Go — simpler, but loads every note's content for a top 10.

**Rolling window on the server.** `since = now − days × 24h` at resolve time, `now` from a function the kind is given, like recent favourites.

**Display title in Go.** Port `firstLine` and `displayTitle` to `notes` (`DisplayTitle(n Note) *string`): title, else the first non-empty trimmed line of text of the content walking blocks in reading order (paragraph/heading/quote inline text split on line breaks, code text, list items and their children, dividers skipped), cut after 80 characters with "…", else nil (shown as "Untitled note"). Tests run the same cases as `text.test.ts` so both stay equal. Alternative: send the content and compute in the browser — rejected because visitors would get the whole note.

**Response data.** Both viewers get `{notes: [{title, icon, time, category}]}` where `title` is the display title (nullable), `time` is the chosen timestamp in RFC 3339 and `category` is `{title}` or null. The owner's rows additionally have `id` and `category.path` (via `categoryOf(..., true)`). Visitor and owner rows are separate structs, as in `note.go`. Categories are loaded once per widget with `Store.Categories`.

**Edit-mode preview from the notes query.** In edit mode a new or changed widget has no data, so the view computes the rows in the browser from `notesQueryOptions` and `noteCategoriesQueryOptions` (only the owner edits): same sort, window, tie-break and limit, with the existing `displayTitle`. A small pure function `recentNotes(notes, sortBy, limit, days, now)` keeps this testable, like `recentLinks`.

**Opening a note.** Rows are buttons for the owner outside edit mode and plain elements otherwise. Activating one opens `NoteDialog` for that ID. Its open state lives in the view, keyed by note ID, so a refetch of the dashboard while it is open doesn't close it. After a save or deletion the dialog's existing callbacks invalidate the dashboard view, which reorders the list for "Last changed" and drops deleted notes. If `NoteDialog` is tied to the note widget's props, generalise it to take a note ID and `onChanged` rather than copying it.

**Frontend reuse.** Move `relativeTime` and `fullDate` out of `recentFavourites/recent.ts` into a shared module in `widgets/` (or import them from there if moving adds churn). Rows use the same S/M/L `sizeStyles` and the title heading of the other list widgets. Use `ChosenIcon` for the note's icon; notes without an icon get no placeholder icon.

## Risks / Trade-offs

- [Visitors see note titles and first lines once the flag is on] → Off by default; the settings dialog says next to the flag that visitors see the titles of the newest notes.
- [The Go display title can drift from the TypeScript one] → Shared test cases in both test suites; the visitor's text is only shown, never edited.
- [With "Last changed", ticking a todo in another widget or on `/notes` reorders the list] → Intended; it's what "last changed" means. The list updates on the next dashboard fetch.
- [Relative times go stale on a page left open] → Same as recent favourites; no timer.
