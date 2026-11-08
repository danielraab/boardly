# Tasks

## 1. Save status

- [x] 1.1 Add the `compact` and `onReload` props to `SaveIndicator` and the `notes.editor.status.conflict` string ("Changed elsewhere" / "Anderswo geändert") in `en.json` and `de.json`; verify with tests that compact keeps the text as the accessible status, that conflict with `onReload` shows the label and a working "Reload" button, and that conflict without `onReload` renders nothing
- [x] 1.2 Add the `toolbarEnd` slot to `NoteEditor`, and in `NoteEditorPage` move `SaveIndicator` from the action row into it with `compact` and `onReload`; verify with a test in `NoteEditor.test.tsx` that the status is rendered once and next to the toolbar, and update existing tests that look up a single "Reload" button
- [x] 1.3 Add the `success` colour to every theme in `styles.css` and to `contrast.test.ts` (against `paper`, `surface` and the note colours), use it for the saved state in `SaveIndicator`; verify with the contrast test and a test that saved is `text-success` and failed `text-danger`

## 2. Toolbar layout

- [x] 2.1 Make the sticky wrapper a flex row and the `Toolbar` root a single, sideways-scrolling row with a hidden scrollbar, and keep the separator from shrinking; verify at 360 px in a browser that the toolbar is one row, scrolls to "Add location" and the status stays in place
- [x] 2.2 Add `bg-inherit` to the editor's wrapper in `NoteEditorPage`; verify in a browser with a coloured note and with no colour that scrolling text isn't visible behind the stuck toolbar, also in the dashboard note dialog

## 3. Keyboard

- [ ] 3.1 Add `interactive-widget=resizes-content` to the viewport meta tag in `frontend/index.html`; verify on Chrome on Android that the toolbar row stays visible while typing at the end of a long note, and that the mini player doesn't get in the way
- [ ] 3.2 Test typing at the end of a long note on iOS Safari; if the toolbar row leaves the view, update this change with a `visualViewport` offset before implementing one

## 4. Docs and checks

- [x] 4.1 Add a `CHANGELOG.md` entry, confirm that export, import and the OpenAPI documents need no change, and verify with `git diff --name-only` that only `frontend/`, `CHANGELOG.md` and OpenSpec paths changed
- [x] 4.2 Run `cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test` and verify it passes
- [x] 4.3 Run `openspec validate improve-mobile-note-editor --strict` and verify it passes
