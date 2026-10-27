# Proposal

## Why

Small changes to a note, like ticking off a todo or fixing a typo, currently mean opening the editor page and going back to `/notes`. That's slow for the quick edits notes get most often ([#57](https://github.com/danielraab/boardly/issues/57)).

## What Changes

- Task checkboxes on the cards of `/notes` can be toggled directly. The change is saved right away.
- Activating a card's title or content no longer opens the editor page. It turns the card into an inline editor for the title and content, with the Markdown shortcuts and keyboard shortcuts of the editor page but no toolbar. Changes are autosaved like on the editor page. Escape, or moving focus out of the card, ends inline editing.
- Every card gets an "Open note" button that opens `/notes/<id>` for everything else: icon, colour, delete and the toolbar.
- Cards changed on `/notes` keep their place in the grid until the page is opened again, so a card doesn't jump to the top while the user is editing it or ticking off its todos.
- **BREAKING** (behaviour): a click on a card no longer opens the note. The "Open note" button does.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `notes`: "Notes page" and "Note cards" change how a card is activated and how it reacts to its checkboxes; a new requirement "Inline editing on cards" covers editing a card's title and content in place and the grid order while doing so.

## Impact

- Frontend only: `frontend/src/features/notes/NoteCard.tsx`, `NoteRenderer.tsx`, `NotesPage.tsx`, `useAutosave.ts`, `editor/NoteEditor.tsx` (a variant without toolbar) and `NoteEditorPage.tsx` (the save indicator is shared).
- New translations in `en.json` and `de.json`.
- No backend, API or database changes; saving uses the existing `PATCH /api/notes/{id}` with its version check.
