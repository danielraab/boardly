# Tasks

## 1. Dashboard

- [x] 1.1 Add `note` to `DashboardSearch`/`validateDashboardSearch`, drop it in `setEditing`, add `useOpenNote()` and a `DashboardNoteDialog` in `DashboardView` (owner, outside edit mode; close via history back when pushed, else replace; 404 and visitors remove the parameter); let `NoteDialog` take its name from the loaded note; verify with tests for the search validation

## 2. Widgets

- [x] 2.1 Make the note, recent notes, note list and selected notes widgets open notes with `useOpenNote()` instead of their own `NoteDialog`; verify the existing modal tests of the four widgets still pass
- [x] 2.2 Add tests: opening puts `note` in the address, back closes the modal and leaves `/ann/home`, the close button does the same without an extra entry, reloading with `note` opens the modal and closing replaces the address, an unknown note and a visitor drop the parameter

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify both pass
