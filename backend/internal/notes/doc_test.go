package notes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readFixtures(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "docs", dir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures in %s: %v", dir, err)
	}
	out := map[string][]byte{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(f)] = b
	}
	return out
}

func TestParseDocValidFixtures(t *testing.T) {
	for name, raw := range readFixtures(t, "valid") {
		t.Run(name, func(t *testing.T) {
			d, err := ParseDoc(raw)
			if err != nil {
				t.Fatalf("ParseDoc: %v", err)
			}
			// The fixtures are in normal form: parsing them changes nothing.
			var want, got any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("normalized document differs:\n got %s\nwant %s", b, raw)
			}
		})
	}
}

func TestParseDocInvalidFixtures(t *testing.T) {
	for name, raw := range readFixtures(t, "invalid") {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseDoc(raw); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("ParseDoc = %v, want ErrInvalidDocument", err)
			}
		})
	}
}

func TestParseDocTooLarge(t *testing.T) {
	text := strings.Repeat("x", 600*1024)
	raw := `{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"` + text + `"}]}]}`
	if _, err := ParseDoc([]byte(raw)); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("ParseDoc = %v, want ErrInvalidDocument", err)
	}
}

func TestParseDocDefaults(t *testing.T) {
	d, err := ParseDoc([]byte(`{"v":1,"blocks":[
		{"type":"paragraph","content":[{"type":"text","text":""},{"type":"text","text":"a"}]},
		{"type":"code"},
		{"type":"list","style":"task","items":[{"id":"a1","content":[]}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := *d.Blocks[0].Content; len(got) != 1 || *got[0].Text != "a" {
		t.Errorf("empty text run kept: %+v", got)
	}
	if d.Blocks[1].Text == nil || *d.Blocks[1].Text != "" {
		t.Errorf("code without text: %+v", d.Blocks[1])
	}
	if it := (*d.Blocks[2].Items)[0]; it.Checked == nil || *it.Checked {
		t.Errorf("task without checked state: %+v", it)
	}
}

// taskIDs returns the task IDs of a document in order.
func taskIDs(d Doc) []string {
	var ids []string
	var visit func(b *Block)
	visit = func(b *Block) {
		if b.Type != BlockList {
			return
		}
		for _, it := range *b.Items {
			if b.Style == ListTask {
				ids = append(ids, it.ID)
			}
			if it.Children != nil {
				visit(it.Children)
			}
		}
	}
	for i := range d.Blocks {
		visit(&d.Blocks[i])
	}
	return ids
}

func TestTaskIDs(t *testing.T) {
	parse := func(t *testing.T, raw string) []string {
		t.Helper()
		d, err := ParseDoc([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		return taskIDs(d)
	}
	task := func(id string) string {
		if id == "" {
			return `{"content":[{"type":"text","text":"x"}]}`
		}
		return `{"id":"` + id + `","content":[{"type":"text","text":"x"}]}`
	}
	list := func(items ...string) string {
		return `{"type":"list","style":"task","items":[` + strings.Join(items, ",") + `]}`
	}
	doc := func(blocks ...string) string { return `{"v":1,"blocks":[` + strings.Join(blocks, ",") + `]}` }

	t.Run("assigned when missing", func(t *testing.T) {
		ids := parse(t, doc(list(task(""), task(""))))
		if len(ids) != 2 || !taskIDRe.MatchString(ids[0]) || len(ids[0]) != 12 || ids[0] == ids[1] {
			t.Errorf("ids = %q, want two distinct 12-character IDs", ids)
		}
	})
	t.Run("kept when valid", func(t *testing.T) {
		if ids := parse(t, doc(list(task("a1"), task("b_2-x")))); !reflect.DeepEqual(ids, []string{"a1", "b_2-x"}) {
			t.Errorf("ids = %q", ids)
		}
	})
	t.Run("duplicate replaced after the first", func(t *testing.T) {
		ids := parse(t, doc(list(task("a1")), list(task("a1"))))
		if ids[0] != "a1" || ids[1] == "a1" || ids[1] == "" {
			t.Errorf("ids = %q, want a1 and a new ID", ids)
		}
	})
	t.Run("duplicate in a nested list", func(t *testing.T) {
		nested := `{"id":"a1","content":[],"children":` + list(task("a1")) + `}`
		ids := parse(t, doc(list(nested)))
		if len(ids) != 2 || ids[0] != "a1" || ids[1] == "a1" {
			t.Errorf("ids = %q", ids)
		}
	})
	t.Run("only task items get IDs", func(t *testing.T) {
		d, err := ParseDoc([]byte(doc(`{"type":"list","style":"bullet","items":[{"content":[]}]}`)))
		if err != nil {
			t.Fatal(err)
		}
		if it := (*d.Blocks[0].Items)[0]; it.ID != "" || it.Checked != nil {
			t.Errorf("bullet item = %+v, want no ID or checked state", it)
		}
	})
}

func TestAttachmentIDs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "docs", "valid", "attachment.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseDoc(raw)
	if err != nil {
		t.Fatal(err)
	}
	d.Blocks = append(d.Blocks, Block{Type: BlockAttachment, ID: "0b8f1c2e-3d4a-4b5c-8d6e-7f8091a2b3c4"})
	want := []string{"0b8f1c2e-3d4a-4b5c-8d6e-7f8091a2b3c4", "1c9e2d3f-4e5b-4c6d-9e7f-8091a2b3c4d5"}
	if got := AttachmentIDs(d); !reflect.DeepEqual(got, want) {
		t.Fatalf("AttachmentIDs = %v, want %v", got, want)
	}
	if FirstLine(Doc{V: 1, Blocks: []Block{{Type: BlockAttachment, ID: "x"}}}) != "" {
		t.Fatal("an attachment has text")
	}
}

func TestTrailingAttachments(t *testing.T) {
	para := func(s string) Block {
		return Block{Type: BlockParagraph, Content: &[]Inline{{Type: InlineText, Text: &s}}}
	}
	att := func(id string) Block { return Block{Type: BlockAttachment, ID: id} }
	for _, tc := range []struct {
		name     string
		blocks   []Block
		body     []Block
		trailing []string
	}{
		{"photo at the end", []Block{para("roses"), att("a"), att("b")}, []Block{para("roses")}, []string{"a", "b"}},
		{"attachment in the middle", []Block{att("a"), para("roses")}, []Block{att("a"), para("roses")}, []string{}},
		{"only attachments", []Block{att("a")}, []Block{}, []string{"a"}},
		{"duplicates", []Block{para("x"), att("a"), att("a")}, []Block{para("x")}, []string{"a"}},
		{"empty", []Block{}, []Block{}, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, ids := SplitTrailingAttachments(Doc{V: 1, Blocks: tc.blocks})
			if !reflect.DeepEqual(body.Blocks, tc.body) || body.V != 1 {
				t.Fatalf("body = %+v, want %+v", body, tc.body)
			}
			if !reflect.DeepEqual(ids, tc.trailing) {
				t.Fatalf("ids = %v, want %v", ids, tc.trailing)
			}
		})
	}
	got := JoinTrailingAttachments(Doc{V: 1, Blocks: []Block{para("x")}}, []string{"a", "b", "a"})
	want := []Block{para("x"), att("a"), att("b")}
	if !reflect.DeepEqual(got.Blocks, want) {
		t.Fatalf("join = %+v, want %+v", got.Blocks, want)
	}
}

func TestLocation(t *testing.T) {
	d, err := ParseDoc([]byte(`{"v":1,"blocks":[{"type":"location","lat":48.20817634,"lng":16.3738189}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if b := d.Blocks[0]; *b.Lat != 48.208176 || *b.Lng != 16.373819 {
		t.Errorf("not rounded: %v, %v", *b.Lat, *b.Lng)
	}
	label := strings.Repeat("ä", MaxLocationLabel)
	raw := `{"v":1,"blocks":[{"type":"location","lat":1,"lng":2,"label":"` + label + `"}]}`
	if _, err := ParseDoc([]byte(raw)); err != nil {
		t.Errorf("label of %d characters: %v", MaxLocationLabel, err)
	}
	raw = `{"v":1,"blocks":[{"type":"location","lat":1,"lng":2,"label":"` + label + `x"}]}`
	if _, err := ParseDoc([]byte(raw)); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("label of %d characters: %v, want ErrInvalidDocument", MaxLocationLabel+1, err)
	}
}

func TestLocationIsNotText(t *testing.T) {
	label, text := "Stephansplatz", "Meet at 8"
	lat, lng := 48.208176, 16.373819
	d := Doc{V: 1, Blocks: []Block{
		{Type: BlockLocation, Lat: &lat, Lng: &lng, Label: &label},
		{Type: BlockParagraph, Content: &[]Inline{{Type: InlineText, Text: &text}}},
	}}
	if got := DisplayTitle(Note{Content: d}); got == nil || *got != "Meet at 8" {
		t.Fatalf("DisplayTitle = %v, want Meet at 8", got)
	}
}

func TestFormatCoordinate(t *testing.T) {
	for in, want := range map[float64]string{48.2: "48.2", 16.3738189: "16.373819", 0: "0", -0.0000001: "0", -180: "-180"} {
		if got := FormatCoordinate(in); got != want {
			t.Errorf("FormatCoordinate(%v) = %q, want %q", in, got, want)
		}
	}
}
