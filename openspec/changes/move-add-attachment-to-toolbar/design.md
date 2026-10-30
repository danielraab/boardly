# Design

## Context

`AttachmentsButton` sits in the header of the note editor page (and modal), outside the `LexicalComposer`. It holds the hidden file input and calls back into the editor through `editorRef` with `INSERT_ATTACHMENTS_COMMAND` and `REMOVE_ATTACHMENT_COMMAND`. Each dropdown row is a `div` with two `Menu.Item`s, both with the shared `menuItem` class (`w-full`), which is what breaks the layout.

`Toolbar` lives inside the composer and is only rendered where the full editor is (the editor page and modal, not inline card editing).

## Decisions

**The add button lives in `Toolbar`.** It's an `AttachButton` next to `LinkButton` with the toolbar's `toolButton` style, a `PaperclipPlus`-style icon (lucide `FilePlus2` if `PaperclipPlus` isn't available), and its own hidden file input. It dispatches `INSERT_ATTACHMENTS_COMMAND` on the composer's editor directly, so no `onAdd` goes through the page. It uses `onMouseDown={keepSelection}` like the other buttons, so the cursor position used for insertion is kept. Allowed types for `accept` come from `attachmentsQueryOptions`, fetched while the toolbar is enabled.

**The dropdown is a plain list.** One `Menu.Item` per attachment (preview/icon, name, size); activating opens it. No separator, no add entry, no remove action. `onAdd`, `onRemove` and the file input leave `AttachmentsButton`; "No attachments" stays for the empty case.

**`REMOVE_ATTACHMENT_COMMAND` is dropped** together with its handler, since nothing dispatches it any more. Removing a file from a note is done by deleting its block, like any other content.

## Risks / Trade-offs

- [Removing from the dropdown was quicker than finding the block in a long note] → The user asked for a plain list; the block can be selected and deleted in the content.
