# Design

## Context

Notes live in the `notes` table (`0017_notes.sql`). `backend/internal/notes` owns the store: `CreateNote` and `UpdateNote` with `NewNote` and `NotePatch`, optimistic locking through `version`, and the widget queries `RecentNotes` (`recent.go`) and `ListedNotes` (`listed.go`, with an in-Go sort for "Alphabetical"). `backend/internal/widgets` turns these into the rows of the widgets, for the owner and for visitors of a public dashboard. The selected notes widget loads its notes by ID in list order.

The frontend loads all notes with `GET /api/notes` and filters and orders them on `/notes` itself (`NotesPage`, `SidePanel` counts). The editor page and the dashboard modal share `NoteEditorPage`, whose Base UI menu holds only "Delete note" today. `NoteRows` renders the rows of the three list widgets.

## Goals / Non-Goals

**Goals:**
- Two flags on notes that the later `memos-api` and `karakeep-note-pin-archive` changes can map onto, with no change to their semantics needed.
- Ordering and filtering by the flags in one place per surface: the server for widgets, the client for `/notes`.

**Non-Goals:**
- Changing the Karakeep, Linkwarden or import behaviour.
- Bulk pin or archive, pinning on cards, or a menu on cards.
- A separate pin order (pinned notes are ordered by the surface's normal sort).

## Decisions

**Two boolean columns.** The migration `0020_note_pin_archive.sql` adds `pinned INTEGER NOT NULL DEFAULT 0` and `archived INTEGER NOT NULL DEFAULT 0` to `notes`. Existing notes become unpinned and unarchived. Considered: an `archived_at` / `pinned_at` time. Memos and Karakeep only need booleans, and "pinned first" needs no pin time, so plain flags keep the API and the store simple.

**Flags go through the normal create and patch.** `NewNote` and `NotePatch` get `Pinned` and `Archived` (`*bool`); the HTTP layer decodes them from JSON and answers `invalid_note` for anything that isn't a boolean. Because they are ordinary patch fields, they bump the version and set `updated_at` like every other field, as the user decided ("ordering: yes it does"). No separate `/pin` or `/archive` endpoints: the connected apps will call the store, not HTTP, and one code path keeps version handling the same.

**`GET /api/notes` keeps its order and returns archived notes.** The client already filters by category; filtering by `archived` there too avoids a second request when switching to "Archived" and keeps the side panel counts in one pass. The page sorts pinned first with a stable sort over the server's order.

**Widgets order on the server.** `RecentNotes` and `ListedNotes` order by `pinned DESC` before their time column, so `LIMIT` applies after pinned notes come first. The "Alphabetical" sort in Go becomes a stable partition of pinned and unpinned around the existing title comparison. The selected notes widget partitions its loaded notes stably, keeping list order inside each group. Archived notes aren't filtered out anywhere in the widgets. The resolved rows get `pinned` and `archived`, also in the visitor response.

**"Archived" is a pseudo-category in the address.** `?category=archived` works like `?category=none`. Category IDs are random strings, so they can't collide with these words.

**Editor menu actions save through the autosave.** "Pin" / "Archive" set the field in the editor's draft and flush the autosave right away, instead of sending their own PATCH. That way unsaved text and the flag go in one request with one version, and conflicts, retries and the save indicator behave like any other save. On `/notes/new` this also creates the note, like any first change.

**One mark component.** A small `NoteMarks` component renders the pin icon and the archived mark (lucide `Pin` and `Archive`) with their labels or tooltips, and is used by `NoteCard`, the editor header, `NoteRows` and the note widget. Dimming is the `opacity` of the card or row, so the note colour still shows.

## Risks / Trade-offs

- [Pinning moves a note to the top of "Last changed" lists even among unpinned notes once unpinned again] → That's the chosen semantics (a flag change is an edit). The scenarios record it.
- [Old pinned notes take up slots in count-limited widgets] → Intended: pinned notes count against the widget's number of notes, and the "Pinned within the count" scenario documents it. Day limits still apply, so "Recent notes" doesn't fill up with old pinned notes.
- [Karakeep still reports every note as not archived until `karakeep-note-pin-archive`] → Archived notes keep showing in the Karakeep apps in the meantime. Acceptable for one release; the third change fixes it.
- [`GET /api/notes` grows with the archive] → Same as today: all notes are already loaded. A server-side filter can be added later without spec changes to the page.

## Migration Plan

The migration only adds columns with defaults; no data is rewritten. Rolling back the binary leaves the columns unused; older code ignores them.
