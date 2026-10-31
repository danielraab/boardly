package notes

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// describe writes a document compactly: one line per block, list items
// indented, marks as <mark>…</mark>, links as [text](url) and line breaks as ⏎.
func describe(d Doc) string {
	var b strings.Builder
	var block func(bl Block, indent string)
	block = func(bl Block, indent string) {
		switch bl.Type {
		case BlockHeading:
			fmt.Fprintf(&b, "%sh%d %s\n", indent, bl.Level, describeInlines(*bl.Content))
		case BlockParagraph, BlockQuote:
			fmt.Fprintf(&b, "%s%s %s\n", indent, bl.Type, describeInlines(*bl.Content))
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
			for _, it := range *bl.Items {
				marker := "-"
				switch {
				case bl.Style == ListOrdered:
					marker = "1."
				case it.Checked != nil && *it.Checked:
					marker = "[x]"
				case it.Checked != nil:
					marker = "[ ]"
				}
				fmt.Fprintf(&b, "%s%s %s\n", indent, marker, describeInlines(it.Content))
				if it.Children != nil {
					block(*it.Children, indent+"  ")
				}
			}
		}
	}
	for _, bl := range d.Blocks {
		block(bl, "")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func describeInlines(content []Inline) string {
	var b strings.Builder
	for _, in := range content {
		switch in.Type {
		case InlineText:
			s := *in.Text
			for _, m := range in.Marks {
				s = "<" + m + ">" + s + "</" + m + ">"
			}
			b.WriteString(s)
		case InlineBreak:
			b.WriteString("⏎")
		case InlineLink:
			fmt.Fprintf(&b, "[%s](%s)", describeInlines(*in.Content), in.URL)
		}
	}
	return b.String()
}

// describeRuns writes the inline content of every block with "||" between
// adjacent text runs with the same marks, which the converter must merge.
func describeRuns(d Doc) string {
	var b strings.Builder
	var runs func(content []Inline)
	runs = func(content []Inline) {
		for i, in := range content {
			if i > 0 && in.Type == InlineText && content[i-1].Type == InlineText && fmt.Sprint(in.Marks) == fmt.Sprint(content[i-1].Marks) {
				b.WriteString("||")
			}
			if in.Type == InlineLink {
				runs(*in.Content)
			}
		}
	}
	var block func(bl Block)
	block = func(bl Block) {
		if bl.Content != nil {
			runs(*bl.Content)
		}
		if bl.Items != nil {
			for _, it := range *bl.Items {
				runs(it.Content)
				if it.Children != nil {
					block(*it.Children)
				}
			}
		}
	}
	for _, bl := range d.Blocks {
		block(bl)
	}
	return b.String()
}

// mustParse checks that d is a valid document and returns it as ParseDoc
// normalizes it.
func mustParse(t *testing.T, d Doc) Doc {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseDoc(raw)
	if err != nil {
		t.Fatalf("ParseDoc rejects the converted document: %v\n%s", err, raw)
	}
	if strings.Contains(describeRuns(parsed), "||") {
		t.Errorf("adjacent text runs with the same marks:\n%s", raw)
	}
	return parsed
}

func TestFromMarkdown(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"empty", "", ""},
		{"headings and tasks", "## Week\n\n- [ ] review PR\n- [x] deploy", "h2 Week\n[ ] review PR\n[x] deploy"},
		{"deep heading", "##### Small", "h3 Small"},
		{"setext heading", "Title\n=====", "h1 Title"},
		{"line breaks kept", "milk\neggs\nbread", "paragraph milk⏎eggs⏎bread"},
		{"paragraphs", "one\n\ntwo", "paragraph one\nparagraph two"},
		{"marks", "**bold** *it* ~~gone~~ `x*y`", "paragraph <bold>bold</bold> <italic>it</italic> <strike>gone</strike> <code>x*y</code>"},
		{"nested marks", "***both***", "paragraph <bold><italic>both</italic></bold>"},
		{"link", "[Go](https://go.dev)", "paragraph [Go](https://go.dev)"},
		{"escaped link address", `[a](https://x.org/a\_b&amp;c)`, "paragraph [a](https://x.org/a_b&c)"},
		{"bold link text", "[**Go**](https://go.dev)", "paragraph [<bold>Go</bold>](https://go.dev)"},
		{"bare URL", "see https://example.org/a now", "paragraph see [https://example.org/a](https://example.org/a) now"},
		{"autolink email", "<ann@example.org>", "paragraph [ann@example.org](mailto:ann@example.org)"},
		{"script link", "[click](javascript:alert(1))", "paragraph click"},
		{"relative link", "[home](/home)", "paragraph home"},
		{"image as alt text", "![logo](https://example.org/l.png)", "paragraph logo"},
		{"inline HTML kept", "a <b>hi</b> b", "paragraph a <b>hi</b> b"},
		{"HTML block kept", "<div>\nhi\n</div>", "paragraph <div>⏎hi⏎</div>"},
		{"escapes", `2 \* 3 &amp; 4`, "paragraph 2 * 3 & 4"},
		{"quote", "> first\n> second\n>\n> third", "quote first⏎second⏎third"},
		{"fenced code", "```go\nfmt.Println()\n```", `code "fmt.Println()"`},
		{"indented code", "    x := 1", `code "x := 1"`},
		{"divider", "a\n\n---\n\nb", "paragraph a\ndivider\nparagraph b"},
		{"bullet list nested", "- a\n  - b\n- c", "- a\n  - b\n- c"},
		{"ordered list", "1. a\n2. b", "1. a\n1. b"},
		{"mixed task list", "- [ ] a\n- b", "[ ] a\n[ ] b"},
		{"loose list item", "- a\n\n  more\n- b", "- a⏎more\n- b"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |", "paragraph a | b⏎1 | 2"},
		{"hard break", "a  \nb", "paragraph a⏎b"},
		{"windows line ends", "a\r\nb\rc", "paragraph a⏎b⏎c"},
		{"location", "[Stephansplatz](geo:48.208176,16.373819)", "location 48.208176,16.373819 Stephansplatz"},
		{"location without label", "[48.2, 16.37](geo:48.2,16.37)", "location 48.2,16.37 -"},
		{"location with empty text", "[](geo:48.2,16.37)", "location 48.2,16.37 -"},
		{"location link inside text", "Meet at [Stephansplatz](geo:48.208176,16.373819)", "paragraph Meet at Stephansplatz"},
		{"location out of range", "[x](geo:91,0)", "paragraph x"},
		{"location with parameters", "[Hut](GEO:47.1,11.2,2100;u=30)", "location 47.1,11.2 Hut"},
		{"location rounded", "[x](geo:48.20817634,16.3738189)", "location 48.208176,16.373819 x"},
		{"location label with marks", "[**Dom** am\nPlatz](geo:1,2)", "location 1,2 Dom am Platz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, FromMarkdown(tt.md, nil))
			if got := describe(d); got != tt.want {
				t.Errorf("FromMarkdown(%q, nil)\n got: %s\nwant: %s", tt.md, got, tt.want)
			}
		})
	}
}

func TestFromMarkdownTaskIDs(t *testing.T) {
	d := mustParse(t, FromMarkdown("- [ ] a\n- [X] b", nil))
	items := *d.Blocks[0].Items
	if items[0].ID == "" || items[1].ID == "" || items[0].ID == items[1].ID {
		t.Errorf("task IDs = %q, %q, want two different IDs", items[0].ID, items[1].ID)
	}
	if !*items[1].Checked {
		t.Error("[X] is not checked")
	}
}

func TestFromMarkdownDeepLists(t *testing.T) {
	var md strings.Builder
	for i := range 9 {
		fmt.Fprintf(&md, "%s- level %d\n", strings.Repeat("  ", i), i+1)
	}
	d := mustParse(t, FromMarkdown(md.String(), nil))
	want := "- level 1\n  - level 2\n    - level 3\n      - level 4\n        - level 5\n          - level 6⏎level 7⏎level 8⏎level 9"
	if got := describe(d); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFromText(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"empty", "", ""},
		{"blank", "\n  \n", ""},
		{"shared message", "Meet at 8\nBring snacks\n\nSee https://example.org/map",
			"paragraph Meet at 8⏎Bring snacks\nparagraph See [https://example.org/map](https://example.org/map)"},
		{"markdown characters stay", "# not a heading *really*", "paragraph # not a heading *really*"},
		{"several blank lines", "a\n\n\n\nb", "paragraph a\nparagraph b"},
		{"windows line ends", "a\r\nb", "paragraph a⏎b"},
		{"URL before punctuation", "Look: https://example.org/a.", "paragraph Look: [https://example.org/a](https://example.org/a)."},
		{"not a URL", "ftp://example.org", "paragraph ftp://example.org"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, FromText(tt.text))
			if got := describe(d); got != tt.want {
				t.Errorf("FromText(%q)\n got: %s\nwant: %s", tt.text, got, tt.want)
			}
		})
	}
}

func FuzzFromMarkdown(f *testing.F) {
	for _, s := range []string{
		"## Week\n- [ ] a\n  - [x] b", "> q\n> - a\n> ```\n> x\n> ```", "| a |\n|---|\n| [x](https://e.org) |",
		"*a **b `c` d** e*", "<div>\n\n*x*\n\n</div>", "1. a\n   ```\n   code\n   ```\n2. b",
		"[a\nb](https://e.org)", "- a\n\n      code\n- [ ]", "\\* &#0; &bogus; \x00",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, md string) {
		raw, err := json.Marshal(FromMarkdown(md, nil))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) > MaxDocSize {
			return
		}
		if _, err := ParseDoc(raw); err != nil {
			t.Fatalf("ParseDoc rejects FromMarkdown(%q, nil): %v\n%s", md, err, raw)
		}
	})
}
