# Tasks

## 1. Toolbar button

- [x] 1.1 Add an "Add attachment" button with its file input to `Toolbar`, right after the link button, dispatching `INSERT_ATTACHMENTS_COMMAND` and offering the allowed types; verify with `Attachments.test.tsx` cases for "Insert at the cursor", "Insert at the end" and "Attachment button in the toolbar"

## 2. Plain dropdown

- [x] 2.1 Reduce `AttachmentsButton` to a list of the note's files (one menu item each, no remove, no add, no file input), drop `onAdd`/`onRemove` from `NoteEditorPage` and remove `REMOVE_ATTACHMENT_COMMAND`; verify with "List" and "Only files" tests and that opening an image still shows the viewer

## 3. Strings and wrap-up

- [x] 3.1 Move the "Add attachment" string to `notes.editor.toolbar.attach` and drop the unused `attachments.button` strings in English and German; verify the i18n test passes
- [x] 3.2 Run `make check` and `make test` and verify both pass
