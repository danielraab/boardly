## MODIFIED Requirements

### Requirement: Content editor
The editor page SHALL edit the content as formatted text, not as Markdown source. It SHALL turn these inputs into formatting while typing:
- `# `, `## `, `### ` at the start of a line → heading of level 1, 2, 3
- `- ` or `* ` → bullet list; `1. ` → ordered list; `- [ ] ` or `[] ` → task list with an unchecked item; `- [x] ` → task list with a checked item
- `> ` → quote; ```` ``` ```` → code block; `---` → divider
- `**text**` → bold, `*text*` or `_text_` → italic, `~~text~~` → strikethrough, `` `text` `` → code, `[text](url)` → link

It SHALL also offer a toolbar with heading, bullet list, ordered list, task list, bold, italic, strikethrough, code, link and "Add attachment" (see "Adding attachments" in the `note-attachments` capability), the keyboard shortcuts Ctrl/Cmd+B, Ctrl/Cmd+I, Ctrl/Cmd+Z and Ctrl/Cmd+Shift+Z, and Tab / Shift+Tab to indent and outdent list items. Enter in a list item starts a new item of the same kind, unchecked for task items; Enter in an empty item ends the list. When the content is empty, the editor shows the placeholder "Write something…" / "Schreib etwas…".

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

#### Scenario: Attachment button in the toolbar
- **WHEN** Ann opens the editor page of "Garden"
- **THEN** the toolbar shows "Add attachment" right after "Link"
