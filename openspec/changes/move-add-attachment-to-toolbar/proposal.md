# Proposal

## Why

The attachments dropdown of the note editor doesn't render correctly: each entry puts two menu items side by side ("Open" and "Remove from note"), and the dropdown mixes the list of files with an "Add attachment" action. Adding a file is an editing action that belongs with the other editing actions, next to the link button in the toolbar.

## What Changes

- The attachments dropdown only lists the note's files: preview or icon, name and size. Activating an entry opens it, as before. The "Remove from note" action and the "Add attachment" entry are removed from it. A file is removed from a note by deleting its block in the content.
- The content editor's toolbar gets an "Add attachment" / "Anhang hinzufügen" button with a paperclip-plus icon, right after the link button. It opens the file picker and inserts the chosen files as before (at the cursor, otherwise at the end).

## Capabilities

### New Capabilities

### Modified Capabilities
- `note-attachments`: "Attachments button" lists the files only; "Adding attachments" starts from the toolbar button instead of the dropdown.
- `note-document`: "Content editor" lists the attachment button in the toolbar.

## Impact

- Frontend only: `features/notes/AttachmentsButton.tsx`, `features/notes/editor/Toolbar.tsx`, `features/notes/NoteEditorPage.tsx`, the attachment tests, en/de strings.
- `REMOVE_ATTACHMENT_COMMAND` loses its only caller and is removed.
