package notes

import (
	"encoding/json"
	"fmt"
	"reflect"
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
		{"location", "[Stephansplatz](geo:48.208176,16.373819)", "[Stephansplatz](geo:48.208176,16.373819)"},
		{"location without label", "[48.2, 16.37](geo:48.2,16.37)", "[48.2, 16.37](geo:48.2,16.37)"},
		{"location with Markdown in its label", `[a \*b\* \[c\]](geo:-1.5,2)`, `[a \*b\* \[c\]](geo:-1.5,2)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, FromMarkdown(tt.md, nil))
			if got := ToMarkdown(d, nil); got != tt.want {
				t.Errorf("ToMarkdown(FromMarkdown(%q), nil)\n got: %q\nwant: %q", tt.md, got, tt.want)
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
	if got, want := ToMarkdown(d, nil), "a\n\nb\n\n# c d"; got != want {
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
		case BlockLocation:
			label := "-"
			if bl.Label != nil {
				label = *bl.Label
			}
			fmt.Fprintf(&b, "%slocation %v,%v %s\n", indent, *bl.Lat, *bl.Lng, label)
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
	md := ToMarkdown(d, nil)
	back := mustParse(t, FromMarkdown(md, nil))
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
		"Trip\n\n[Stephansplatz](geo:48.208176,16.373819)\n\n[-12.5, 0](geo:-12.5,0)\n\n[# 1 \\* x](geo:0,-180)",
	} {
		roundTrip(t, mustParse(t, FromMarkdown(md, nil)), true)
	}
}

func FuzzToMarkdown(f *testing.F) {
	for _, s := range []string{
		"## Week\n- [ ] a\n  - [x] b", "> q\n> - a", "*a **b `c` d** e*", "1. a\n2. b\n\n1) c",
		"[a\nb](https://e.org)", "[x](geo:1,2)", "[ \\* ](geo:-0.0000004,180)", "a\n\\\nb", "\\# x\n\\- y", "<https://e.org/a(b)>", "`` ` ``",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, md string) {
		// Notes arrive as JSON, which has no invalid UTF-8.
		if !utf8.ValidString(md) {
			return
		}
		d := FromMarkdown(md, nil)
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
	old := mustParse(t, FromMarkdown("- [x] milk\n- [ ] eggs\n- [ ] call\n- [ ] call", nil))
	before := tasksByText(old)

	t.Run("edited list", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [x] milk\n- [ ] eggs\n- [ ] bread", nil))
		after := tasksByText(d)
		if after["milk"][0] != before["milk"][0] || after["eggs"][0] != before["eggs"][0] {
			t.Errorf("milk or eggs lost their IDs: %v → %v", before, after)
		}
		if id := after["bread"][0]; id == "" || slices.Contains(slices.Concat(before["milk"], before["eggs"], before["call"]), id) {
			t.Errorf("bread has ID %q, want a new one", id)
		}
	})

	t.Run("reordered tasks", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] eggs\n- [x] milk", nil))
		after := tasksByText(d)
		if after["milk"][0] != before["milk"][0] || after["eggs"][0] != before["eggs"][0] {
			t.Errorf("IDs changed: %v → %v", before, after)
		}
	})

	t.Run("renamed task", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [x] oat milk", nil))
		if id := tasksByText(d)["oat milk"][0]; id == before["milk"][0] {
			t.Error("a renamed task kept its ID")
		}
	})

	t.Run("same text more often", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] call\n- [ ] call\n- [ ] call", nil))
		after := tasksByText(d)["call"]
		if after[0] != before["call"][0] || after[1] != before["call"][1] || slices.Contains(before["call"], after[2]) {
			t.Errorf("call IDs %v, want the old %v and a new one", after, before["call"])
		}
	})

	t.Run("checked state from the text", func(t *testing.T) {
		d := mustParse(t, ReplaceContentFromMarkdown(old, "- [ ] milk", nil))
		if *(*d.Blocks[0].Items)[0].Checked {
			t.Error("milk is still checked")
		}
	})
}

func TestTitleHeading(t *testing.T) {
	content := FromMarkdown("- [ ] milk\n\n# Inner", nil)
	for _, title := range []string{"Groceries", "2 * 3 # not a heading", "[x] _a_"} {
		md := ToMarkdown(WithTitleHeading(&title, content), nil)
		got, rest := TitleFromHeading(FromMarkdown(md, nil))
		if got == nil || *got != title {
			t.Errorf("title %q: read back %v from %q", title, got, md)
		}
		if ToMarkdown(rest, nil) != ToMarkdown(content, nil) {
			t.Errorf("title %q: content %q, want %q", title, ToMarkdown(rest, nil), ToMarkdown(content, nil))
		}
	}
	if md := ToMarkdown(WithTitleHeading(nil, content), nil); md != ToMarkdown(content, nil) {
		t.Errorf("no title = %q", md)
	}
	for name, md := range map[string]string{
		"level 2":     "## Week\n\n- [ ] review",
		"paragraph":   "milk",
		"empty":       "",
		"too long":    "# " + strings.Repeat("x", MaxNoteTitle+1),
		"blank title": "#\n\ntext",
	} {
		if title, rest := TitleFromHeading(FromMarkdown(md, nil)); title != nil || len(rest.Blocks) != len(FromMarkdown(md, nil).Blocks) {
			t.Errorf("%s: title %v, want none and all blocks kept", name, title)
		}
	}
}

func TestMarkdownAttachments(t *testing.T) {
	const origin = "https://boardly.example"
	roses := NoteAttachment{ID: "00000000-0000-4000-8000-000000000001", Name: "roses.jpg", ContentType: "image/jpeg"}
	report := NoteAttachment{ID: "00000000-0000-4000-8000-000000000002", Name: "my report (1).pdf", ContentType: "application/pdf"}
	att := &MarkdownAttachments{Origin: origin, ByID: map[string]NoteAttachment{roses.ID: roses, report.ID: report}}
	d := Doc{V: DocVersion, Blocks: []Block{
		{Type: BlockParagraph, Content: &[]Inline{textRun("Garden", nil)}},
		{Type: BlockAttachment, ID: roses.ID},
		{Type: BlockAttachment, ID: report.ID},
	}}

	md := ToMarkdown(d, att)
	want := "Garden\n\n![roses.jpg](" + origin + "/api/attachments/" + roses.ID + "/roses.jpg)\n\n" +
		"[my report (1).pdf](" + origin + "/api/attachments/" + report.ID + "/my%20report%20%281%29.pdf)"
	if md != want {
		t.Fatalf("ToMarkdown =\n%s\nwant\n%s", md, want)
	}
	if back := FromMarkdown(md, att); !reflect.DeepEqual(back, d) {
		t.Fatalf("round trip = %+v", back.Blocks)
	}

	// Without attachments to resolve, the image is its alt text and the file a link.
	plain := FromMarkdown(md, nil)
	if len(plain.Blocks) != 3 || plain.Blocks[1].Type != BlockParagraph || plain.Blocks[2].Type != BlockParagraph {
		t.Fatalf("without attachments = %+v", plain.Blocks)
	}

	for name, tt := range map[string]struct {
		md   string
		want []string // block types, attachments as their ID
	}{
		"own image by path":       {"![x](/api/attachments/" + roses.ID + "/x.jpg)", []string{roses.ID}},
		"file link alone":         {"[report](/api/attachments/" + report.ID + ")", []string{report.ID}},
		"image splits paragraph":  {"before ![x](" + origin + "/api/attachments/" + roses.ID + "/x) after", []string{BlockParagraph, roses.ID, BlockParagraph}},
		"image on its own line":   {"before\n![x](" + origin + "/api/attachments/" + roses.ID + "/x)\nafter", []string{BlockParagraph, roses.ID, BlockParagraph}},
		"file link in text":       {"see [report](/api/attachments/" + report.ID + ") here", []string{BlockParagraph}},
		"other user's attachment": {"![x](" + origin + "/api/attachments/00000000-0000-4000-8000-0000000000ff/x.png)", []string{BlockParagraph}},
		"other origin":            {"![x](https://evil.example/api/attachments/" + roses.ID + "/x.jpg)", []string{BlockParagraph}},
	} {
		got := FromMarkdown(tt.md, att)
		var types []string
		for _, b := range got.Blocks {
			if b.Type == BlockAttachment {
				types = append(types, b.ID)
			} else {
				types = append(types, b.Type)
			}
		}
		if !reflect.DeepEqual(types, tt.want) {
			t.Errorf("%s: blocks %v, want %v", name, types, tt.want)
		}
	}

	split := FromMarkdown("before\n![x](/api/attachments/"+roses.ID+")\nafter", att)
	if first := *split.Blocks[0].Content; len(first) != 1 || *first[0].Text != "before" {
		t.Errorf("text before the image = %+v", first)
	}

	// Replacing content keeps attachments too.
	if got := ReplaceContentFromMarkdown(d, md, att); !reflect.DeepEqual(got.Blocks, d.Blocks) {
		t.Errorf("ReplaceContentFromMarkdown = %+v", got.Blocks)
	}
}
