package importer

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"boardly/internal/favourites"
)

func parseFile(t *testing.T, format Format, name string) favourites.ImportTree {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tree, ok, err := Parse(format, f)
	if !ok || err != nil {
		t.Fatalf("parse %s: ok=%v err=%v", name, ok, err)
	}
	return tree
}

// render prints the items as an indented outline: folders end with "/",
// links show their title, URL, tags, description and date.
func render(items []favourites.ImportItem) string {
	var sb strings.Builder
	var walk func(items []favourites.ImportItem, indent string)
	walk = func(items []favourites.ImportItem, indent string) {
		for _, it := range items {
			if it.Folder != nil {
				fmt.Fprintf(&sb, "%s%s/\n", indent, it.Folder.Name)
				walk(it.Folder.Items, indent+"  ")
				continue
			}
			l := it.Link
			title, desc, date := "-", "", ""
			if l.Title != nil {
				title = *l.Title
			}
			if l.Description != nil {
				desc = " " + fmt.Sprintf("%q", *l.Description)
			}
			if !l.CreatedAt.IsZero() {
				date = " @" + l.CreatedAt.Format(time.RFC3339)
			}
			fmt.Fprintf(&sb, "%s%s <%s> %v%s%s\n", indent, title, l.URL, l.Tags, desc, date)
		}
	}
	walk(items, "")
	return sb.String()
}

func check(t *testing.T, tree favourites.ImportTree, want string, report favourites.ImportReport) {
	t.Helper()
	if got := render(tree.Root.Items); got != want {
		t.Errorf("tree =\n%s\nwant\n%s", got, want)
	}
	if tree.Report != report {
		t.Errorf("report = %+v, want %+v", tree.Report, report)
	}
}

func TestKarakeep(t *testing.T) {
	tree := parseFile(t, FormatKarakeep, "karakeep.json")
	check(t, tree, `Dev/
  Docker/
    Containers/
      Go by Example <https://gobyexample.com/> [go Learning] @2026-10-22T12:01:22Z
Linux/
  Linux docs <https://kernel.org> [linux] "Summary: The kernel" @2026-10-09T22:12:17Z
Floccus/
  dev/
    - <https://example.org/video?v=1&t=2> [] "Note: login with the family account" @2026-10-20T16:27:53Z
Empty/
In a smart list only <https://example.org/top> [] @2026-10-10T11:42:06Z
`, favourites.ImportReport{Skipped: favourites.ImportSkipped{InvalidURLs: 1, NotLinks: 1, SmartLists: 1}})
}

func TestKarakeepParentCycle(t *testing.T) {
	tree, err := ParseKarakeep(strings.NewReader(`{"bookmarks": [], "lists": [
		{"id": "a", "name": "A", "type": "manual", "parentId": "b"},
		{"id": "b", "name": "B", "type": "manual", "parentId": "a"},
		{"id": "c", "name": "  ", "type": "manual", "parentId": null},
		{"id": "d", "name": "D", "type": "manual", "parentId": "c"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	check(t, tree, "A/\nB/\nD/\n", favourites.ImportReport{})
}

func TestKarakeepHTML(t *testing.T) {
	tree := parseFile(t, FormatNetscape, "karakeep.html")
	check(t, tree, `Search results for "a" – Example <https://example.org/search?q=a&b=1> [] @2026-10-22T12:01:22Z
https://floorplan.example/ <https://floorplan.example/> [] @2026-10-17T06:55:58Z
Sensor & Co <https://shop.example/sensor> [Heizung 4-20mA Pressure Sensors] @2026-09-25T19:48:21Z
`, favourites.ImportReport{})
}

func TestBrowserHTML(t *testing.T) {
	tree := parseFile(t, FormatNetscape, "browser.html")
	check(t, tree, `Toolbar/
  Go <https://go.dev/> [] "A tour of Go" @2023-11-14T22:15:00Z
  Dev/
    Dev link <https://example.org/dev> []
  After Dev <https://example.org/after> []
Top <https://example.org/top> [] @2023-11-14T22:16:40Z
`, favourites.ImportReport{})
}

func TestLinkwarden(t *testing.T) {
	tree := parseFile(t, FormatLinkwarden, "linkwarden.json")
	check(t, tree, `Reading/
  Later/
    Rust book <https://doc.rust-lang.org/book/> [rust] @2025-05-01T10:00:00Z
  Go tour <https://go.dev/tour> [go docs] "A tour of Go" @2025-04-01T09:30:00Z
`, favourites.ImportReport{Skipped: favourites.ImportSkipped{NotLinks: 1}})
}

func TestInvalidFiles(t *testing.T) {
	html, err := os.ReadFile("testdata/browser.html")
	if err != nil {
		t.Fatal(err)
	}
	karakeep, err := os.ReadFile("testdata/karakeep.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		format Format
		file   string
	}{
		{"HTML as Karakeep", FormatKarakeep, string(html)},
		{"HTML as Linkwarden", FormatLinkwarden, string(html)},
		{"Karakeep as Linkwarden", FormatLinkwarden, string(karakeep)},
		{"Karakeep JSON as HTML", FormatNetscape, string(karakeep)},
		{"JSON array as Karakeep", FormatKarakeep, `[]`},
		{"empty HTML", FormatNetscape, `<html><body><p>Hello</p></body></html>`},
		{"empty file", FormatKarakeep, ``},
	} {
		if _, ok, err := Parse(tt.format, strings.NewReader(tt.file)); !ok || !errors.Is(err, ErrInvalidFile) {
			t.Errorf("%s: ok=%v err=%v, want ErrInvalidFile", tt.name, ok, err)
		}
	}
	if _, ok, _ := Parse("pocket", strings.NewReader("")); ok {
		t.Error("unknown format accepted")
	}
}

func TestDescriptionIsCut(t *testing.T) {
	b := &builder{}
	d := b.describe(strings.Repeat("ä", 2000), strings.Repeat("n", 1000), "")
	if d == nil || utf8.RuneCountInString(*d) != favourites.MaxDescriptionLength || !strings.HasSuffix(*d, "…") {
		t.Fatalf("description has %d characters", utf8.RuneCountInString(*d))
	}
	if b.report.ShortenedDescriptions != 1 {
		t.Fatalf("shortened = %d, want 1", b.report.ShortenedDescriptions)
	}
	if d := b.describe("A tour of Go", "Read later", ""); d == nil || *d != "A tour of Go\n\nNote: Read later" {
		t.Fatalf("description = %v", d)
	}
}

func TestLongValuesAreCut(t *testing.T) {
	b := &builder{}
	l := b.link(entry{url: "https://example.org", title: strings.Repeat("t", 1200), tags: []string{strings.Repeat("x", 101), "ok"}})
	if utf8.RuneCountInString(*l.Title) != favourites.MaxLinkTitleLength || len(l.Tags) != 1 || l.Tags[0] != "ok" {
		t.Fatalf("link = title %d chars, tags %v", utf8.RuneCountInString(*l.Title), l.Tags)
	}
	if got := folderName(strings.Repeat("f", 150)); utf8.RuneCountInString(got) != favourites.MaxFolderNameLength {
		t.Fatalf("folder name has %d characters", utf8.RuneCountInString(got))
	}
}
