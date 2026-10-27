# Proposal

## Why

Boardly holds links but has no place for free text: shopping lists, sprint todos, scribbled ideas (issue #19). Notes with checkable todos make it a more complete personal start page. The note content is stored in a structured format of our own, so later features (todo widgets, link tagging, imports, a Memos-compatible API) can build on stable, addressable blocks instead of parsing text.

## What Changes

- New **Notes** page at `/notes`, linked from the user menu: a card grid of the user's notes, newest edit first, with a category side panel (a drawer at phone width) to filter by category.
- Each note has an optional title, an optional icon (symbol or own image, chosen with the existing icon chooser), an optional colour from a fixed palette, at most one category, and rich content.
- New editor page `/notes/$noteId`: a rich text editor (Lexical, MIT) with Markdown shortcuts while typing (`## `, `- `, `1. `, `- [ ] `, `**bold**`, `` `code` `` …) and checkable task items. Changes are saved automatically after a short pause. A new note is only created once something is typed.
- The content is stored as a versioned JSON document of our own (`{v: 1, blocks: […]}`), not as Markdown and not as the editor's own format. The server validates it strictly. Every node type has a Markdown representation, so Markdown import and export remain possible later. Every task item has a stable ID.
- **Note categories**: nested, with a required title and an optional icon and colour, managed in the side panel (create, rename, edit, move to another parent, delete). Categories stay when no note uses them. Deleting a category makes its notes uncategorized and moves its subcategories to its parent. A note shows its category as a badge; clicking the badge changes the category.
- A fixed palette of named note colours, defined for each of the 7 themes and meeting the contrast minimums.
- Uploaded images used as note or category icons are kept by the unused-upload cleanup.
- Deleting a note asks for confirmation; there is no trash.

Non-goals (later changes): note and todo dashboard widgets, Karakeep / Linkwarden note import, a Memos-compatible API, export (notes and favourites), link tagging in notes, search, pasting Markdown as blocks (only if it falls out for free), manual note order and drag and drop of categories.

## Capabilities

### New Capabilities
- `notes`: the note entity, its API, the `/notes` card grid and the `/notes/$noteId` editor page with autosave, title, icon, colour and category badge.
- `note-document`: the stored JSON document format of a note, its server-side validation, stable task IDs, the Markdown-mappable node set, and the editor's Markdown shortcuts.
- `note-categories`: nested note categories with title, icon and colour, their API, the side panel with filtering and management, and what happens on delete.

### Modified Capabilities
- `app-shell`: the signed-in user menu gets a "Notes" link between "Favourites" and "Settings".
- `theming`: themes define a palette of note colours that meets contrast minimums.
- `images`: the cleanup of unused uploads also keeps images used as note or category icons.

## Impact

- Backend: new package `backend/internal/notes` (store, document validation), new migration with `note_categories` and `notes`, new endpoints under `/api/notes` and `/api/note-categories`, the symbol list moved out of `widgets` so notes can validate symbols, `images` cleanup query extended.
- Frontend: new dependencies `lexical`, `@lexical/react`, `@lexical/rich-text`, `@lexical/list`, `@lexical/markdown`, `@lexical/link`, `@lexical/history` (all MIT, versions pinned); new feature folder `frontend/src/features/notes`, routes `notes.tsx` / `notes.$noteId.tsx`, the `IconChooser` made reusable without the "Website icon" tab, note colour tokens in the theme definitions, en + de translations.
- No change to dashboards, favourites, Karakeep or Linkwarden APIs.
