# Spec Delta

## Purpose

Defines the content of a note: the stored, versioned JSON document format of its own, what the server accepts, stable IDs for task items, and how the content is written in the editor with Markdown shortcuts.

## ADDED Requirements

### Requirement: Document format
A note's content SHALL be a JSON document `{"v": 1, "blocks": [...]}`, not Markdown and not the editor's internal format. The blocks are:
- `paragraph` and `heading` (level 1, 2 or 3), with inline content;
- `quote`, with inline content;
- `code`, with plain text;
- `divider`, without content;
- `list` with the style `bullet`, `ordered` or `task`, whose items have inline content and optionally one nested list. Items of a `task` list additionally have an ID and a `checked` state.

Inline content is a sequence of text runs, each with any of the marks bold, italic, strikethrough and code, links (an `http`, `https` or `mailto` URL around text runs) and line breaks. Every block, item and inline node type SHALL have a Markdown equivalent, so that the content can be converted to Markdown without losing structure.

#### Scenario: Stored content
- **WHEN** Ann types `## This week`, then `- [ ] review PR`, and the note is saved
- **THEN** `GET /api/notes/{id}` returns `content` with `v: 1`, a heading block of level 2 "This week", and a task list with one unchecked item "review PR" that has an ID

### Requirement: Document validation
The server SHALL reject a content document with 400 `invalid_document` when it isn't valid JSON of the format above, has a version other than 1, contains an unknown block, inline node or mark type, an unknown field, a heading level outside 1–3, a link with another URL scheme, lists nested more than 6 levels deep, or is larger than 512 KB in its JSON form.

#### Scenario: Unknown block type
- **WHEN** a note is saved whose content contains a block of type `iframe`
- **THEN** the server responds 400 with `invalid_document`

#### Scenario: Script link
- **WHEN** a note is saved with a link to `javascript:alert(1)`
- **THEN** the server responds 400 with `invalid_document`

#### Scenario: Too large
- **WHEN** a note is saved whose content is 600 KB of JSON
- **THEN** the server responds 400 with `invalid_document`

### Requirement: Stable task IDs
Every task item SHALL have an ID of 1 to 32 letters, digits, `-` or `_` that is unique within its note. The server SHALL give a task item without an ID a new one, and SHALL replace the ID of a task item whose ID already appeared earlier in the same note. A task item SHALL keep its ID across saves for as long as it exists, also when its text, its checked state or its position changes.

#### Scenario: ID assigned
- **WHEN** a note is saved with a task item without an ID
- **THEN** the saved note's task item has an ID

#### Scenario: ID kept on edit
- **WHEN** Ann changes the text of the task "review PR" to "review PR #12" and checks it
- **THEN** the task item has the same ID as before

#### Scenario: Duplicated ID
- **WHEN** a note is saved with two task items with the ID `a1`
- **THEN** the first keeps `a1` and the second gets a different ID

### Requirement: Content editor
The editor page SHALL edit the content as formatted text, not as Markdown source. It SHALL turn these inputs into formatting while typing:
- `# `, `## `, `### ` at the start of a line → heading of level 1, 2, 3
- `- ` or `* ` → bullet list; `1. ` → ordered list; `- [ ] ` or `[] ` → task list with an unchecked item; `- [x] ` → task list with a checked item
- `> ` → quote; ```` ``` ```` → code block; `---` → divider
- `**text**` → bold, `*text*` or `_text_` → italic, `~~text~~` → strikethrough, `` `text` `` → code, `[text](url)` → link

It SHALL also offer a toolbar with heading, bullet list, ordered list, task list, bold, italic, strikethrough, code and link, the keyboard shortcuts Ctrl/Cmd+B, Ctrl/Cmd+I, Ctrl/Cmd+Z and Ctrl/Cmd+Shift+Z, and Tab / Shift+Tab to indent and outdent list items. Enter in a list item starts a new item of the same kind, unchecked for task items; Enter in an empty item ends the list. When the content is empty, the editor shows the placeholder "Write something…" / "Schreib etwas…".

#### Scenario: Task shortcut
- **WHEN** Ann types `- [ ] milk` on an empty line
- **THEN** the line becomes an unchecked task item "milk"

#### Scenario: Next task
- **WHEN** Ann presses Enter at the end of the task item "milk" and types "eggs"
- **THEN** "eggs" is a new unchecked task item below "milk"

#### Scenario: Bold
- **WHEN** Ann types `**urgent**`
- **THEN** "urgent" is shown bold without the asterisks

#### Scenario: Toolbar on a phone
- **WHEN** Ann selects a line on a phone and taps "Task list" in the toolbar
- **THEN** the line becomes an unchecked task item

### Requirement: Task items in the editor
Each task item SHALL be shown with a checkbox that can be toggled with a click, a tap or the keyboard. Checked items SHALL be shown struck through in `muted`. Toggling counts as a change and is saved automatically.

#### Scenario: Check a task
- **WHEN** Ann clicks the checkbox of "milk"
- **THEN** "milk" is shown checked and struck through, and after the autosave the note's task item has `checked: true`

### Requirement: Links in the content
In the editor, a plain click on a link SHALL place the cursor, and a click with Ctrl/Cmd held SHALL open the link's URL in a new tab. Links on the cards of `/notes` don't open (see "Note cards" in the `notes` capability). The editor SHALL only accept `http`, `https` and `mailto` URLs for links.

#### Scenario: Open a link
- **WHEN** Ann Ctrl-clicks the link "docs" in the editor
- **THEN** its URL opens in a new tab

#### Scenario: Invalid link
- **WHEN** Ann tries to create a link to `javascript:alert(1)` from the toolbar
- **THEN** the editor shows a translated validation message and creates no link
