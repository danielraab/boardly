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
