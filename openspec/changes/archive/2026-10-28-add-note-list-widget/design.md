# Design

## Context

This builds on the `add-recent-notes-widget` change, whose code is already on this branch. `widgets/recentnotes.go` has the strict config, separate visitor and owner row structs, `notes.DisplayTitle` and the `maxRecentLimit` bound. `widgets/note.go` has `categoryOf(categories, id, withPath)`. `notes/scope.go` already has a recursive "category and all its subcategories" SQL condition for app tokens. `explorer.go` and `linklist.go` return `{"missing": true}` to the owner when their folder is gone and leave the widget out for visitors.

On the frontend, `CategoryPicker` offers a radio tree of categories with one extra choice for `null`. `categories/tree.ts` has `CategoryFilter` (`"all" | "none" | {id}`), `filterNotes` (always includes subcategories) and `categoryPath`. `recentNotes/RecentNotesView.tsx` renders the rows with icon, title, category second line and time, and `note/NoteDialog.tsx` is the editor modal.

## Goals / Non-Goals

**Goals:**
- Reuse the recent notes rows, the modal and the server-side display title rather than copying them.
- The same selection and order on the server and in the edit-mode preview.

**Non-Goals:**
- Content previews, colours, times or ticking tasks in the rows.
- Choosing several categories, or filtering by anything other than category.
- An age limit ("only the last N days").

## Decisions

**Config shape.** `{category, categoryId, subcategories, sort, limit, title, size, showIfPublic}`, decoded strictly. `category` is `"all" | "none" | "category"`, and `categoryId` is required only for `"category"` (at most the ID length, otherwise cleared to null). `subcategories` is a bool and is only used for `"category"`. `sort` is `"updated" | "created" | "title"`, and `limit` is 1–`maxRecentLimit`. Alternative: one `category` string holding `"all"`, `"none"` or an ID. That's shorter, but it mixes markers with IDs.

**Store query.** Add `notes.Store.ListedNotes(ctx, userID string, sel Selection, sort string, limit int) ([]Note, error)`. The `Selection` is `{Mode all|none|category, CategoryID, Subcategories}`. The condition is `user_id = ?`, plus `category_id IS NULL` for none, `category_id = ?` for a category without subcategories, or the recursive `IN (WITH RECURSIVE …)` taken out of `scopeQuery` into a shared helper for a category with subcategories. For the time sorts, `ORDER BY <col> DESC, id DESC LIMIT ?` uses one of two fixed column names. The `title` sort needs the display title, which only exists in Go. It loads the selected notes without a limit, sorts them with a `collate.Collator` (the one link list uses, a new one per sort), untitled notes last and ties by ID, and then cuts to the limit. A user's notes per category are few, so that is fine. Alternative: always load and sort in Go. That's simpler, but it loads every note's content for a top 10 by time.

**Missing category.** Resolve loads `Store.Categories` once. For `"category"`, an ID that isn't among them gives the owner `{"missing": true}`, and visitors get the widget left out. The same list also gives each row's category via `categoryOf`.

**Second line.** A row gets `category` when the note has a category and that category differs from the chosen one. With "All notes" that is every categorized note. With "Uncategorized" there are none. Without subcategories every note is directly in the chosen category, so there are none either. The owner gets `{title, path}` and visitors `{title}`, as in recent notes.

**Category ID in the config.** The dashboard lookup sends every widget's config to visitors, so a public note list carries its `categoryId`, as a link list carries its folder ID. The rows never contain category IDs, and the ID alone reveals nothing. Stripping configs for visitors would be a dashboard-wide change and is out of scope.

**Response data.** Visitors get `{notes: [{title, icon, category}]}`, and the owner's rows add `id`. There's no `time`. These are new structs next to the recent notes ones, because those have `Time`.

**Edit-mode preview.** A pure `noteList(notes, categories, config)` in `widgets/noteList/list.ts` uses `filterNotes`, or a direct `categoryId === id` filter when subcategories are off. It sorts with `Intl.Collator(undefined, {sensitivity: "base"})` on `displayTitle`, with untitled last and ties by ID, then slices and adds the second line with `categoryPath`. If the category is missing, it returns `{missing: true}`. It is tested with the same cases as the Go side, like `recentNotes` / `flattenLinks`.

**Frontend reuse.** Generalise `RecentNotesView`'s row into a shared `NoteRows` component (icon, title, optional second line, optional trailing time). Recent notes passes the time and the note list doesn't. The view owns the `NoteDialog` open state keyed by note ID, as in recent notes. The settings use `CategoryPicker` extended with an optional second extra choice, so it can offer both "All notes" and "Uncategorized" above the tree. Alternative: a new picker, but that would duplicate the tree rendering. The subcategories checkbox is disabled unless a tree category is chosen.

## Risks / Trade-offs

- [Server collation and `Intl.Collator` can differ in rare cases] → Same accepted trade-off as link list; the server's order wins once saved.
- [Both this change and `add-recent-notes-widget` modify "Add widgets" in `dashboard-editing`] → This delta already includes "recent notes"; archive `add-recent-notes-widget` first.
- [Visitors see note titles and first lines once the flag is on] → Off by default, with a hint next to the flag, as in recent notes.
- [Alphabetical sort loads every note of the selection with content] → Acceptable for a personal note collection; revisit only if it shows up in profiling.
