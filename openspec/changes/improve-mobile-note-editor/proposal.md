# Proposal

## Why

The save status sits in the top action row of the note editor, which scrolls away, so on a longer note "Saved" is never seen. The failed state with "Retry" and the version conflict are just as invisible there, so a phone user can lose changes or face a read-only editor without any visible explanation.

## What Changes

- The save status moves from the top action row into the sticky formatting toolbar row, in that one place at every width and in both the editor page and the dashboard note dialog. Inline editing on cards has no toolbar and keeps its indicator.
- On phones, "Saving…" and "Saved" are compact icons with an accessible label. "Not saved" with "Retry" stays visible as text.
- A version conflict also shows in the status slot, with a "Reload" button, so it is reachable while scrolled down. The explanation at the top of the editor stays.
- The toolbar stays on a single row and scrolls sideways when it doesn't fit, with the status fixed at its right end outside the scrolling part.
- The sticky toolbar gets the note's background. Today it is transparent, so text scrolls visibly behind the buttons.
- The viewport meta tag asks for `interactive-widget=resizes-content`, so that on Android the on-screen keyboard doesn't push the sticky toolbar out of view.

Not part of this change: the doubled page and card padding on phones, and larger touch targets for the 32 px toolbar buttons.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `notes`: the editor's save status stays visible while scrolling, including failed saves and version conflicts.
- `note-document`: the editor toolbar stays on one row that scrolls sideways, and hides the content scrolling under it.

## Impact

- Frontend only: `NoteEditorPage`, `SaveIndicator`, `editor/NoteEditor`, `editor/Toolbar`, `index.html`, and new `notes.editor.status` strings in `en.json` and `de.json`.
- `interactive-widget=resizes-content` applies to the whole app: on Android, pages and fixed elements such as the mini player are laid out above the keyboard while it is open.
- Export and import are unaffected: no user data changes.
- The OpenAPI documents are unaffected: no route or request or response shape changes.
