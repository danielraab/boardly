# Tasks

## 1. Shared groundwork

- [ ] 1.1 Move the symbol list from `widgets/link.go` to `backend/internal/symbols` and `icons.ts` to `frontend/src/components/icons/symbols.ts`, updating all imports and the sync comments; verify the existing link widget Go and Vitest tests still pass
- [ ] 1.2 Make `IconChooser` reusable (move to `frontend/src/components/IconChooser.tsx`, optional `favicon`; without it no "Website icon" tab and start on "Symbol") and verify the link widget tests pass plus a new test that the chooser without `favicon` shows only "Own image" and "Symbol"
- [ ] 1.3 Add the 9 `--color-note-<name>` tokens to the default block and all 7 theme blocks in `styles.css`, plus `features/notes/colours.ts`, and extend `contrast.test.ts` (ink/muted ≥ 4.5, accent ≥ 3 against each note colour, ≥ 1.1 against `paper`); verify the test passes for every theme
- [ ] 1.4 Add the Lexical packages (`lexical`, `@lexical/react`, `@lexical/rich-text`, `@lexical/list`, `@lexical/markdown`, `@lexical/link`, `@lexical/history`) with exact versions, check that every new package's license is MIT, and verify `pnpm install` and `pnpm build` succeed

## 2. Database and document format (backend)

- [ ] 2.1 Add migration `0017_notes.sql` (`note_categories`, `notes`, indexes, constraints as in design.md) and verify with a Go test that it applies, that the icon `CHECK` rejects both icon columns set, and that deleting an image nulls the icon columns
- [ ] 2.2 Add `backend/internal/notes/doc.go` with the `NoteDoc` types, strict decoding and validation (version, unknown types or fields, heading level, URL schemes, list depth ≤ 6, ≤ 512 KB) and verify with table tests over fixtures in `testdata/docs/valid` and `testdata/docs/invalid`
- [ ] 2.3 Add task ID normalization (missing → new, duplicate → new for later occurrences, valid IDs kept, IDs only on task items) and verify with Go tests for each case

## 3. Notes and categories store and API (backend)

- [ ] 3.1 Add icon and colour validation shared by notes and categories (symbol known, image is an upload of the same user, colour in the palette) and verify with Go tests, including another user's image
- [ ] 3.2 Add the category store (list, create, update with the ancestor cycle check, delete that moves children to the parent in one transaction) and verify with Go tests: nesting, move to the top level, move into itself or a descendant → `invalid_parent`, another user's parent, delete with notes and subcategories, notes' `updated_at` unchanged on delete
- [ ] 3.3 Add the notes store (list ordered by `updated_at DESC, id`, get, create, update with the version check → `ErrVersionConflict`, delete, category ownership check) and verify with Go tests for ordering, version increment, stale version, title limit and another user's category or note
- [ ] 3.4 Add the handlers in `httpapi/notes.go` and the routes for `/api/notes` and `/api/note-categories` with the error codes from the specs, and verify with httptest tests: 401 without a session, 404 for another user, 400 codes, 409 `version_conflict`, 201/200/204 bodies
- [ ] 3.5 Extend `images.Store.Sweep` to keep uploads used by `notes.icon_image_id` and `note_categories.icon_image_id`, and verify with a cleanup test that an old upload used as a note icon and one used as a category icon are kept

## 4. Document mapping and rendering (frontend)

- [ ] 4.1 Add `features/notes/document/` with the `NoteDoc` types and `displayTitle` (title, first text line cut at 80 characters with "…", "Untitled note"), and verify unit tests
- [ ] 4.2 Add the Lexical ↔ `NoteDoc` mapper (all blocks, marks, links, nested lists, task items with `checked` and the task ID from NodeState, or a `ListItemNode` subclass as fallback), and verify round-trip tests over the shared valid fixtures and that unsupported formatting is dropped
- [ ] 4.3 Add `NoteRenderer` (read-only `NoteDoc` → React, task items as disabled checkboxes, links inert) and verify Testing Library tests for each block type

## 5. API hooks and routes (frontend)

- [ ] 5.1 Add `features/notes/api.ts` with TanStack Query hooks for notes and categories (list, get, create, patch with version, delete) and typed errors for `version_conflict`, and verify with tests against the test API helper
- [ ] 5.2 Add the routes `notes.tsx` (layout with side panel), `notes.index.tsx` (grid, `?category=` search param) and a lazy-loaded `notes.$noteId.tsx`; redirect signed-out visitors to `/login`; set the document titles; verify with route tests for the redirect, the titles and not-found for an unknown ID
- [ ] 5.3 Add the "Notes" link to the user menu between "Favourites" and "Settings" and verify the updated user menu tests

## 6. Categories UI

- [ ] 6.1 Add category tree helpers (build tree, path, descendants, note counts including subcategories, locale-aware title sort) and verify unit tests
- [ ] 6.2 Add the side panel ("All notes", "Uncategorized", collapsible tree with icons and counts, "New category", choice in the address, unknown category → "All notes") and the phone drawer that closes after a choice; verify with Testing Library tests
- [ ] 6.3 Add the category dialog (title required, icon chooser without website icon, colour, parent without itself and descendants) and the per-category menu (Edit, New subcategory with the parent preselected, Delete with a confirmation that names the counts); verify with tests for create, rename, move, parent choices and delete
- [ ] 6.4 Add the category badge (icon, title, colour or `surface` with border, path tooltip) and the category picker ("No category" + tree), usable on cards without opening the note; verify with tests for the tooltip, changing from a card and removing the category

## 7. Notes grid

- [ ] 7.1 Add the notes grid and cards (responsive columns, one at phone width, icon, title, rendered content with max height and fade, colour background, badge, whole card opens the note) with the empty states and the category filter; verify with Testing Library tests for order, filtering incl. subcategories, empty states and the card contents

## 8. Editor page

- [ ] 8.1 Add the Lexical editor (rich text, lists and check lists, links with the http/https/mailto check, history, Markdown shortcut transformers from the spec, placeholder, Tab/Shift+Tab, Ctrl/Cmd+click to open links) and the task-ID node transform; verify with tests for each shortcut, Enter in a task item and duplicate IDs after splitting an item
- [ ] 8.2 Add the toolbar (heading, bullet, ordered, task, bold, italic, strikethrough, code, link with validation) and verify with tests that each button applies its format and that an invalid link shows the message
- [ ] 8.3 Add the editor page header (back link, icon button, title field, badge or "Add category", colour button with the palette and "No colour", menu with "Delete note" and its confirmation) with the note colour as background; verify with tests for each control and for delete/cancel
- [ ] 8.4 Add autosave (1 s debounce, one request at a time, status "Saving…"/"Saved"/"Not saved" + "Retry", flush on leave with `keepalive`, `beforeunload` prompt, conflict banner with read-only editor and "Reload") and verify with fake-timer tests for each state, including a 409
- [ ] 8.5 Add `/notes/new` (local draft, POST on the first change, `replace` to `/notes/<id>` without remounting, preselected category from the side panel, no note when left unchanged) and verify with tests

## 9. Translations, docs and checks

- [ ] 9.1 Add all en + de strings (page, cards, empty states, editor, toolbar, autosave, categories, dialogs, colour names, menu entry) and verify the i18n test passes
- [ ] 9.2 Document notes and categories in README.md (feature section) and verify the README describes the Notes page, categories and that content is stored as JSON
- [ ] 9.3 Run `make check` and `make test` and verify both pass
- [ ] 9.4 Run the app and check by hand: create a note with Markdown shortcuts and tasks, tick tasks, autosave and reload, two-tab conflict, colours in a light and a dark theme, nested categories with filter and delete, badge change from a card, phone width with the drawer
