# Tasks

## 1. Backend

- [x] 1.1 Add `color` to the `recentNote` and `listedNote` rows, filled from the note, for owner and visitors; verify with updated tests in `recentnotes_test.go`, `notelist_test.go` and `selectednotes_test.go` that the colour is in the data and nothing else hidden appears

## 2. Frontend

- [x] 2.1 Add `color` to the `NoteRow`, `ListedNote` and recent note types and render each row in `NoteRows` as a small card (border, rounded corners, note colour or `surface`, gap between rows, hover without changing the background); verify with tests in the three widgets' test files that a coloured row has the note background and an uncoloured one `bg-surface`, and update preview helpers (`recent.ts`, `list.ts`, `selected.ts`) and their tests to pass the colour

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify both pass
