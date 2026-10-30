# Design

## Context

`NoteCard` renders `NoteRenderer` inside a `pointer-events-none` wrapper (`relative z-[1]`) and covers the card with an absolutely positioned edit button, so clicks on the content fall through to the edit button. Task checkboxes opt back in with `pointer-events-auto`. `NoteRenderer` only makes attachments interactive with `interactiveAttachments` (note widgets), which also enables file downloads and media players; `AttachmentView` renders the image as a button only when both `interactive` and `onOpenImage` are set.

## Goals / Non-Goals

**Goals:**
- Images on cards open the image viewer with the note's images, like in widgets.

**Non-Goals:**
- Downloading files or playing audio and video from cards; they stay inert.

## Decisions

**`AttachmentView` renders the image button whenever `onOpenImage` is given**, independent of `interactive`, which keeps governing downloads and players. The button gets `pointer-events-auto` so it receives clicks inside the card's `pointer-events-none` wrapper, the same way the checkboxes do; elsewhere the class has no effect.

**`NoteRenderer` passes `openImage` and renders the viewer always**, not only with `interactiveAttachments`. Every place that renders `NoteRenderer` (cards and widgets) wants images to open, so no new prop is needed. Its doc comment and the "is inert on cards" test are updated.

Alternative: a separate `viewImages` prop for cards. Rejected, since no caller would turn it off.

## Risks / Trade-offs

- [A click meant to start editing lands on a large image] → The rest of the card, including the title and other blocks, still starts editing; on a card that is only an image, the "Open note" button and the keyboard (Enter on the card's edit button) still edit it.
