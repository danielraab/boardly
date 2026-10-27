# Design

## Context

Boardly's data so far is favourites (`favourite_nodes`, tags) and dashboards. Notes are a separate domain: they share no table with favourites, and their categories have nothing to do with favourite folders (see proposal.md).

Things in the codebase this change builds on:
- Widgets store icons as `iconSource` (`favicon|image|symbol|none`) + `icon` + `iconImageId`. The allowed symbols live in `backend/internal/widgets/link.go` (`linkIcons`) and `frontend/src/features/dashboards/widgets/link/icons.ts`, kept in sync by hand.
- `frontend/src/features/dashboards/widgets/link/IconChooser.tsx` is the chooser with the tabs "Website icon", "Own image" and "Symbol". It always gets a favicon and a host.
- `images.Store.Sweep` (`backend/internal/images/cleanup.go`) keeps uploads that a widget config refers to (via `json_tree`) or that a favourite uses.
- Theme colours are CSS custom properties per `[data-theme]` in `frontend/src/styles.css`; `frontend/src/theme/contrast.test.ts` checks the minimum contrasts.
- The frontend has `useDebounced` and TanStack Query.

## Goals / Non-Goals

**Goals:**
- A stored document format that we own, that the Go backend can read and validate without the editor, and that later changes (todo widgets, link tagging, Memos API, Markdown import and export) can extend without a data migration.
- An editor that feels like writing Markdown, without storing Markdown.
- Groundwork for todo widgets: stable task IDs and optimistic versioning, already in v1.

**Non-Goals:**
- A `note_tasks` projection table, server-side Markdown conversion, and full-text search. They come with the changes that need them; the format is designed so they can be added from the stored documents.
- Real-time collaboration or merging of concurrent edits; a conflict stops the stale editor.
- Rendering notes on the server.

## Decisions

### Editor: Lexical, our own document format, a mapper in between

The editor is Lexical (MIT, no commercial tier), with `@lexical/react`, `rich-text`, `list` (incl. check lists), `markdown` (shortcut transformers), `link`, `history`, `code` (the code block node), `selection` and `utils`. Versions are pinned exactly, because Lexical is still 0.x.

**Shortcuts Lexical doesn't cover.** Lexical's own transformers allow headings up to level 6 and have no divider, so the editor uses its own `HEADING` (levels 1–3) and `DIVIDER` transformers, and a `LINK` wrapper that only accepts http, https and mailto. Lexical runs block shortcuts only at the start of a top-level paragraph, so after `- ` has made a bullet item, the following `[ ] ` would stay text. A small `TaskShortcutPlugin` turns a bullet or numbered item that starts with `[ ] `, `[] ` or `[x] ` into a task, splitting the list around it when it has other items.

The stored format is **not** Lexical's serialized JSON. A TypeScript mapper `lexicalToDoc` / `docToLexical` in `frontend/src/features/notes/document/` converts between the editor state and `NoteDoc`:

```ts
type NoteDoc = { v: 1; blocks: Block[] };
type Block =
  | { type: "paragraph"; content: Inline[] }
  | { type: "heading"; level: 1 | 2 | 3; content: Inline[] }
  | { type: "quote"; content: Inline[] }
  | { type: "code"; text: string }
  | { type: "divider" }
  | { type: "list"; style: "bullet" | "ordered" | "task"; items: Item[] };
type Item = { id?: string; checked?: boolean; content: Inline[]; children?: ListBlock };
type Inline =
  | { type: "text"; text: string; marks?: ("bold" | "italic" | "strike" | "code")[] }
  | { type: "link"; url: string; content: TextRun[] }
  | { type: "break" };
```

`id` and `checked` are only allowed on items of a `task` list. Lexical's own JSON is verbose (`format`, `indent`, `direction` on every node) and still changing in 0.x. Keeping it out of the database means a Lexical update or an editor switch only touches the mapper, and the Go code works with a small schema it fully understands.

*Alternatives:* storing Lexical JSON as is (no mapper, but Go would have to understand it and any format change would need a data migration); Tiptap (open core with proprietary Pro extensions, rejected by the user); storing Markdown (no stable IDs, and later features would have to parse text).

**Markdown rule.** Every block, inline node and mark must have a Markdown form (task list → `- [ ]` / `- [x]`, divider → `---`, link → `[text](url)`). A future node type such as a favourite link has to define its Markdown form too (e.g. `[title](url)`). This keeps the Memos API and imports possible. The mapper's tests check the editor's Markdown shortcuts against this list.

**Pasting Markdown** is not built separately. If Lexical's `$convertFromMarkdownString` can handle pasted plain text with little code, it may be added; otherwise it's a later change.

### Stable task IDs

Lexical's `ListItemNode` has `checked` but no ID. The ID is kept in **NodeState** (`createState('taskId', …)`) on the list item. The `$` state is copied when Lexical splits or copies a node, so on Enter and on paste the editor would duplicate IDs. Two safeguards:
1. a node transform on check list items gives a new ID (`crypto.randomUUID()`, shortened to 12 base64url characters) to an item whose ID is missing or already used in the editor;
2. the server repeats this check on save (see "Stable task IDs" in the `note-document` spec), keeping the first occurrence.

If NodeState isn't available in the pinned Lexical version, the fallback is a subclass of `ListItemNode` registered via node replacement. Both options only change the mapper.

### Backend package `internal/notes`

- `doc.go`: Go types for `NoteDoc`, strict decoding (`DisallowUnknownFields`, discriminated by `type`), validation (the limits from the spec: version 1, heading level, URL scheme, list depth ≤ 6, ≤ 512 KB), and `normalizeTaskIDs`. It returns `ErrInvalidDocument`.
- `notes.go`: the store with `List`, `Get`, `Create`, `Update(ctx, userID, id, version, patch)`, `Delete`. `Update` runs `UPDATE … SET …, version = version + 1 WHERE id = ? AND user_id = ? AND version = ?` and returns `ErrVersionConflict` when no row changed but the note exists. A PATCH with only `version` and no fields is a no-op.
- `categories.go`: the category store. A parent check walks the ancestors with a recursive CTE to reject cycles. On delete, in one transaction: `UPDATE note_categories SET parent_id = <deleted's parent> WHERE parent_id = ?`, then delete. The notes' `category_id` becomes NULL through `ON DELETE SET NULL`, which doesn't touch `updated_at`.
- `icons.go`: icon validation, shared with categories.
- HTTP handlers in `backend/internal/httpapi/notes.go`, routes in `router.go` next to the others, error codes `invalid_note`, `invalid_document`, `invalid_category`, `invalid_parent`, `version_conflict`, `not_found`.

### Shared symbol list

`linkIcons` moves from `widgets/link.go` to a small package `backend/internal/symbols` (`symbols.Valid(name)`), which `widgets` and `notes` both use. On the frontend, `icons.ts` moves to `frontend/src/components/icons/symbols.ts`, and the comments of both still point at each other.

### Icon field shape

Notes and categories use `icon: null | {type: "symbol", name} | {type: "image", imageId}` in the API, instead of the widgets' three flat fields. There is no favicon, so there is no `none` kind to tell apart from "website icon". The tables store `icon_symbol` and `icon_image_id` (at most one set, enforced by a `CHECK`), with `icon_image_id REFERENCES images (id) ON DELETE SET NULL`. Real columns make the cleanup query a plain `NOT EXISTS`. The server checks that an image is an `upload` of the same user.

### Reusable icon chooser

`IconChooser` gets an optional `favicon` prop. Without it, the "Website icon" tab is left out and the chooser starts on "Symbol". The link widget passes it as before. The component moves to `frontend/src/components/IconChooser.tsx`. The value type for notes is mapped to `IconChoice` and back in the notes feature.

### Schema

Migration `0017_notes.sql`:

```sql
CREATE TABLE note_categories (
  id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  parent_id TEXT, title TEXT NOT NULL,
  icon_symbol TEXT, icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
  color TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  UNIQUE (id, user_id),
  FOREIGN KEY (parent_id, user_id) REFERENCES note_categories (id, user_id),
  CHECK (icon_symbol IS NULL OR icon_image_id IS NULL));
CREATE TABLE notes (
  id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  category_id TEXT REFERENCES note_categories (id) ON DELETE SET NULL,
  title TEXT, icon_symbol TEXT, icon_image_id TEXT REFERENCES images (id) ON DELETE SET NULL,
  color TEXT, content TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  CHECK (icon_symbol IS NULL OR icon_image_id IS NULL));
CREATE INDEX notes_user_updated ON notes (user_id, updated_at DESC, id);
CREATE INDEX note_categories_parent ON note_categories (parent_id);
```

The composite foreign key keeps a parent inside the owner's categories, like `favourite_nodes`. The parent FK has no `ON DELETE` action, because the store moves the children up before deleting. That a note's category belongs to the same user is checked in the store, because a composite FK with `ON DELETE SET NULL` would also null `user_id`. IDs are generated like those of other tables. `content` holds the normalized `NoteDoc` JSON.

### Frontend structure

```
frontend/src/routes/notes.tsx            layout: sign-in check + <Outlet/>
frontend/src/routes/notes.index.tsx      card grid with side panel (?category=<id>|none)
frontend/src/routes/notes.$noteId.tsx    editor      ($noteId = "new" for a draft, ?category=<id> preselects)
frontend/src/features/notes/
  api.ts            query options and requests: notes list/get/create/patch/delete, categories
  NotesPage.tsx, NoteCard.tsx, NoteRenderer.tsx (read-only NoteDoc → React, no Lexical)
  NoteEditorPage.tsx, useAutosave.ts, editor/ (Lexical composer, toolbar, plugins, transformers)
  document/ (types, lexical mapper, displayTitle, tests)
  categories/ (SidePanel, CategoryDialog, DeleteCategoryDialog, CategoryBadge with picker, tree helpers)
  ColourPicker.tsx, colours.ts (palette names ↔ CSS variables), noteIcon.ts
```

`NoteRenderer` renders cards straight from `NoteDoc`, so the grid doesn't load Lexical. The router's automatic code splitting puts the editor route, and with it Lexical, into its own chunk. The side panel lives on the grid only; "New note" passes the chosen category to `/notes/new` as `?category=`.

**Filtering and counts** happen on the client: `GET /api/notes` returns every note (personal scale), and the side panel derives the subtree sets from `GET /api/note-categories`. A `?category=` filter on the server can come later without changing the specs.

### Autosave and versioning

The editor keeps a local draft (title, icon, colour, category, doc) and the last saved `version`. A change starts a 1 s debounce; when it fires, the editor sends the whole changed set of fields with `version`. There is only ever one request in flight, and changes made during a request are sent after it. On success, the response's `version` becomes the base. On `version_conflict`, the editor goes read-only and shows the reload banner. On leaving the page (route change or `pagehide`), pending changes are sent right away (`fetch` with `keepalive`). `beforeunload` asks for confirmation while something is unsaved.

Badge changes on cards send a PATCH with the card's `version` from the list query and then invalidate the list. On a conflict, they reload the list and show a toast.

`/notes/new` keeps the draft locally until the first change, then POSTs, and replaces the route with `/notes/<id>` (`replace: true`) without remounting the editor.

### Note colours

`styles.css` gets `--color-note-<name>` for the 9 colours in the default block and in each of the 7 `[data-theme]` blocks: light tints for light themes, dark shades for dark themes. `contrast.test.ts` is extended with ink/muted ≥ 4.5 and accent ≥ 3 against each note colour, and a minimum difference from `paper` (RGB distance ≥ 24, since light tints on a light paper differ in hue more than in luminance). Categories use the same tokens for their badges.

### Image cleanup

`Sweep`'s upload query gets two more `NOT EXISTS` clauses, for `notes.icon_image_id` and `note_categories.icon_image_id`.

## Risks / Trade-offs

- [Lexical 0.x breaking changes] → Exact version pins, the mapper is the only code that touches serialized state, and mapper round-trip tests catch format changes on updates.
- [The mapper loses information Lexical has but our format doesn't (alignment, indent of paragraphs, colours from pasted HTML)] → Intended. The editor registers only the nodes we map and strips unsupported formatting on paste, so what the user sees is what is stored.
- [Two parsers of one format (TS mapper and Go validator) drift apart] → Shared JSON fixtures in `backend/internal/notes/testdata/docs/` (valid and invalid documents) that both the Go tests and the Vitest tests read.
- [Lost edits when a tab is closed during the debounce] → `keepalive` flush on `pagehide`, plus the `beforeunload` prompt.
- [A conflict throws away unsaved text in the stale tab] → The banner keeps the stale content visible (read-only) until "Reload", so it can be copied.
- [All notes in one list request] → Fine for personal use. Pagination can be added behind the same endpoint later.
- [Duplicated task IDs from copy/paste] → Fixed by the editor transform and again by the server.

## Migration Plan

One new migration that only adds tables; no existing data changes. Rolling back the binary leaves the unused tables in place, which is harmless.

## Open Questions

- The exact tints of the 9 note colours per theme are picked during implementation, guided by the contrast test.
- Whether pasting Markdown comes almost for free with Lexical (see above).
