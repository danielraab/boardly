# Design

## Context

- `importer.Parse` reads a file into a `favourites.ImportTree` (folders and links plus a report), and `favourites.Store.Import` writes it in its own transaction, reusing folders by case-insensitive name and skipping duplicate URLs. `httpapi.importFavourites` glues the two and returns the `favourites.ImportReport`.
- `notes.Store` (`DB`, `Now`) owns `notes` and `note_categories`. Content is a `notes.Doc` validated by `ParseDoc` (512 KB limit, list depth 6, task IDs normalized). There is no root category: "top level" means `category_id`/`parent_id` `NULL`.
- The default folder for new links is `users.share_folder_id` (migration 0016) with `ON DELETE SET NULL`, validated in `users.Store.Update` and edited in the Profile tab via `ShareFolderForm`. `note_categories` is deleted row by row with subcategories moved up, so `ON DELETE SET NULL` on a `note_categories(id)` reference works the same way.
- `SharePage` builds its initial values in `share/sharedLink.ts` from `title`, `text` and `url`.

## Goals / Non-Goals

**Goals:**
- One converter on the server, used by the import now and by the Karakeep API in `karakeep-text-bookmarks`.
- Links and notes of one import commit or roll back together.

**Non-Goals:**
- Converting a document back to Markdown (comes with `karakeep-text-bookmarks`, which needs it).
- Notes from Linkwarden or Netscape files (they have none), and a separate Markdown/notes import (later change).
- Using the default category for "New note" on `/notes` or anywhere besides `/share`.
- An "Add note" app shortcut.

## Decisions

- **Markdown parsing with goldmark** (`github.com/yuin/goldmark` with the `TaskList`, `Strikethrough`, `Linkify` and `Table` extensions), walked into `notes.Doc` in `internal/notes/markdown.go` as `FromMarkdown(string) Doc`. Tables are parsed only so they can be flattened to text rather than shown as pipes in a paragraph. Soft line breaks become `break` inlines, because Karakeep and phone notes are written line by line, not wrapped. Links with a scheme outside `linkSchemes` keep only their text. Alternative: a hand-written subset parser — smaller, but emphasis, nesting and edge cases of CommonMark are exactly where hand-rolled parsers break; goldmark is pure Go, widely used and has no dependencies.
- **Plain text** in the same file as `FromText(string) Doc`: split on blank lines into paragraphs, `\n` into `break`s, and a URL regexp for `http(s)` links. No Markdown, so shared `*`/`#` text stays literal.
- **Converter output is always valid**: every result goes through `json.Marshal` + `ParseDoc` in the callers, which assigns task IDs and enforces size; tests (including a fuzz test of `FromMarkdown`) assert that `ParseDoc` accepts any output under the size limit.
- **`POST /api/notes` with `text`**: `createNote` decodes an optional `text`, rejects it together with `content` (`invalid_request`), and passes `FromText(text)` as JSON content to `CreateNote`. Keeps conversion on the server, so the share page needs no client converter. Alternative: convert in TypeScript and send `content` — would duplicate the converter that the Karakeep API needs in Go anyway.
- **Parser result** — `importer.Parse` returns `importer.Result{Links favourites.ImportTree, Notes notes.ImportTree}`. `notes.ImportTree` is a root `ImportCategory{Name, Categories, Notes}` plus counts (`TooLarge`). `ParseKarakeep` builds the category tree from the same list tree as the folders (`treeOf`), then prunes lists without notes in their subtree. Size is checked in the parser (marshalled content > `MaxDocSize` → counted, dropped), so the store gets only importable notes. Tag line: `#` + `strings.Join(strings.Fields(tag), "-")`.
- **One transaction across both stores**: `favourites.Store.Import` and a new `notes.Store.Import` are split into `…ImportTx(ctx, tx *sql.Tx, …)` variants that only write, plus the existing `Import` wrapper for tests. `httpapi.importFavourites` begins the transaction on `favourites.DB` (the same `*sql.DB` as notes), checks the folder and category, runs both, commits, then calls `favourites.Store.AfterImport(report)` to wake the fetch workers. Alternative: one store calling the other — couples favourites to notes for a single endpoint.
- **Category check up front**: `categoryId` is checked before writing (`notes.ErrInvalidCategory` → 400 `invalid_category`) for every format, so a typo fails fast even for a Netscape file.
- **Categories reused by `strings.EqualFold`** on titles of the parent's children, loaded once per parent like `favourites.importer.add`. Category title cut to 200 (`normalizeCategoryTitle` rules).
- **Duplicate notes**: key = trimmed title + `\x00` + canonical content JSON with every task `id` cleared. Existing notes are loaded once per import (`SELECT title, content FROM notes WHERE user_id = ?`); keys of imported notes are added as they are written. Only with `skipDuplicates`.
- **Timestamps**: `created_at = updated_at =` the bookmark's time (or the import time), so old notes sort by their age on `/notes`.
- **Report**: the handler answers with one JSON object built from both reports (`links`, `folders`, `notes`, `categories`, `skipped.{duplicates, duplicateNotes, invalidUrls, notLinks, smartLists, tooLargeNotes}`, `shortenedDescriptions`). `favourites.ImportReport` keeps its shape; the response type lives in `httpapi`.
- **Default category**: migration `0018_note_category_default.sql` adds `users.note_category_id TEXT REFERENCES note_categories (id) ON DELETE SET NULL`; `users.User.NoteCategoryID` (`json:"noteCategoryId"`), `users.Patch.NoteCategoryID` validated with `SELECT 1 FROM note_categories WHERE id = ? AND user_id = ?` → `ErrInvalidCategory` → 400 `invalid_category`. Same pattern as `share_folder_id`.
- **Frontend category picker**: a `CategoryPicker` select (`notes/categories/CategoryPicker.tsx`) with indented options from `categories/tree.ts` and a configurable empty choice label ("No category" / "Top level"). Used in the Profile `NoteCategoryForm`, the import form and the share note form.
- **Share page**: `sharedLink.ts` also derives the initial mode (`link` when a URL was found) and the note text (`text` plus a missing `url` as last line). `SharePage` keeps both form states in one component and renders a Base UI `ToggleGroup` switch; the note form posts `{title, text, categoryId}` and its confirmation links to `/notes/<id>`. After saving it invalidates the notes queries.
- **Import form**: the category picker renders only for `format === "karakeep"`, and `categoryId` is sent only then; after success it invalidates the favourites, notes and categories queries.

## Risks / Trade-offs

- [Markdown edge cases produce something the document can't hold] → Fallback to text for every unknown node, plus the fuzz test against `ParseDoc`.
- [Lists become both folders and categories] → Only lists with notes become categories, so link-only lists don't clutter `/notes`.
- [Duplicate check loads all notes' content] → Fine at personal scale (thousands of notes, one query per import); no index needed.
- [Tags with whitespace change spelling (`weekend-plans`)] → Documented; the tag line is plain text and can be edited.
- [Shared text that is Markdown stays literal] → Intended: most shared text is plain, and turning `*` into emphasis would mangle it.

## Migration Plan

Additive migration with a nullable column; existing users start with `NULL` (no category). API changes are additive (new optional parameter, new fields). Rollback: older binaries ignore the column and the new parameters.
