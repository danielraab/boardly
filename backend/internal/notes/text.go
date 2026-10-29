package notes

import (
	"strings"
)

// MaxFirstLine is how many characters of its first line an untitled note
// shows as its title.
const MaxFirstLine = 80

// Keep in sync with frontend/src/features/notes/document/text.ts.

// inlineText is the text of inline content, with line breaks as "\n".
func inlineText(content []Inline) string {
	var b strings.Builder
	for _, in := range content {
		switch in.Type {
		case InlineText:
			if in.Text != nil {
				b.WriteString(*in.Text)
			}
		case InlineLink:
			if in.Content != nil {
				for _, t := range *in.Content {
					if t.Text != nil {
						b.WriteString(*t.Text)
					}
				}
			}
		default:
			b.WriteString("\n")
		}
	}
	return b.String()
}

// blockLines returns the lines of text of a block, in reading order.
func blockLines(b Block) []string {
	switch b.Type {
	case BlockParagraph, BlockHeading, BlockQuote:
		if b.Content == nil {
			return nil
		}
		return strings.Split(inlineText(*b.Content), "\n")
	case BlockCode:
		if b.Text == nil {
			return nil
		}
		return strings.Split(*b.Text, "\n")
	case BlockList:
		if b.Items == nil {
			return nil
		}
		var lines []string
		for _, item := range *b.Items {
			lines = append(lines, strings.Split(inlineText(item.Content), "\n")...)
			if item.Children != nil {
				lines = append(lines, blockLines(*item.Children)...)
			}
		}
		return lines
	}
	return nil
}

// FirstLine returns the first non-empty line of text of a document, trimmed,
// or "".
func FirstLine(d Doc) string {
	for _, b := range d.Blocks {
		for _, line := range blockLines(b) {
			if t := strings.TrimSpace(line); t != "" {
				return t
			}
		}
	}
	return ""
}

// DisplayTitle returns the title shown for a note: its title, else its first
// line of text cut off after MaxFirstLine characters, else nil (shown as
// "Untitled note").
func DisplayTitle(n Note) *string {
	if n.Title != nil && *n.Title != "" {
		return n.Title
	}
	line := FirstLine(n.Content)
	if line == "" {
		return nil
	}
	if chars := []rune(line); len(chars) > MaxFirstLine {
		line = string(chars[:MaxFirstLine]) + "…"
	}
	return &line
}
