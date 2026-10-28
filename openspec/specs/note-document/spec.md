# note-document Specification

## Purpose
Defines the content of a note: the stored, versioned JSON document format of its own, what the server accepts, stable IDs for task items, and how the content is written in the editor with Markdown shortcuts.

## Requirements

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

### Requirement: Content from Markdown
The server SHALL be able to build a content document from Markdown text, for imports and connected apps. CommonMark with GitHub's task lists and strikethrough SHALL map to the blocks of the document format: ATX and Setext headings of level 1–3 to headings (levels 4–6 become level 3), paragraphs, block quotes (their inline text; nested blocks are joined as paragraphs of text), fenced and indented code, thematic breaks to dividers, and bullet, ordered and task lists with their nesting (lists nested deeper than allowed are joined into the deepest allowed item). Emphasis, strong emphasis, strikethrough, code spans and links with an allowed URL scheme map to marks and links; autolinks and bare `http(s)` URLs become links. Every line break inside a paragraph, list item, quote or heading SHALL be kept as a line break. Anything else (images, tables, HTML, links with other URL schemes) SHALL be kept as its text: an image as its alt text, a table row by row with cells separated by ` | `, HTML as its literal source. Every task item gets a new ID, and its checked state is `[x]` or `[X]`. Empty Markdown gives an empty document.

#### Scenario: Headings and tasks
- **WHEN** the Markdown is `## Week` followed by `- [ ] review PR` and `- [x] deploy`
- **THEN** the document has a heading of level 2 "Week" and a task list with an unchecked "review PR" and a checked "deploy", each with an ID

#### Scenario: Line breaks kept
- **WHEN** the Markdown is "milk\neggs\nbread" without blank lines
- **THEN** the document has one paragraph with "milk", a line break, "eggs", a line break and "bread"

#### Scenario: Unsupported Markdown kept as text
- **WHEN** the Markdown contains `![logo](https://example.org/l.png)` and `<b>hi</b>`
- **THEN** the document contains the text "logo" and the text `<b>hi</b>`, and is a valid document

#### Scenario: Script link
- **WHEN** the Markdown contains `[click](javascript:alert(1))`
- **THEN** the document contains the text "click" without a link

### Requirement: Content from plain text
The server SHALL be able to build a content document from plain text, for text shared into the app. Text separated by one or more blank lines SHALL become separate paragraphs; single line breaks SHALL become line breaks inside a paragraph. `http` and `https` URLs in the text SHALL become links around the URL text. No other character has a special meaning, so `*`, `#` or `-` stay as they are. Leading and trailing blank lines are left out; empty text gives an empty document.

#### Scenario: Shared message
- **WHEN** the text is "Meet at 8\nBring snacks\n\nSee https://example.org/map"
- **THEN** the document has two paragraphs: "Meet at 8", a line break, "Bring snacks"; and "See " followed by a link to `https://example.org/map`

#### Scenario: Markdown characters stay
- **WHEN** the text is "# not a heading *really*"
- **THEN** the document has one paragraph with exactly that text

### Requirement: Content as Markdown
The server SHALL be able to write a content document as Markdown, for connected apps: headings as ATX headings (`#`, `##`, `###`), paragraphs separated by blank lines, quotes as `> ` lines, code as fenced code blocks (with a fence longer than any backtick run in the code), dividers as `---`, bullet lists as `- `, ordered lists as `1. `, `2. `, …, task items as `- [ ] ` and `- [x] `, nested lists indented below their item, marks as `**bold**`, `*italic*`, `~~strike~~` and `` `code` ``, links as `[text](url)`, or `<url>` when the text is the address, and line breaks as a single line break, an empty line as a line with only `\`. Lists directly after each other of the same kind SHALL use another bullet (`*`) or delimiter (`1)`), so they stay separate. Characters in text that would otherwise be read as Markdown, including addresses that would become links, SHALL be escaped with a backslash. Reading the Markdown back (see "Content from Markdown") SHALL give the same document apart from task IDs and what Markdown can't express: empty paragraphs are left out, line breaks in headings become spaces, spaces at the start and end of a line and line breaks at the end of a block are dropped, and a mark may be lost where Markdown can't express it for sure: when it starts or ends inside a word, between punctuation and a letter or digit, or right where another mark ends. Such a mark is left out rather than written as delimiters that would show as text.

#### Scenario: Task list
- **WHEN** a note has a heading "Week" of level 2 and a task list with a checked "deploy"
- **THEN** its Markdown is `## Week`, a blank line and `- [x] deploy`

#### Scenario: Round trip
- **WHEN** a valid document is written as Markdown and read back
- **THEN** the result equals the document, apart from task IDs and the listed exceptions

#### Scenario: Literal asterisks
- **WHEN** a paragraph has the plain text "2 * 3 * 4"
- **THEN** the Markdown escapes the asterisks, and reading it back gives the text "2 * 3 * 4" without italics

#### Scenario: Empty paragraph
- **WHEN** a document has an empty paragraph between two paragraphs
- **THEN** its Markdown has the two paragraphs separated by one blank line

#### Scenario: Adjacent lists
- **WHEN** a bullet list is directly followed by a task list
- **THEN** the Markdown starts the task list with `*`, and reading it back gives two lists

### Requirement: Content replaced from Markdown
When a note's content is replaced by Markdown from a connected app, the server SHALL read the Markdown as in "Content from Markdown" and then give each task item of the new document the ID of a task item of the old document with the same text (its inline content as plain text, trimmed), matching in document order, each old task at most once. A task item without a match gets a new ID. The checked state always comes from the Markdown.

#### Scenario: Reordered tasks
- **WHEN** a note has the tasks "a" and "b" and the new Markdown lists "b" before "a"
- **THEN** both tasks keep their IDs

#### Scenario: Renamed task
- **WHEN** a note has the task "milk" and the new Markdown has the task "oat milk" instead
- **THEN** "oat milk" gets a new ID

#### Scenario: Same text twice
- **WHEN** a note has two tasks "call" and the new Markdown has three tasks "call"
- **THEN** the first two keep the old IDs in order, and the third gets a new ID
