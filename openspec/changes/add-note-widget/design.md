# Design

## Context

Content widget kinds live in `backend/internal/widgets` as a `dashboards.Kind` with `Normalize` (validates the config on layout save) and `Resolve` (loads `data` per `dashboards.Viewer` on lookup). Kinds that show private data (explorer, link list, recent favourites) have `showIfPublic` and return "left out" for visitors without it, and return a `{"missing": true}` sentinel to the owner when their target is gone. `widgets.Register` currently gets the favourites store and favicons only. Notes live in `backend/internal/notes` (`Store.Note`, `Store.Categories`, `UpdateNote` with optimistic versioning).

On the frontend, each kind is a folder under `features/dashboards/widgets/` exporting a `WidgetKind` listed in `registry.ts`; `WidgetPicker` renders `widgetKinds` as one flat list. Fixed-height content widgets already scroll (`SectionGrid`'s widget cell is `overflow-auto`). `NoteCard` renders a note with `NoteRenderer` (which takes `onToggleTask`), `ChosenIcon`, `noteBackground` and `CategoryBadge`, and saves checkbox toggles through `useAutosave` with conflict handling. Images are served to anyone, so note icon images work for visitors.

## Goals / Non-Goals

**Goals:**
- One note per widget, resolved on the server so visitors never call the notes API.
- Reuse the card's pieces (renderer, icon, background, badge look, toggle-and-save path) instead of a second note rendering.

**Non-Goals:**
- Inline text editing, changing the category, or creating a note from the widget.
- A max-height/fade mode; the common height choice is used.
- Live updates when the note changes in another tab while the dashboard is open.

## Decisions

**Config shape.** `{noteId, size, showIfPublic}`, decoded strictly like the other kinds. `noteId` must be non-empty and at most `maxIDLength`; its existence is not checked on save (like a link list's folder), so a deleted note never blocks saving the layout.

**Resolve.** `Register` takes the notes store as well. `Resolve` loads `notes.Store.Note(ownerID, noteId)`:
- visitor and `!showIfPublic` → left out;
- not found → `missingNote` (`{"missing": true}`) for the owner, left out for visitors;
- otherwise data `{title, icon, color, content, category}` with `category` = `{title, icon, color}` or null. The owner's data also has `id`, `version` and `category.path` (titles from the root down, for the tooltip). Category lookup uses `Store.Categories` (one query per widget; a user's categories are few). Building the visitor data as its own struct, rather than stripping fields, makes "nothing else" hard to break.

**Owner view from server data, edit mode from the notes query.** Outside edit mode the view renders `data` like the link widget does. In edit mode a new or changed widget has no data, so the view looks the note up in `notesQueryOptions` and `noteCategoriesQueryOptions` (the owner is the only one in edit mode). Alternative: always fetch the notes list for the owner; rejected because it adds a request per dashboard view and the data is already in the lookup.

**Toggling.** The owner view keeps the resolved note in local state and saves toggles through `useAutosave` with `saveNow({content})`, the same path as the card, so rapid toggles chain versions correctly and `version_conflict` is detected. On conflict or failure it invalidates the dashboard view query (to show the server's note) and shows the message inside the widget, using the card's existing strings. On success it also invalidates `notesQueryKey` so `/notes` is fresh.

**Shared note body.** Extract the non-editing card markup (frame with background, header with icon and title, renderer, badge slot) into a small component used by both `NoteCard` and the widget, if that keeps `NoteCard` readable; otherwise compose the same pieces in the widget. The widget passes no `onToggleTask` for visitors and in edit mode, which renders the checkboxes disabled. The badge is shown read-only: a non-interactive variant of `CategoryBadge` (or a `readOnly` prop) with the same look and the path or title as tooltip.

**Sizes.** `NoteRenderer` gets an optional size that applies `sizeStyles[size].text` (and a matching checkbox/icon size) to its root; the card keeps its current look by not passing it. Padding is the card's, independent of size.

**Picker groups.** `WidgetKind` gets `group: "favourites" | "notes"`. The picker iterates a fixed `widgetGroups` order, rendering a heading (`dashboards.widgetPicker.groups.<group>`) and the kinds of that group in registry order, skipping empty groups.

## Risks / Trade-offs

- [The owner's view shows the note as of page load] → Toggling saves against that version; a conflict reloads the server's note and says so. Reloading the page refreshes it.
- [A visitor sees the whole note once the flag is on] → The flag is off by default, and the settings dialog says next to it that visitors will see the note.
- [`Register`'s signature changes] → Only `main.go` and the httpapi test setup call it; update both.
