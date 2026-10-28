# Tasks

## 1. Widget kind (backend)

- [x] 1.1 Add the `note` kind in `backend/internal/widgets/note.go` with config validation (`noteId` required and at most `maxIDLength`, size S/M/L, `showIfPublic`) and verify Normalize tests accept a valid config and reject a missing note ID, a bad size and unknown fields with `invalid_layout`
- [x] 1.2 Implement Resolve (left out for visitors without the flag; `missing` for the owner and left out for visitors when the note is gone; owner data with id, version, title, icon, colour, content and category with path; visitor data with only title, icon, colour, content and category title/icon/colour) and verify with Go tests for each case, including that the visitor response has no version, no ID, no parent category names and nothing from other notes
- [x] 1.3 Pass the notes store to `widgets.Register` (`main.go`, httpapi test setup), add an httptest-level check that a dashboard lookup returns the note's data, and verify `make test` passes for the backend

## 2. Picker groups (frontend)

- [x] 2.1 Add `group` to `WidgetKind`, set it on the four existing kinds, and render the picker grouped under headings in the fixed order; verify with a Testing Library test that the headings and kinds appear in order (en and de)

## 3. Note widget (frontend)

- [x] 3.1 Add an optional size to `NoteRenderer` (text, checkbox and icon scale from `sizeStyles`), keeping the card unchanged, and verify with a renderer test that S/M/L set different text classes and no size keeps the current classes
- [x] 3.2 Add a read-only mode to the category badge (same look, tooltip given by the caller, no picker) and verify with a test that it opens nothing on click
- [x] 3.3 Add `widgets/note` settings (note select by display title, size, public flag with hint, common width and height; can't confirm without a note) and add the kind to `registry.ts` in the "notes" group; verify with a Testing Library test of the defaults, the untitled display title and validation
- [x] 3.4 Add the view: card look (colour, icon, display title, content, read-only badge), server data outside edit mode and the notes query in edit mode, missing-note message, no fade with `auto` height; verify with Testing Library tests for each
- [x] 3.5 Add owner interactions: "Open note" link (inert in edit mode) and checkbox toggles saved via `useAutosave` with failure and conflict messages and query invalidation; no toggling in edit mode or for visitors; verify with Testing Library tests for toggle success, failure, conflict, edit mode and visitor
- [x] 3.6 Add all en + de strings (kind name and description, group headings, setting labels, public hint, missing note) and verify the i18n test passes

- [x] 3.7 Open the note in a modal instead of on `/notes/<id>`: make the note editor usable in a dialog (close button instead of the way back, no page title, deletion and saves reported to the caller), open it from the widget's "Open note" button, refresh the dashboard after each save and on deletion; verify with Testing Library tests that the modal opens on the dashboard, an edit is saved and shown on the widget, and a deletion closes the modal and shows the missing note

## 4. Check

- [x] 4.1 Run `make check` and `make test` and verify both pass
- [ ] 4.2 Run the app, add a note widget with a fixed height and size L, tick a todo, open the note in the modal, edit and delete it, and check the public dashboard with the flag off and on by hand
