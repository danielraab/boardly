# Spec Delta

## ADDED Requirements

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
