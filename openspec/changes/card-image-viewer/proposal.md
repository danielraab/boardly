# Proposal

## Why

Clicking a photo on a note card of `/notes` starts inline editing instead of showing the photo large. To look at an image, Ann has to open the note first, although note widgets and the editors already open the image viewer on a click.

## What Changes

- On the note cards of `/notes`, activating an image of the content opens the image viewer with the note's images, as in note widgets and the editors, instead of starting inline editing.
- The rest of the card's content, including file cards, audio and video, stays inert and starts inline editing as before.

## Capabilities

### New Capabilities

### Modified Capabilities
- `note-attachments`: "Attachments in shown notes" no longer excludes note cards from opening the image viewer.

## Impact

- Frontend only: `features/notes/NoteRenderer.tsx`, `features/notes/NoteCard.tsx`, `features/attachments/AttachmentView.tsx` and their tests.
