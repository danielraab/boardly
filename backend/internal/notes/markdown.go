package notes

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// maxLinkURLLength is the longest link URL a document accepts.
const maxLinkURLLength = 2048

var markdown = goldmark.New(goldmark.WithExtensions(
	extension.TaskList, extension.Strikethrough, extension.Linkify, extension.Table,
))

// FromMarkdown builds a document from Markdown (CommonMark with GitHub's task
// lists, strikethrough, tables and bare URLs). Whatever the document can't
// hold is kept as text, and line breaks inside a paragraph stay line breaks.
// Task items have no IDs yet; ParseDoc assigns them. Images, and links alone
// in their paragraph, whose URL is that of one of att's attachments become
// attachment blocks; att may be nil.
func FromMarkdown(src string, att *MarkdownAttachments) Doc {
	source := []byte(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(src))
	root := markdown.Parser().Parse(text.NewReader(source))
	c := converter{src: source, att: att}
	d := EmptyDoc()
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if p, ok := n.(*ast.Paragraph); ok && att != nil {
			d.Blocks = append(d.Blocks, c.paragraph(p)...)
		} else if b, ok := c.block(n); ok {
			d.Blocks = append(d.Blocks, b)
		}
	}
	for i := range d.Blocks {
		mergeBlockRuns(&d.Blocks[i])
	}
	return d
}

// mergeBlockRuns joins the text runs that goldmark splits, in b and its
// list items.
func mergeBlockRuns(b *Block) {
	if b.Content != nil {
		merged := mergeRuns(*b.Content)
		b.Content = &merged
	}
	if b.Items != nil {
		for i := range *b.Items {
			it := &(*b.Items)[i]
			it.Content = mergeRuns(it.Content)
			if it.Children != nil {
				mergeBlockRuns(it.Children)
			}
		}
	}
}

type converter struct {
	src []byte
	att *MarkdownAttachments
}

// paragraph converts a top-level paragraph that may hold attachments: a link
// to one alone in the paragraph, or images of them, which split the
// paragraph around them.
func (c converter) paragraph(p *ast.Paragraph) []Block {
	if l, ok := p.FirstChild().(*ast.Link); ok && l.NextSibling() == nil {
		if id := c.att.id(c.destination(l.Destination)); id != "" {
			return []Block{{Type: BlockAttachment, ID: id}}
		}
	}
	var (
		blocks []Block
		cur    = []Inline{}
		split  bool
	)
	flush := func() {
		for len(cur) > 0 && cur[0].Type == InlineBreak {
			cur = cur[1:]
		}
		for len(cur) > 0 && cur[len(cur)-1].Type == InlineBreak {
			cur = cur[:len(cur)-1]
		}
		if len(cur) > 0 {
			content := cur
			blocks = append(blocks, Block{Type: BlockParagraph, Content: &content})
		}
		cur = []Inline{}
	}
	for child := p.FirstChild(); child != nil; child = child.NextSibling() {
		if img, ok := child.(*ast.Image); ok {
			if id := c.att.id(c.destination(img.Destination)); id != "" {
				flush()
				blocks = append(blocks, Block{Type: BlockAttachment, ID: id})
				split = true
				continue
			}
		}
		cur = append(cur, c.inline(child, nil)...)
	}
	if !split {
		return []Block{{Type: BlockParagraph, Content: &cur}}
	}
	flush()
	return blocks
}

// destination returns a link or image address without Markdown escapes and
// entities, which goldmark keeps.
func (c converter) destination(dest []byte) string {
	return string(util.UnescapePunctuations(util.ResolveNumericReferences(util.ResolveEntityNames(dest))))
}

// block converts a top-level block node, or reports false to leave it out.
func (c converter) block(n ast.Node) (Block, bool) {
	switch n := n.(type) {
	case *ast.Heading:
		content := c.inlines(n, nil)
		return Block{Type: BlockHeading, Level: min(n.Level, 3), Content: &content}, true
	case *ast.Paragraph, *ast.TextBlock:
		content := c.inlines(n, nil)
		return Block{Type: BlockParagraph, Content: &content}, true
	case *ast.Blockquote:
		content := c.flatten(n)
		return Block{Type: BlockQuote, Content: &content}, true
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		code := strings.TrimSuffix(c.lines(n), "\n")
		return Block{Type: BlockCode, Text: &code}, true
	case *ast.ThematicBreak:
		return Block{Type: BlockDivider}, true
	case *ast.List:
		return c.list(n, 1), true
	default:
		content := c.flatten(n)
		if len(content) == 0 {
			return Block{}, false
		}
		return Block{Type: BlockParagraph, Content: &content}, true
	}
}

// list converts a list at depth (1 for a top-level list). A list with at
// least one checkbox is a task list. Lists deeper than MaxListDepth, and a
// second nested list of an item, are joined into the item's text.
func (c converter) list(l *ast.List, depth int) Block {
	style := ListBullet
	if l.IsOrdered() {
		style = ListOrdered
	}
	for it := l.FirstChild(); it != nil; it = it.NextSibling() {
		if checkbox(it) != nil {
			style = ListTask
			break
		}
	}
	items := []Item{}
	for it := l.FirstChild(); it != nil; it = it.NextSibling() {
		item := Item{Content: []Inline{}}
		if style == ListTask {
			checked := false
			if cb := checkbox(it); cb != nil {
				checked = cb.IsChecked
			}
			item.Checked = &checked
		}
		first := true
		for child := it.FirstChild(); child != nil; child = child.NextSibling() {
			if sub, ok := child.(*ast.List); ok && item.Children == nil && depth < MaxListDepth {
				children := c.list(sub, depth+1)
				item.Children = &children
				continue
			}
			var part []Inline
			switch child.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				part = c.inlines(child, nil)
			default:
				part = c.flatten(child)
			}
			if first {
				item.Content = trimLeft(part)
				first = false
			} else {
				item.Content = joinInlines(item.Content, part)
			}
		}
		items = append(items, item)
	}
	return Block{Type: BlockList, Style: style, Items: &items}
}

// checkbox returns the task checkbox at the start of a list item, or nil.
func checkbox(item ast.Node) *extast.TaskCheckBox {
	if first := item.FirstChild(); first != nil {
		if cb, ok := first.FirstChild().(*extast.TaskCheckBox); ok {
			return cb
		}
	}
	return nil
}

// flatten returns any block node as inline content, its parts separated by
// line breaks.
func (c converter) flatten(n ast.Node) []Inline {
	switch n := n.(type) {
	case *ast.Paragraph, *ast.TextBlock, *ast.Heading:
		return c.inlines(n, nil)
	case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
		var lines string
		if h, ok := n.(*ast.HTMLBlock); ok {
			lines = c.lines(h)
			if h.HasClosure() {
				lines += string(h.ClosureLine.Value(c.src))
			}
		} else {
			lines = c.lines(n)
		}
		return textLines(strings.TrimRight(lines, "\n"), nil)
	case *ast.ThematicBreak:
		return nil
	case *extast.TableRow, *extast.TableHeader:
		var out []Inline
		for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if cell != n.FirstChild() {
				out = append(out, textRun(" | ", nil))
			}
			out = append(out, c.inlines(cell, nil)...)
		}
		return out
	}
	var out []Inline
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		out = joinInlines(out, c.flatten(child))
	}
	return out
}

// inlines converts the inline children of n with the marks.
func (c converter) inlines(n ast.Node, marks []string) []Inline {
	out := []Inline{}
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		out = append(out, c.inline(child, marks)...)
	}
	return out
}

// inline converts one inline node with the marks.
func (c converter) inline(child ast.Node, marks []string) []Inline {
	var out []Inline
	switch child := child.(type) {
	case *ast.Text:
		v := child.Value(c.src)
		if !child.IsRaw() {
			v = util.UnescapePunctuations(util.ResolveNumericReferences(util.ResolveEntityNames(v)))
		}
		if len(v) > 0 {
			out = append(out, textRun(string(v), marks))
		}
		if child.SoftLineBreak() || child.HardLineBreak() {
			out = append(out, Inline{Type: InlineBreak})
		}
	case *ast.String:
		out = append(out, textRun(string(child.Value), marks))
	case *ast.CodeSpan:
		var b strings.Builder
		for t := child.FirstChild(); t != nil; t = t.NextSibling() {
			if s, ok := t.(*ast.Text); ok {
				b.Write(s.Value(c.src))
			} else if s, ok := t.(*ast.String); ok {
				b.Write(s.Value)
			}
		}
		out = append(out, textRun(strings.ReplaceAll(b.String(), "\n", " "), withMark(marks, "code")))
	case *ast.Emphasis:
		mark := "italic"
		if child.Level >= 2 {
			mark = "bold"
		}
		out = append(out, c.inlines(child, withMark(marks, mark))...)
	case *extast.Strikethrough:
		out = append(out, c.inlines(child, withMark(marks, "strike"))...)
	case *ast.Link:
		out = append(out, link(c.destination(child.Destination), c.inlines(child, marks))...)
	case *ast.AutoLink:
		u := string(child.URL(c.src))
		if child.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(u), "mailto:") {
			u = "mailto:" + u
		}
		out = append(out, link(u, []Inline{textRun(string(child.Label(c.src)), marks)})...)
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < child.Segments.Len(); i++ {
			s := child.Segments.At(i)
			b.Write(s.Value(c.src))
		}
		out = append(out, textLines(b.String(), marks)...)
	case *extast.TaskCheckBox:
	default:
		// Images (their alt text) and anything else: the inline content.
		out = append(out, c.inlines(child, marks)...)
	}
	return out
}

// mergeRuns joins adjacent text runs with the same marks, also inside links.
func mergeRuns(content []Inline) []Inline {
	out := []Inline{}
	for _, in := range content {
		if in.Type == InlineLink {
			merged := mergeRuns(*in.Content)
			in.Content = &merged
		}
		if n := len(out); n > 0 && in.Type == InlineText && out[n-1].Type == InlineText && slices.Equal(out[n-1].Marks, in.Marks) {
			joined := *out[n-1].Text + *in.Text
			out[n-1].Text = &joined
			continue
		}
		out = append(out, in)
	}
	return out
}

// lines returns the raw lines of a block.
func (c converter) lines(n ast.Node) string {
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		s := n.Lines().At(i)
		b.Write(s.Value(c.src))
	}
	return b.String()
}

// link returns content as a link to u, or just content when the document
// doesn't accept u. A link holds text runs only, so line breaks become spaces.
func link(u string, content []Inline) []Inline {
	parsed, err := url.Parse(u)
	if err != nil || !linkSchemes[strings.ToLower(parsed.Scheme)] || len(u) > maxLinkURLLength {
		return content
	}
	inner := []Inline{}
	for _, in := range content {
		switch in.Type {
		case InlineText:
			inner = append(inner, in)
		case InlineBreak:
			inner = append(inner, textRun(" ", nil))
		case InlineLink:
			inner = append(inner, *in.Content...)
		}
	}
	return []Inline{{Type: InlineLink, URL: u, Content: &inner}}
}

func textRun(s string, marks []string) Inline {
	return Inline{Type: InlineText, Text: &s, Marks: slices.Clone(marks)}
}

// textLines returns s as text runs separated by line breaks.
func textLines(s string, marks []string) []Inline {
	var out []Inline
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			out = append(out, Inline{Type: InlineBreak})
		}
		if line != "" {
			out = append(out, textRun(line, marks))
		}
	}
	return out
}

func withMark(marks []string, mark string) []string {
	if slices.Contains(marks, mark) {
		return marks
	}
	return append(slices.Clone(marks), mark)
}

// joinInlines appends b to a, with a line break between them when both have
// content.
func joinInlines(a, b []Inline) []Inline {
	if len(b) == 0 {
		return a
	}
	if len(a) > 0 {
		a = append(a, Inline{Type: InlineBreak})
	}
	return append(a, b...)
}

// trimLeft removes leading spaces of the first text run, such as the one
// after a task checkbox.
func trimLeft(content []Inline) []Inline {
	if len(content) > 0 && content[0].Type == InlineText {
		t := strings.TrimLeft(*content[0].Text, " \t")
		if t == "" {
			return content[1:]
		}
		content[0].Text = &t
	}
	return content
}

var (
	blankLinesRe = regexp.MustCompile(`\n[ \t]*\n\s*`)
	plainURLRe   = regexp.MustCompile(`https?://[^\s<>"]+[^\s<>".,;:!?)\]}'*]`)
)

// FromText builds a document from plain text: text separated by blank lines
// becomes paragraphs, other line breaks stay line breaks, and http(s) URLs
// become links. Nothing else has a special meaning.
func FromText(src string) Doc {
	d := EmptyDoc()
	src = strings.Trim(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if strings.TrimSpace(src) == "" {
		return d
	}
	for _, para := range blankLinesRe.Split(src, -1) {
		content := []Inline{}
		for i, line := range strings.Split(para, "\n") {
			if i > 0 {
				content = append(content, Inline{Type: InlineBreak})
			}
			content = append(content, linkify(line)...)
		}
		d.Blocks = append(d.Blocks, Block{Type: BlockParagraph, Content: &content})
	}
	return d
}

// linkify returns a line as text runs with its http(s) URLs as links.
func linkify(line string) []Inline {
	var out []Inline
	last := 0
	for _, m := range plainURLRe.FindAllStringIndex(line, -1) {
		if m[0] > last {
			out = append(out, textRun(line[last:m[0]], nil))
		}
		u := line[m[0]:m[1]]
		out = append(out, link(u, []Inline{textRun(u, nil)})...)
		last = m[1]
	}
	if last < len(line) {
		out = append(out, textRun(line[last:], nil))
	}
	return out
}

// ToMarkdown writes a document as Markdown that FromMarkdown reads back to the
// same document, apart from task IDs and what Markdown can't express: empty
// paragraphs are left out, line breaks in headings become spaces, spaces at
// the start and end of a line and line breaks at the end of a block are
// dropped, and a mark may be lost where CommonMark can't read it for sure:
// when it starts or ends inside a word, between punctuation and a letter or
// digit, or right where another mark ends. Attachments are written as images
// or links to their URL, with their names from att, which may be nil.
func ToMarkdown(d Doc, att *MarkdownAttachments) string {
	var parts []string
	// Lists directly after each other stay separate when their markers differ.
	prevMarker := ""
	for _, b := range d.Blocks {
		var md string
		marker := ""
		switch b.Type {
		case BlockHeading:
			text := strings.ReplaceAll(writeInlines(content(b)), "\n", " ")
			// A heading's trailing "#"s would be read as its closing sequence.
			if run := len(text) - len(strings.TrimRight(text, "#")); run > 0 && !strings.HasSuffix(text[:len(text)-run], `\`) {
				text = text[:len(text)-run] + `\` + text[len(text)-run:]
			}
			md = strings.TrimRight(strings.Repeat("#", b.Level)+" "+text, " ")
		case BlockParagraph:
			md = writeInlines(content(b))
		case BlockQuote:
			md = prefixLines(writeInlines(content(b)), "> ", ">")
			if md == "" {
				md = ">"
			}
		case BlockCode:
			md = codeBlock(*b.Text)
		case BlockDivider:
			md = "---"
		case BlockList:
			marker = listMarker(b.Style, prevMarker)
			md = writeList(b, marker)
		case BlockAttachment:
			md = att.markdown(b.ID)
		}
		prevMarker = marker
		if md != "" {
			parts = append(parts, md)
		}
	}
	return strings.Join(parts, "\n\n")
}

func content(b Block) []Inline {
	if b.Content == nil {
		return nil
	}
	return *b.Content
}

// listMarker returns the bullet ("-" or "*") or the ordered delimiter ("." or
// ")") of a list, the other one when the previous block used it.
func listMarker(style, prev string) string {
	if style == ListOrdered {
		if prev == "." {
			return ")"
		}
		return "."
	}
	if prev == "-" {
		return "*"
	}
	return "-"
}

// writeList writes a list with its nested lists, using marker for its items.
func writeList(b Block, marker string) string {
	var lines []string
	for i, it := range *b.Items {
		var head string
		switch b.Style {
		case ListOrdered:
			head = fmt.Sprintf("%d%s ", i+1, marker)
		case ListTask:
			box := "[ ]"
			if it.Checked != nil && *it.Checked {
				box = "[x]"
			}
			head = marker + " " + box + " "
		default:
			head = marker + " "
		}
		// Continuation lines and nested lists line up with the item's text,
		// which starts after the marker (a task's box is part of the text).
		indent := strings.Repeat(" ", utf8.RuneCountInString(head))
		if b.Style == ListTask {
			indent = strings.Repeat(" ", len(marker)+1)
		}
		text := writeInlines(it.Content)
		// An empty item with a nested list starts that list on its marker's
		// line, the only way Markdown keeps the list inside the item, with the
		// other marker, so "- - -" doesn't become a divider. A task's box
		// already starts the item's text, so its list can follow below.
		sameLine := text == "" && b.Style != ListTask
		var children string
		if it.Children != nil {
			prev := ""
			if sameLine {
				prev = marker
			}
			children = writeList(*it.Children, listMarker(it.Children.Style, prev))
		}
		if sameLine {
			text, children = children, ""
		}
		first, rest, _ := strings.Cut(text, "\n")
		lines = append(lines, strings.TrimRight(head+first, " "))
		if rest != "" {
			lines = append(lines, prefixLines(rest, indent, ""))
		}
		if children != "" {
			// An empty item can't interrupt the item's text, so a blank line
			// comes first.
			if first, _, _ := strings.Cut(children, "\n"); emptyItemRe.MatchString(first) {
				lines = append(lines, "")
			}
			lines = append(lines, prefixLines(children, indent, ""))
		}
	}
	return strings.Join(lines, "\n")
}

// prefixLines puts prefix before every line of text, or empty before empty
// lines.
func prefixLines(text, prefix, empty string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l == "" {
			lines[i] = empty
		} else {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

// emptyItemRe matches the line of an empty bullet or ordered list item.
var emptyItemRe = regexp.MustCompile(`^(?:[-*]|\d+[.)])$`)

// codeBlock fences code with more backticks than any run in it.
func codeBlock(code string) string {
	fence := strings.Repeat("`", max(3, longestRun(code, '`')+1))
	if code == "" {
		return fence + "\n" + fence
	}
	return fence + "\n" + code + "\n" + fence
}

func longestRun(s string, c rune) int {
	longest, n := 0, 0
	for _, r := range s {
		if r == c {
			n++
			longest = max(longest, n)
		} else {
			n = 0
		}
	}
	return longest
}

// markOrder is the order marks open in, outermost first; code is always the
// innermost, as a code span.
var markOrder = []string{"bold", "italic", "strike"}

var markDelims = map[string]string{"bold": "**", "italic": "*", "strike": "~~"}

// writeInlines writes inline content with line breaks as "\n". An empty line
// is written as "\", a hard line break, and line breaks at the end are left
// out.
func writeInlines(content []Inline) string {
	w := &inlineWriter{lineStart: true, closedAt: -1}
	for _, in := range content {
		switch in.Type {
		case InlineText:
			w.text(*in.Text, in.Marks)
		case InlineBreak:
			w.newLine()
		case InlineLink:
			w.closeTo(0)
			w.link(in)
		}
	}
	w.closeTo(0)
	w.resolve(-1)
	w.lines = append(w.lines, string(w.buf))
	for i, l := range w.lines {
		l = strings.TrimRight(l, " \t")
		if l == "" {
			l = `\`
		}
		w.lines[i] = l
	}
	for len(w.lines) > 0 && w.lines[len(w.lines)-1] == `\` {
		w.lines = w.lines[:len(w.lines)-1]
	}
	return strings.Join(w.lines, "\n")
}

// openMark is an open mark and where its opening delimiter is in the line.
type openMark struct {
	mark string
	pos  int
}

// inlineWriter writes one line at a time into buf. CommonMark only reads a
// delimiter as emphasis where it can open or close, depending on the
// characters around it, so marks that can't open aren't opened, and closing
// delimiters wait in pending until the next character is known; if they
// can't close there, their opening delimiters are taken out again. Either way
// the mark is lost, but no delimiter shows as text.
type inlineWriter struct {
	lines []string
	buf   []byte
	// open are the open marks, outermost first.
	open []openMark
	// pending are closed marks whose closing delimiters, innermost first, go
	// at pendingAt once the next character is known.
	pending   []openMark
	pendingAt int
	// closedAt is where the last closing delimiters ended, so opening ones
	// right after them, which would join their run, are left out.
	closedAt  int
	lineStart bool
	// lineDigits is set while the line has only digits so far.
	lineDigits bool
}

// text writes a text run, opening and closing marks around it as needed.
func (w *inlineWriter) text(s string, marks []string) {
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			w.newLine()
		}
		w.run(line, marks)
	}
}

func (w *inlineWriter) newLine() {
	w.closeTo(0)
	w.resolve(-1)
	w.lines = append(w.lines, string(w.buf))
	w.buf = w.buf[:0]
	w.closedAt = -1
	w.lineStart = true
	w.lineDigits = false
}

func (w *inlineWriter) run(s string, marks []string) {
	if w.lineStart {
		s = strings.TrimLeft(s, " \t")
	}
	core := strings.TrimRight(strings.TrimLeft(s, " \t"), " \t")
	if core == "" {
		w.write(s)
		return
	}
	lead := s[:strings.Index(s, core)]
	trail := s[len(lead)+len(core):]

	// Keep the open marks this run also has, close the others.
	keep := 0
	for keep < len(w.open) && slices.Contains(marks, w.open[keep].mark) {
		keep++
	}
	w.closeTo(keep)

	var body string
	// A code span right after another one would join it.
	if slices.Contains(marks, "code") && lastRune(w.buf) != '`' {
		body = codeSpan(core)
	} else {
		body = escapeText(core, w.lineStart && len(w.open) == 0)
		// "1" and "." in two runs still start an ordered list.
		if d := len(core) - len(strings.TrimLeft(core, "0123456789")); w.lineDigits && !w.lineStart && lead == "" &&
			d < len(core) && (core[d] == '.' || core[d] == ')') {
			body = body[:d] + `\` + body[d:]
		}
	}
	digits := !slices.Contains(marks, "code") && lead == "" && strings.Trim(core, "0123456789") == ""
	w.lineDigits = digits && (w.lineStart || w.lineDigits) && len(w.open) == 0
	w.write(lead)
	var opening []string
	for _, m := range markOrder {
		if slices.Contains(marks, m) && !slices.ContainsFunc(w.open, func(o openMark) bool { return o.mark == m }) {
			opening = append(opening, m)
		}
	}
	// Closers right before the text are judged by it: openers never go
	// directly after closers.
	w.resolve(firstRune(body))
	if len(opening) > 0 && w.closedAt != len(w.buf) {
		delims := make([]string, len(opening))
		for i, m := range opening {
			delims[i] = markDelims[m]
		}
		keep := keepDelims(lastRune(w.buf), delims, firstRune(body), true)
		for i, m := range opening {
			if keep[i] {
				w.open = append(w.open, openMark{m, len(w.buf)})
				w.buf = append(w.buf, markDelims[m]...)
			}
		}
	}
	w.write(body)
	w.lineStart = false
	w.write(trail)
}

// write appends s, first writing or dropping the pending closing delimiters.
func (w *inlineWriter) write(s string) {
	if s == "" {
		return
	}
	w.resolve(firstRune(s))
	w.buf = append(w.buf, s...)
}

// closeTo closes the open marks above the first n. Their closing delimiters
// go before trailing spaces, since a closing delimiter can't follow a space.
func (w *inlineWriter) closeTo(n int) {
	if len(w.open) <= n {
		return
	}
	if len(w.pending) == 0 {
		w.pendingAt = len(bytes.TrimRight(w.buf, " \t"))
	}
	for i := len(w.open) - 1; i >= n; i-- {
		w.pending = append(w.pending, w.open[i])
	}
	w.open = w.open[:n]
}

// resolve writes the pending closing delimiters at pendingAt if they can
// close before next (-1 for the end of the line), and otherwise takes out
// their opening delimiters.
func (w *inlineWriter) resolve(next rune) {
	if len(w.pending) == 0 {
		return
	}
	pending := w.pending
	w.pending = nil
	// Spaces moved after the closers come right after them.
	if w.pendingAt < len(w.buf) {
		next, _ = utf8.DecodeRune(w.buf[w.pendingAt:])
	}
	delims := make([]string, len(pending))
	for i, o := range pending {
		delims[i] = markDelims[o.mark]
	}
	keep := keepDelims(lastRune(w.buf[:w.pendingAt]), delims, next, false)
	var closers []byte
	var dropped []openMark
	for i, o := range pending {
		if keep[i] {
			closers = append(closers, delims[i]...)
		} else {
			dropped = append(dropped, o)
		}
	}
	// Take the openers of the dropped closers out, last first, so earlier
	// positions stay right.
	slices.SortFunc(dropped, func(a, b openMark) int { return b.pos - a.pos })
	for _, o := range dropped {
		n := len(markDelims[o.mark])
		w.buf = slices.Delete(w.buf, o.pos, o.pos+n)
		for i := range w.open {
			if w.open[i].pos > o.pos {
				w.open[i].pos -= n
			}
		}
		w.pendingAt -= n
	}
	if len(closers) == 0 {
		return
	}
	w.buf = slices.Insert(w.buf, w.pendingAt, closers...)
	w.closedAt = w.pendingAt + len(closers)
	if w.closedAt != len(w.buf) {
		// Spaces follow the closers, so nothing can join their run.
		w.closedAt = -1
	}
}

// keepDelims returns which of the delimiters, written in this order between
// prev and next, CommonMark reads as opening (open) or closing ones, and only
// as those. Delimiters of the same character form one run, which can open or
// close depending on its neighbours, so runs that can't, or could do both,
// are dropped until all others can. A run next to the same character, such as
// an escaped one, doesn't count either.
func keepDelims(prev rune, delims []string, next rune, open bool) []bool {
	keep := make([]bool, len(delims))
	for i := range keep {
		keep[i] = true
	}
	for changed := true; changed; {
		changed = false
		var kept []int
		for i := range delims {
			if keep[i] {
				kept = append(kept, i)
			}
		}
		for j := 0; j < len(kept); {
			c := firstRune(delims[kept[j]])
			k := j
			for k+1 < len(kept) && firstRune(delims[kept[k+1]]) == c {
				k++
			}
			p, n := prev, next
			if j > 0 {
				p = lastRune([]byte(delims[kept[j-1]]))
			}
			if k+1 < len(kept) {
				n = firstRune(delims[kept[k+1]])
			}
			// A run that could both open and close, such as one inside a
			// word, may be paired either way.
			ok := p != c && n != c && canOpen(p, n) != canClose(p, n) && (open == canOpen(p, n))
			if !ok {
				for m := j; m <= k; m++ {
					keep[kept[m]] = false
				}
				changed = true
				break
			}
			j = k + 1
		}
	}
	return keep
}

// isPunct reports whether CommonMark counts r as punctuation for emphasis.
func isPunct(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }

// isSpace reports whether r is whitespace, or -1 for the start or end of a
// line.
func isSpace(r rune) bool { return r == -1 || unicode.IsSpace(r) }

// canOpen reports whether a delimiter run between prev and next can open
// emphasis: it is left-flanking (CommonMark).
func canOpen(prev, next rune) bool {
	return !isSpace(next) && (!isPunct(next) || isSpace(prev) || isPunct(prev))
}

// canClose reports whether a delimiter run between prev and next can close
// emphasis: it is right-flanking.
func canClose(prev, next rune) bool {
	return !isSpace(prev) && (!isPunct(prev) || isSpace(next) || isPunct(next))
}

func firstRune(s string) rune {
	if s == "" {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func lastRune(b []byte) rune {
	if len(b) == 0 {
		return -1
	}
	r, _ := utf8.DecodeLastRune(b)
	return r
}

// link writes a link as an autolink when its text is its address, and as
// [text](url) otherwise.
func (w *inlineWriter) link(in Inline) {
	text := ""
	for _, t := range *in.Content {
		text += *t.Text
	}
	plain := len(*in.Content) == 0 || len(*in.Content) == 1 && len((*in.Content)[0].Marks) == 0
	u := in.URL
	switch {
	case plain && text == u && !strings.ContainsAny(u, " <>"):
		w.write("<" + u + ">")
	case plain && strings.HasPrefix(strings.ToLower(u), "mailto:") && text == u[len("mailto:"):] && !strings.ContainsAny(text, " <>"):
		w.write("<" + text + ">")
	default:
		inner := &inlineWriter{closedAt: -1}
		for _, t := range *in.Content {
			inner.run(*t.Text, t.Marks)
		}
		inner.closeTo(0)
		inner.resolve(']')
		dest := u
		if strings.ContainsAny(u, " ()<>\\") {
			dest = "<" + strings.NewReplacer(`\`, `\\`, "<", `\<`, ">", `\>`).Replace(u) + ">"
		}
		w.write("[" + string(inner.buf) + "](" + dest + ")")
	}
	w.lineStart = false
}

// codeSpan writes text as a code span with more backticks than any run in
// it, padded with spaces where CommonMark would strip them.
func codeSpan(text string) string {
	delim := strings.Repeat("`", longestRun(text, '`')+1)
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") ||
		(strings.HasPrefix(text, " ") && strings.HasSuffix(text, " ") && strings.Trim(text, " ") != "") {
		text = " " + text + " "
	}
	return delim + text + delim
}

var (
	// lineStartRe matches what makes a line a heading, list, quote, code
	// fence or setext underline, or an ordered list item's number.
	lineStartRe = regexp.MustCompile(`^(?:[#>+=-]|\d{1,9}[.)])`)
	bareLinkRe  = regexp.MustCompile(`(?i)(https?|ftp)://|\bwww\.`)
)

// escapeText escapes the characters of plain text that Markdown would read
// as markup, including bare URLs and email addresses that would become links.
// atLineStart also escapes what starts a block.
func escapeText(s string, atLineStart bool) string {
	var b strings.Builder
	if atLineStart {
		if m := lineStartRe.FindString(s); m != "" {
			last := len(m) - 1
			b.WriteString(m[:last] + `\` + m[last:])
			s = s[len(m):]
		}
	}
	links := bareLinkRe.FindAllStringIndex(s, -1)
	for i, r := range s {
		switch {
		case strings.ContainsRune("\\`*_~[]<>&|!@", r):
			b.WriteByte('\\')
		case r == ':' || r == '.':
			// Break up "https://" and "www." so they stay text.
			for _, l := range links {
				if i == l[1]-3 && r == ':' || i == l[1]-1 && r == '.' {
					b.WriteByte('\\')
				}
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ReplaceContentFromMarkdown reads md as the new content of a note whose
// content was old. A task item of the new content gets the ID of the first
// unused task item of the old content with the same text, in document order,
// so tasks keep their IDs when a connected app edits a note's text; the
// others get new IDs when the result is parsed. Checked states come from md.
func ReplaceContentFromMarkdown(old Doc, md string, att *MarkdownAttachments) Doc {
	ids := map[string][]string{}
	walkTasks(&old, func(it *Item) {
		key := taskText(it.Content)
		ids[key] = append(ids[key], it.ID)
	})
	d := FromMarkdown(md, att)
	walkTasks(&d, func(it *Item) {
		key := taskText(it.Content)
		if free := ids[key]; len(free) > 0 {
			it.ID, ids[key] = free[0], free[1:]
		}
	})
	return d
}

// walkTasks calls fn for every task item of d, in document order.
func walkTasks(d *Doc, fn func(*Item)) {
	var visit func(b *Block)
	visit = func(b *Block) {
		if b.Type != BlockList || b.Items == nil {
			return
		}
		for i := range *b.Items {
			it := &(*b.Items)[i]
			if b.Style == ListTask {
				fn(it)
			}
			if it.Children != nil {
				visit(it.Children)
			}
		}
	}
	for i := range d.Blocks {
		visit(&d.Blocks[i])
	}
}

// taskText is a task item's text without marks and links, trimmed.
func taskText(content []Inline) string {
	var b strings.Builder
	for _, in := range content {
		switch in.Type {
		case InlineText:
			b.WriteString(*in.Text)
		case InlineBreak:
			b.WriteString("\n")
		case InlineLink:
			for _, t := range *in.Content {
				b.WriteString(*t.Text)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// TitleFromHeading splits a leading level-1 heading off d, for connected apps
// that keep a note's title as the first line of its Markdown. When the
// heading's text, trimmed and with line breaks as spaces, has 1 to
// MaxNoteTitle characters, it is the title and the rest of d is the content;
// otherwise the title is nil and the content is all of d.
func TitleFromHeading(d Doc) (*string, Doc) {
	if len(d.Blocks) == 0 || d.Blocks[0].Type != BlockHeading || d.Blocks[0].Level != 1 || d.Blocks[0].Content == nil {
		return nil, d
	}
	title := strings.TrimSpace(strings.ReplaceAll(inlineText(*d.Blocks[0].Content), "\n", " "))
	if title == "" || utf8.RuneCountInString(title) > MaxNoteTitle {
		return nil, d
	}
	return &title, Doc{V: d.V, Blocks: d.Blocks[1:]}
}

// WithTitleHeading is d with title as a leading level-1 heading, the
// opposite of TitleFromHeading. A nil title leaves d as it is.
func WithTitleHeading(title *string, d Doc) Doc {
	if title == nil {
		return d
	}
	text := *title
	heading := Block{Type: BlockHeading, Level: 1, Content: &[]Inline{{Type: InlineText, Text: &text}}}
	return Doc{V: d.V, Blocks: append([]Block{heading}, d.Blocks...)}
}
