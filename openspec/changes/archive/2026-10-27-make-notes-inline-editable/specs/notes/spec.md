## MODIFIED Requirements

### Requirement: Notes page
`/notes` SHALL require a signed-in user and show the heading "Notes" / "Notizen", a "New note" / "Neue Notiz" button, the category side panel (see the `note-categories` capability) and the user's notes as a grid of cards, ordered by time of last change, newest first. The number of columns SHALL follow the available width, with one column at phone width. The grid SHALL show only the notes of the category chosen in the side panel. Each card's "Open note" / "Notiz öffnen" button SHALL open `/notes/<id>`. Signed-out visitors SHALL be sent to `/login`. The document title is "Notes - <app title>".

#### Scenario: Open a note
- **WHEN** Ann activates "Open note" on the card "Groceries" on `/notes`
- **THEN** the browser shows `/notes/<id of Groceries>`

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/notes`
- **THEN** the browser shows `/login`

#### Scenario: Phone width
- **WHEN** `/notes` is viewed 375 px wide
- **THEN** the cards are shown in one column

### Requirement: Note cards
A note card SHALL show the note's icon, if any, and its title, if any, followed by its rendered content, with task items shown as checkboxes in their state, and an "Open note" / "Notiz öffnen" button. Content that doesn't fit into the card's maximum height is cut off and fades out. A note with a colour SHALL have that colour as the card's background; otherwise the card uses `surface`. A note with a category SHALL show the category badge (see "Category badge" in the `note-categories` capability) at the bottom of the card. A checkbox on the card SHALL toggle its task item with a click, a tap or the keyboard; the change is saved right away and shown at once. Links in the card's content SHALL NOT open; activating them starts inline editing like the rest of the content (see "Inline editing on cards"). When saving a toggled checkbox fails, the card SHALL show the note as it is on the server again and a message SHALL say that the change couldn't be saved, or that the note was changed elsewhere when the save got `version_conflict`.

#### Scenario: Card with colour and category
- **WHEN** "Groceries" has the colour `green`, the symbol icon "cart" and the category "Home"
- **THEN** its card has the green note background, shows the cart symbol and "Groceries", its content, the "Open note" button and the badge "Home"

#### Scenario: Long note
- **WHEN** a note has 60 lines of content
- **THEN** its card shows the first lines and fades out at its maximum height

#### Scenario: Untitled card
- **WHEN** a note has no title
- **THEN** its card shows no title line and starts with its content

#### Scenario: Tick off a todo on the card
- **WHEN** Ann clicks the checkbox of the unchecked task "milk" on the card "Groceries"
- **THEN** "milk" is shown checked and struck through, the browser stays on `/notes`, and the note's task item has `checked: true`

#### Scenario: Toggle fails
- **WHEN** Ann toggles a checkbox on a card while the server can't be reached
- **THEN** the checkbox returns to its previous state and a message says the change couldn't be saved

## ADDED Requirements

### Requirement: Inline editing on cards
Activating the title or content of a card on `/notes` with a click, a tap, or Enter when the card has focus SHALL turn the card into an inline editor without leaving `/notes`. The inline editor SHALL show a title field with the placeholder "Title" / "Titel" and the content editor with its Markdown shortcuts, keyboard shortcuts and task checkboxes (see "Content editor" and "Task items in the editor" in the `note-document` capability) but without the toolbar, and SHALL focus the title field when the title was activated, and the end of the content otherwise. While editing, the card SHALL grow to show all of its content instead of fading out. Only one card SHALL be in inline editing at a time. Escape, activating another card, or moving the focus out of the card SHALL end inline editing and show the card again. Changes SHALL be saved as described in "Autosave": 1 second after the last change and immediately when inline editing ends or the page is left, with the save status shown on the card. When a save gets `version_conflict`, inline editing SHALL end, the card SHALL show the current note, and a message SHALL say that the note was changed elsewhere.

Notes changed on `/notes`, whether by inline editing, a checkbox or the category badge, SHALL keep their place in the grid while the page stays open; the order by time of last change applies again the next time `/notes` is opened.

#### Scenario: Edit text in place
- **WHEN** Ann clicks the content of the card "Groceries", types "eggs" at the end and presses Escape
- **THEN** the browser still shows `/notes`, the card shows "eggs", and the saved note contains "eggs"

#### Scenario: Edit the title in place
- **WHEN** Ann clicks the content of an untitled card and types "Shopping" into the title field
- **THEN** after the autosave the note has the title "Shopping", and the card shows it once editing ends

#### Scenario: Markdown shortcut inline
- **WHEN** Ann edits a card inline and types `- [ ] bread` on an empty line
- **THEN** the line becomes an unchecked task item "bread"

#### Scenario: No toolbar
- **WHEN** Ann edits a card inline
- **THEN** the card shows no formatting toolbar, and Ctrl/Cmd+B still makes the selected text bold

#### Scenario: Card keeps its place
- **WHEN** "Groceries" is the third card and Ann edits it inline and ends editing
- **THEN** "Groceries" is still the third card, and after reopening `/notes` it is the first

#### Scenario: Changed elsewhere
- **WHEN** Ann edits "Groceries" inline while it was changed in another tab
- **THEN** inline editing ends, the card shows the other tab's version, and a message says the note was changed elsewhere
