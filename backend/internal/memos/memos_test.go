package memos

import (
	"encoding/json"
	"strings"
	"testing"

	"boardly/internal/notes"
)

func TestMatchesFilter(t *testing.T) {
	for filter, want := range map[string]bool{
		"":                                      true,
		`creator == "users/u1"`:                 true,
		`creator == "users/u2"`:                 false,
		`visibility in ["PUBLIC", "PROTECTED"]`: false,
		`visibility in ["PRIVATE"]`:             true,
		`visibility == "PRIVATE"`:               true,
		`visibility == PUBLIC`:                  false,
		`creator == "users/u1" && visibility in ["PUBLIC"]`: false,
		`creator == "users/u1" && tag in ["x"]`:             true,
	} {
		if got := matchesFilter(filter, "u1"); got != want {
			t.Errorf("%s = %v, want %v", filter, got, want)
		}
	}
}

func TestUpdateFromBody(t *testing.T) {
	body := map[string]json.RawMessage{
		"content": json.RawMessage(`"text"`), "pinned": json.RawMessage(`true`),
		"state": json.RawMessage(`"ARCHIVED"`), "visibility": json.RawMessage(`"PUBLIC"`), "location": json.RawMessage(`null`),
	}
	in, err := UpdateFromBody(body, "")
	if err != nil || in.Content == nil || in.Pinned == nil || in.State == nil {
		t.Fatalf("without a mask = %+v, %v", in, err)
	}
	in, err = UpdateFromBody(body, "pinned,update_time")
	if err != nil || in.Content != nil || in.Pinned == nil || in.State != nil {
		t.Errorf("with a mask = %+v, %v", in, err)
	}
	if _, err := UpdateFromBody(map[string]json.RawMessage{"pinned": json.RawMessage(`"yes"`)}, ""); err == nil {
		t.Error("a string as pinned was accepted")
	}
}

func TestUpdateLocationFromBody(t *testing.T) {
	cafe := json.RawMessage(`{"placeholder":"Café Central","latitude":48.2104,"longitude":16.3655}`)
	for _, tc := range []struct {
		name     string
		body     map[string]json.RawMessage
		mask     string
		set, has bool
	}{
		{"not sent", map[string]json.RawMessage{"content": json.RawMessage(`"x"`)}, "", false, false},
		{"set", map[string]json.RawMessage{"location": cafe}, "", true, true},
		{"null removes", map[string]json.RawMessage{"location": json.RawMessage(`null`)}, "", true, false},
		{"masked without a value removes", map[string]json.RawMessage{}, "location", true, false},
		{"masked set", map[string]json.RawMessage{"location": cafe}, "content,location", true, true},
		{"not in the mask", map[string]json.RawMessage{"location": cafe}, "content", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, err := UpdateFromBody(tc.body, tc.mask)
			if err != nil || in.SetLocation != tc.set || (in.Location != nil) != tc.has {
				t.Errorf("= %+v, %v", in, err)
			}
		})
	}
	for _, bad := range []string{`{"latitude":95,"longitude":0}`, `{"latitude":0,"longitude":-181}`, `{"latitude":48}`, `{"latitude":"48","longitude":16}`, `"here"`} {
		if _, err := UpdateFromBody(map[string]json.RawMessage{"location": json.RawMessage(bad)}, ""); err != errLocation {
			t.Errorf("location %s: err = %v", bad, err)
		}
	}
}

func TestLocationBlock(t *testing.T) {
	long := strings.Repeat("é", notes.MaxLocationLabel+10)
	for _, tc := range []struct {
		raw, label string
		lat, lng   float64
	}{
		{`{"placeholder":"  Café\nCentral ","latitude":48.21043219,"longitude":-0.0000001}`, "Café Central", 48.210432, 0},
		{`{"placeholder":"","latitude":48.2,"longitude":16.37}`, "", 48.2, 16.37},
		{`{"latitude":-90,"longitude":180}`, "", -90, 180},
		{`{"placeholder":"` + long + `","latitude":1,"longitude":2}`, long[:2*notes.MaxLocationLabel], 1, 2},
	} {
		b, err := locationBlock(json.RawMessage(tc.raw))
		if err != nil || b.Type != notes.BlockLocation || *b.Lat != tc.lat || *b.Lng != tc.lng {
			t.Fatalf("%s = %+v, %v", tc.raw, b, err)
		}
		if label := locationOf(b).Placeholder; label != tc.label || (tc.label == "") != (b.Label == nil) {
			t.Errorf("%s: label %q, want %q", tc.raw, label, tc.label)
		}
	}
	if b, err := locationBlock(json.RawMessage(`null`)); b != nil || err != nil {
		t.Errorf("null = %+v, %v", b, err)
	}
}

func TestMemoLocation(t *testing.T) {
	text := func(s string) notes.Block {
		return notes.Block{Type: notes.BlockParagraph, Content: &[]notes.Inline{{Type: notes.InlineText, Text: &s}}}
	}
	loc := func(label string, lat, lng float64) notes.Block {
		b := notes.Block{Type: notes.BlockLocation, Lat: &lat, Lng: &lng}
		if label != "" {
			b.Label = &label
		}
		return b
	}
	trip := "Trip"
	groceries := "Groceries"
	for _, tc := range []struct {
		name    string
		title   *string
		blocks  []notes.Block
		content string
		want    *Location
	}{
		{"location at the end", &trip, []notes.Block{text("Meet here"), loc("Stephansplatz", 48.208176, 16.373819), {Type: notes.BlockAttachment, ID: "a1"}},
			"# Trip\n\nMeet here", &Location{"Stephansplatz", 48.208176, 16.373819}},
		{"location in the middle", nil, []notes.Block{loc("Stephansplatz", 48.208176, 16.373819), text("Meet here")},
			"[Stephansplatz](geo:48.208176,16.373819)\n\nMeet here", nil},
		{"location without label", nil, []notes.Block{loc("", 48.2, 16.37)}, "", &Location{"", 48.2, 16.37}},
		{"two locations at the end", nil, []notes.Block{loc("A", 1, 2), loc("B", 3, 4)}, "[A](geo:1,2)", &Location{"B", 3, 4}},
		{"titled note", &groceries, []notes.Block{text("milk")}, "# Groceries\n\nmilk", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := notes.Note{ID: "n1", Title: tc.title, Content: notes.Doc{V: 1, Blocks: tc.blocks}}
			m := MemoOf("u1", "https://boardly.example", n)
			if m.Content != tc.content || (m.Location == nil) != (tc.want == nil) || (tc.want != nil && *m.Location != *tc.want) {
				t.Errorf("memo = %q, %+v; want %q, %+v", m.Content, m.Location, tc.content, tc.want)
			}
			raw, _ := json.Marshal(m)
			if strings.Contains(string(raw), `"location"`) != (tc.want != nil) {
				t.Errorf("JSON %s", raw)
			}
		})
	}
}

func TestPageToken(t *testing.T) {
	if n, err := pageOffset(pageToken(200)); err != nil || n != 200 {
		t.Errorf("round trip = %d, %v", n, err)
	}
	for _, bad := range []string{"!", pageToken(-1)} {
		if _, err := pageOffset(bad); err == nil {
			t.Errorf("token %q accepted", bad)
		}
	}
}
