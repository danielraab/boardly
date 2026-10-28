# Spec Delta

## ADDED Requirements

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
