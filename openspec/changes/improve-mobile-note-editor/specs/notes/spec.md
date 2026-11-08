# Spec Delta

## ADDED Requirements

### Requirement: Save status stays visible
The editor page and the note editor modal of dashboards SHALL show the save status described in "Autosave" at the end of the sticky toolbar row of the content editor, and only there, so that it stays visible while the content is scrolled. On a viewport narrower than 640 px, "Saving…" / "Wird gespeichert…" and "Saved" / "Gespeichert" SHALL be shown as an icon that keeps the text as its accessible name; from 640 px on they SHALL be shown as text. "Not saved" / "Nicht gespeichert" and its "Retry" / "Erneut versuchen" button SHALL be shown as text at every width. After a `version_conflict`, the status SHALL show "Changed elsewhere" / "Anderswo geändert" with a "Reload" / "Neu laden" button, in addition to the explanation at the top of the editor. "Saved" / "Gespeichert" and its icon SHALL use the theme colour `success`, and "Not saved" / "Nicht gespeichert" and "Changed elsewhere" / "Anderswo geändert" SHALL use `danger`, in the editor and on cards. As soon as the note is changed again after a save, "Saved" / "Gespeichert" SHALL no longer be shown, until the next save has succeeded. The status of inline editing on cards SHALL stay on the card. On a phone whose browser supports resizing the page for the on-screen keyboard, the sticky toolbar row SHALL stay in view while the keyboard is open.

#### Scenario: Saved on a long note
- **WHEN** Ann has scrolled to the end of a long note on a phone and types a word
- **THEN** about a second later the toolbar row, still in view, shows the saved icon, and a screen reader announces "Saved"

#### Scenario: Status colours
- **WHEN** a save succeeds, and a later save fails
- **THEN** the status is first shown in the `success` colour, then "Not saved" in the `danger` colour

#### Scenario: Changed again after a save
- **WHEN** the status shows "Saved" and Ann types another letter
- **THEN** "Saved" disappears at once, and shows again after the next save has succeeded

#### Scenario: Status as text on a wider viewport
- **WHEN** Ann changes a note in a viewport 1024 px wide
- **THEN** the toolbar row shows "Saved" as text, and the row above the title doesn't show a save status

#### Scenario: Failed save while scrolled down
- **WHEN** a save fails while Ann is at the end of a long note on a phone
- **THEN** the toolbar row shows "Not saved" and "Retry", and tapping "Retry" saves the note again

#### Scenario: Conflict while scrolled down
- **WHEN** a save gets `version_conflict` while Ann is at the end of a long note
- **THEN** the toolbar row shows "Changed elsewhere" and "Reload", and "Reload" loads the current note

#### Scenario: Note editor modal
- **WHEN** Ann changes a long note in the note editor modal of a dashboard and scrolls to its end
- **THEN** the toolbar row at the top of the modal shows the save status

#### Scenario: Inline editing on a card
- **WHEN** Ann edits a card on `/notes` inline
- **THEN** the save status is shown on the card as before

#### Scenario: Typing with the keyboard open
- **WHEN** Ann types at the end of a long note in Chrome on Android, with the on-screen keyboard open
- **THEN** the toolbar row with the save status stays visible above the text being typed
