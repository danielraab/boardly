## MODIFIED Requirements

### Requirement: Document format
A note's content SHALL be a JSON document `{"v": 1, "blocks": [...]}`, not Markdown and not the editor's internal format. The blocks are:
- `paragraph` and `heading` (level 1, 2 or 3), with inline content;
- `quote`, with inline content;
- `code`, with plain text;
- `divider`, without content;
- `list` with the style `bullet`, `ordered` or `task`, whose items have inline content and optionally one nested list. Items of a `task` list additionally have an ID and a `checked` state;
- `attachment`, with only the `id` of an attachment of the note's owner (see the `note-attachments` capability). It has no text: the display title and other plain text of a note skip it.

Inline content is a sequence of text runs, each with any of the marks bold, italic, strikethrough and code, links (an `http`, `https` or `mailto` URL around text runs) and line breaks. Every block, item and inline node type SHALL have a Markdown equivalent, so that the content can be converted to Markdown without losing structure. Adding the `attachment` block doesn't change the version, which stays `1`.

#### Scenario: Stored content
- **WHEN** Ann types `## This week`, then `- [ ] review PR`, and the note is saved
- **THEN** `GET /api/notes/{id}` returns `content` with `v: 1`, a heading block of level 2 "This week", and a task list with one unchecked item "review PR" that has an ID

#### Scenario: Stored attachment
- **WHEN** Ann adds `report.pdf` below a paragraph and the note is saved
- **THEN** the content has `v: 1`, the paragraph and then `{"type": "attachment", "id": "<id of report.pdf>"}`

### Requirement: Document validation
The server SHALL reject a content document with 400 `invalid_document` when it isn't valid JSON of the format above, has a version other than 1, contains an unknown block, inline node or mark type, an unknown field, a heading level outside 1–3, a link with another URL scheme, an attachment block whose ID isn't an existing attachment of the note's owner, lists nested more than 6 levels deep, or is larger than 512 KB in its JSON form. Attachments marked for deletion exist and are accepted.

#### Scenario: Unknown block type
- **WHEN** a note is saved whose content contains a block of type `iframe`
- **THEN** the server responds 400 with `invalid_document`

#### Scenario: Script link
- **WHEN** a note is saved with a link to `javascript:alert(1)`
- **THEN** the server responds 400 with `invalid_document`

#### Scenario: Too large
- **WHEN** a note is saved whose content is 600 KB of JSON
- **THEN** the server responds 400 with `invalid_document`

#### Scenario: Other user's attachment
- **WHEN** Ann saves a note whose content contains an attachment block with the ID of one of Bob's attachments
- **THEN** the server responds 400 with `invalid_document`, and Bob's attachment gets no reference

#### Scenario: Unknown attachment
- **WHEN** a note is saved with an attachment block whose ID doesn't exist
- **THEN** the server responds 400 with `invalid_document`

### Requirement: Content from Markdown
The server SHALL be able to build a content document from Markdown text, for imports and connected apps. CommonMark with GitHub's task lists and strikethrough SHALL map to the blocks of the document format: ATX and Setext headings of level 1–3 to headings (levels 4–6 become level 3), paragraphs, block quotes (their inline text; nested blocks are joined as paragraphs of text), fenced and indented code, thematic breaks to dividers, and bullet, ordered and task lists with their nesting (lists nested deeper than allowed are joined into the deepest allowed item). Emphasis, strong emphasis, strikethrough, code spans and links with an allowed URL scheme map to marks and links; autolinks and bare `http(s)` URLs become links. Every line break inside a paragraph, list item, quote or heading SHALL be kept as a line break.

An image whose URL is the URL of an existing attachment of the note's owner, as `<BOARDLY_BASE_URL>/api/attachments/<id>/<name>` or as the path `/api/attachments/<id>/<name>`, SHALL become an attachment block, splitting its paragraph when there is other content around it. A link with such a URL SHALL become an attachment block when it is the only content of its paragraph, and stays a link otherwise. Any other attachment URL, for example of another user's attachment, is treated like any other URL.

Anything else (other images, tables, HTML, links with other URL schemes) SHALL be kept as its text: an image as its alt text, a table row by row with cells separated by ` | `, HTML as its literal source. Every task item gets a new ID, and its checked state is `[x]` or `[X]`. Empty Markdown gives an empty document.

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

#### Scenario: Own attachment
- **WHEN** Ann's Markdown contains a paragraph `![roses.jpg](<BOARDLY_BASE_URL>/api/attachments/<id>/roses.jpg)` with the ID of her attachment
- **THEN** the document has an attachment block with that ID

#### Scenario: File link alone in its paragraph
- **WHEN** Ann's Markdown has the paragraph `[report.pdf](/api/attachments/<id>/report.pdf)` with the ID of her attachment
- **THEN** the document has an attachment block with that ID

#### Scenario: Other user's attachment
- **WHEN** Ann's Markdown contains `![x](<BOARDLY_BASE_URL>/api/attachments/<id>/x.png)` with the ID of one of Bob's attachments
- **THEN** the document contains the text "x" and no attachment block

### Requirement: Content as Markdown
The server SHALL be able to write a content document as Markdown, for connected apps: headings as ATX headings (`#`, `##`, `###`), paragraphs separated by blank lines, quotes as `> ` lines, code as fenced code blocks (with a fence longer than any backtick run in the code), dividers as `---`, bullet lists as `- `, ordered lists as `1. `, `2. `, …, task items as `- [ ] ` and `- [x] `, nested lists indented below their item, marks as `**bold**`, `*italic*`, `~~strike~~` and `` `code` ``, links as `[text](url)`, or `<url>` when the text is the address, and line breaks as a single line break, an empty line as a line with only `\`. An attachment SHALL be written as a paragraph of its own: `![<name>](<url>)` for an image and `[<name>](<url>)` for any other file, where `<url>` is `<BOARDLY_BASE_URL>/api/attachments/<id>/<name>` with the name escaped for a URL path and `<name>` is escaped like text. Lists directly after each other of the same kind SHALL use another bullet (`*`) or delimiter (`1)`), so they stay separate. Characters in text that would otherwise be read as Markdown, including addresses that would become links, SHALL be escaped with a backslash. Reading the Markdown back (see "Content from Markdown") SHALL give the same document apart from task IDs and what Markdown can't express: empty paragraphs are left out, line breaks in headings become spaces, spaces at the start and end of a line and line breaks at the end of a block are dropped, and a mark may be lost where Markdown can't express it for sure: when it starts or ends inside a word, between punctuation and a letter or digit, or right where another mark ends. Such a mark is left out rather than written as delimiters that would show as text.

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

#### Scenario: Attachments
- **WHEN** a note contains the image `roses.jpg` and the file `my report.pdf`
- **THEN** its Markdown has the paragraphs `![roses.jpg](<BOARDLY_BASE_URL>/api/attachments/<id>/roses.jpg)` and `[my report.pdf](<BOARDLY_BASE_URL>/api/attachments/<id>/my%20report.pdf)`, and reading it back gives the two attachment blocks
