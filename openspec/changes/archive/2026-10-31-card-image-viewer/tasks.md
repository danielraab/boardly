# Tasks

## 1. Images open on cards

- [x] 1.1 Render the image of `AttachmentView` as a `pointer-events-auto` button whenever `onOpenImage` is given, independent of `interactive`; verify that files, audio and video stay inert without `interactive`
- [x] 1.2 Let `NoteRenderer` always pass `openImage` and render the image viewer, and update its doc comment; verify with a `NoteRenderer` test that without `interactiveAttachments` the image opens the viewer while `report.pdf` has no download link
- [x] 1.3 Verify with a notes page test that clicking an image on a card opens the viewer without starting inline editing and closing it returns focus to the image (clicks on file cards falling through to the edit button rely on `pointer-events-none`, which jsdom doesn't apply, so that part is covered by the unchanged card markup and the inert-file test of 1.2)

## 2. Wrap-up

- [x] 2.1 Run `make check` and `make test` and verify both pass
