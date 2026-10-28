package notes

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestToMarkdown(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"heading and tasks", "## Week\n\n- [x] deploy", "## Week\n\n- [x] deploy"},
		{"literal asterisks", `2 \* 3 \* 4`, `2 \* 3 \* 4`},
		{"paragraphs", "one\ntwo\n\nthree", "one\ntwo\n\nthree"},
		{"marks", "**b** _i_ ~~s~~ `c`", "**b** *i* ~~s~~ `c`"},
		{"nested marks", "**a _b_ c**", "**a *b* c**"},
		{"link", "[Go](https://go.dev)", "[Go](https://go.dev)"},
		{"bare URL", "see https://go.dev", "see <https://go.dev>"},
		{"URL as text", `see https\://go.dev`, `see https\://go.dev`},
		{"email", "<ann@example.org>", "<ann@example.org>"},
		{"quote", "> a\n> b", "> a\n> b"},
		{"code with backticks", "````\n```\n````", "````\n```\n````"},
		{"code span with backtick", "``a`b``", "``a`b``"},
		{"divider", "a\n\n---\n\nb", "a\n\n---\n\nb"},
		{"nested lists", "1. a\n   - b\n     - [ ] c\n2. d", "1. a\n   - b\n     - [ ] c\n2. d"},
		{"adjacent lists", "- a\n\n* b\n\n- [ ] c", "- a\n\n* b\n\n- [ ] c"},
		{"bullet then task list", "- a\n\n* [ ] c", "- a\n\n* [ ] c"},
		{"line start", `\# not a heading` + "\n" + `1\. not a list`, `\# not a heading` + "\n" + `1\. not a list`},
		{"empty line in a paragraph", "a\n\\\nb", "a\n\\\nb"},
		{"item with lines", "- a\n  b", "- a\n  b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, FromMarkdown(tt.md))
			if got := ToMarkdown(d); got != tt.want {
				t.Errorf("ToMarkdown(FromMarkdown(%q))\n got: %q\nwant: %q", tt.md, got, tt.want)
			}
		})
	}
}

func TestToMarkdownExceptions(t *testing.T) {
	empty := []Inline{}
	d := Doc{V: 1, Blocks: []Block{
		{Type: BlockParagraph, Content: &[]Inline{textRun("a", nil)}},
		{Type: BlockParagraph, Content: &empty},
		{Type: BlockParagraph, Content: &[]Inline{textRun("b", nil), {Type: InlineBreak}}},
		{Type: BlockHeading, Level: 1, Content: &[]Inline{textRun("c", nil), {Type: InlineBreak}, textRun("d", nil)}},
	}}
	if got, want := ToMarkdown(d), "a\n\nb\n\n# c d"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// canon writes a document in a form that ignores what ToMarkdown may change:
// task IDs, empty paragraphs, line breaks in headings, spaces at the start
// and end of lines, line breaks at the end of a block, and marks on spaces.
// Without marks, it leaves marks out altogether.
func canon(d Doc, marks bool) string {
	var b strings.Builder
	var block func(bl Block, indent string)
	block = func(bl Block, indent string) {
		switch bl.Type {
		case BlockParagraph, BlockQuote, BlockHeading:
			c := canonInlines(content(bl), bl.Type == BlockHeading, marks)
			if bl.Type == BlockParagraph && c == "" {
				return
			}
			fmt.Fprintf(&b, "%s%s%d %s\n", indent, bl.Type, bl.Level, c)
		case BlockCode:
			fmt.Fprintf(&b, "%scode %q\n", indent, *bl.Text)
		case BlockDivider:
			fmt.Fprintf(&b, "%sdivider\n", indent)
		case BlockList:
			fmt.Fprintf(&b, "%slist %s\n", indent, bl.Style)
			for _, it := range *bl.Items {
				checked := it.Checked != nil && *it.Checked
				fmt.Fprintf(&b, "%s  item %v %s\n", indent, checked, canonInlines(it.Content, false, marks))
				if it.Children != nil {
					block(*it.Children, indent+"    ")
				}
			}
		}
	}
	for _, bl := range d.Blocks {
		block(bl, "")
	}
	return b.String()
}

type unit struct {
	r    rune
	attr string
}

func canonInlines(content []Inline, heading, marks bool) string {
	var units []unit
	add := func(text string, marks []string, link string) {
		sorted := slices.Clone(marks)
		slices.Sort(sorted)
		for _, r := range text {
			attr := link + strings.Join(sorted, ",")
			if unicode.IsSpace(r) {
				attr = link
			}
			units = append(units, unit{r, attr})
		}
	}
	for _, in := range content {
		switch in.Type {
		case InlineText:
			add(*in.Text, in.Marks, "")
		case InlineBreak:
			if heading {
				units = append(units, unit{' ', ""})
			} else {
				units = append(units, unit{'\n', ""})
			}
		case InlineLink:
			units = append(units, unit{'[', "link"})
			for _, t := range *in.Content {
				add(*t.Text, t.Marks, "link:")
			}
			units = append(units, unit{']', "link " + in.URL})
		}
	}
	// Trim spaces at line edges and line breaks at the end.
	var lines [][]unit
	start := 0
	for i, u := range units {
		if u.r == '\n' && u.attr == "" {
			lines = append(lines, units[start:i])
			start = i + 1
		}
	}
	lines = append(lines, units[start:])
	space := func(u unit) bool { return u.r == ' ' || u.r == '\t' }
	for i, l := range lines {
		for len(l) > 0 && space(l[0]) {
			l = l[1:]
		}
		for len(l) > 0 && space(l[len(l)-1]) {
			l = l[:len(l)-1]
		}
		lines[i] = l
	}
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteString("⏎")
		}
		attr := ""
		for _, u := range l {
			if marks && u.attr != attr && !space(u) {
				fmt.Fprintf(&b, "{%s}", u.attr)
				attr = u.attr
			}
			b.WriteRune(u.r)
		}
	}
	return b.String()
}

// roundTrip checks that d survives ToMarkdown and FromMarkdown, apart from
// the listed exceptions. Without strictMarks, marks may be lost as well, but
// no text and no structure.
func roundTrip(t *testing.T, d Doc, strictMarks bool) {
	t.Helper()
	md := ToMarkdown(d)
	back := mustParse(t, FromMarkdown(md))
	got, want := canon(back, strictMarks), canon(d, strictMarks)
	if got != want {
		t.Errorf("round trip changed the document\nmarkdown:\n%s\n got:\n%s\nwant:\n%s", md, got, want)
	}
}

func TestToMarkdownRoundTrip(t *testing.T) {
	for _, md := range []string{
		"# Plan\n\nBuy **milk** and _eggs_, see [shop](https://shop.example/list?a=1&b=(2)).\n\n- [ ] call\n- [x] done\n  - [ ] nested",
		"> quoted *text*\n> with `code` and https://example.org\n\n```go\nfmt.Println(\"`\")\n```",
		"1. one\n2. two\n   1. sub\n\n1) other\n\n---\n\nend with \\* and \\_ and <tag> & co",
		"Mail ann@example.org or visit www.example.org.\n\nprice: 5 * 3 = 15 # total\n- not a list\n\\\nafter an empty line",
	} {
		roundTrip(t, mustParse(t, FromMarkdown(md)), true)
	}
}

func FuzzToMarkdown(f *testing.F) {
	for _, s := range []string{
		"## Week\n- [ ] a\n  - [x] b", "> q\n> - a", "*a **b `c` d** e*", "1. a\n2. b\n\n1) c",
		"[a\nb](https://e.org)", "a\n\\\nb", "\\# x\n\\- y", "<https://e.org/a(b)>", "`` ` ``",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, md string) {
		// Notes arrive as JSON, which has no invalid UTF-8.
		if !utf8.ValidString(md) {
			return
		}
		d := FromMarkdown(md)
		raw, err := jsonSize(d)
		if err != nil || raw > MaxDocSize/2 {
			return
		}
		roundTrip(t, mustParse(t, d), false)
	})
}

func jsonSize(d Doc) (int, error) {
	b, err := json.Marshal(d)
	return len(b), err
}

func tasksByText(d Doc) map[string][]string {
	ids := map[string][]string{}
	walkTasks(&d, func(it *Item) {
		ids[taskText(it.Content)] = append(ids[taskText(it.Content)], it.ID)
	})
	return ids
}

func TestReplaceContentFromMarkdown(t *testing.T) {
	old := mustParse(t, FromMarkdown("- [x] milk\n- [ ] eggs\n- [ ] call\n- [ ] call"))
	before := tasksByText(old)

	t.Run("edited list", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [x] milk\n- [ ] eggs\n- [ ] bread"))
		after := tasksByText(d)
		if after["milk"][0] != before["milk"][0] || after["eggs"][0] != before["eggs"][0] {
			t.Errorf("milk or eggs lost their IDs: %v → %v", before, after)
		}
		if id := after["bread"][0]; id == "" || slices.Contains(slices.Concat(before["milk"], before["eggs"], before["call"]), id) {
			t.Errorf("bread has ID %q, want a new one", id)
		}
	})

	t.Run("reordered tasks", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] eggs\n- [x] milk"))
		after := tasksByText(d)
		if after["milk"][0] != before["milk"][0] || after["eggs"][0] != before["eggs"][0] {
			t.Errorf("IDs changed: %v → %v", before, after)
		}
	})

	t.Run("renamed task", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [x] oat milk"))
		if id := tasksByText(d)["oat milk"][0]; id == before["milk"][0] {
			t.Error("a renamed task kept its ID")
		}
	})

	t.Run("same text more often", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] call\n- [ ] call\n- [ ] call"))
		after := tasksByText(d)["call"]
		if after[0] != before["call"][0] || after[1] != before["call"][1] || slices.Contains(before["call"], after[2]) {
			t.Errorf("call IDs %v, want the old %v and a new one", after, before["call"])
		}
	})

	t.Run("checked state from the text", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] milk"))
		if *(*d.Blocks[0].Items)[0].Checked {
			t.Error("milk is still checked")
		}
	})
}
