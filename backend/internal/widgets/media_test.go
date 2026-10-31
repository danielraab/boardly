package widgets

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"boardly/internal/dashboards"
	"boardly/internal/media"
	"boardly/internal/notes"
)

func mediaWidget(config string) dashboards.Widget {
	return dashboards.Widget{Kind: KindMedia, Width: 2, Config: json.RawMessage(config)}
}

func (f *fixture) station(userID, title string, icon *notes.Icon) media.Source {
	f.t.Helper()
	src, err := f.media.Create(f.ctx, userID, media.NewSource{Kind: media.KindStream, Title: title, URL: "https://stream.example/" + title, Icon: icon})
	if err != nil {
		f.t.Fatal(err)
	}
	return src
}

func TestMediaConfig(t *testing.T) {
	f := newFixture(t)
	if err := f.save(mediaWidget(`{"sourceIds":["gone","b"],"title":"  Radio ","layout":"tiles","size":"L","showIfPublic":true}`)); err != nil {
		t.Fatalf("missing sources: %v", err)
	}
	got, _ := json.Marshal(f.lookup(f.ann)[0]["config"])
	want := `{"layout":"tiles","showIfPublic":true,"size":"L","sourceIds":["gone","b"],"title":"Radio"}`
	if string(got) != want {
		t.Fatalf("config = %s, want %s", got, want)
	}
	fifty := make([]string, maxMediaSources)
	for i := range fifty {
		fifty[i] = fmt.Sprintf("s%d", i)
	}
	ids, _ := json.Marshal(fifty)
	if err := f.save(mediaWidget(`{"sourceIds":` + string(ids) + `,"layout":"list","size":"M"}`)); err != nil {
		t.Fatalf("50 sources: %v", err)
	}
	tooMany, _ := json.Marshal(append(fifty, "one more"))
	for name, config := range map[string]string{
		"no sources":    `{"layout":"list","size":"M"}`,
		"empty sources": `{"sourceIds":[],"layout":"list","size":"M"}`,
		"51 sources":    `{"sourceIds":` + string(tooMany) + `,"layout":"list","size":"M"}`,
		"empty ID":      `{"sourceIds":["a",""],"layout":"list","size":"M"}`,
		"long ID":       `{"sourceIds":["` + strings.Repeat("a", 65) + `"],"layout":"list","size":"M"}`,
		"duplicate":     `{"sourceIds":["a","a"],"layout":"list","size":"M"}`,
		"layout grid":   `{"sourceIds":["a"],"layout":"grid","size":"M"}`,
		"no layout":     `{"sourceIds":["a"],"size":"M"}`,
		"size XL":       `{"sourceIds":["a"],"layout":"list","size":"XL"}`,
		"unknown field": `{"sourceIds":["a"],"layout":"list","size":"M","autoplay":true}`,
		"long title":    `{"sourceIds":["a"],"title":"` + strings.Repeat("a", 201) + `","layout":"list","size":"M"}`,
	} {
		if err := f.save(mediaWidget(config)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v, want ErrInvalidLayout", name, err)
		}
	}
}

func TestMediaResolve(t *testing.T) {
	f := newFixture(t)
	fm4 := f.station(f.ann, "FM4", &notes.Icon{Type: notes.IconSymbol, Name: "music"})
	jazz := f.station(f.ann, "Jazz", nil)
	f.station(f.ann, "Secret", nil)
	gone := f.station(f.ann, "Gone", nil)
	bobs := f.station(f.bob, "Bobs", nil)
	if err := f.media.Delete(f.ctx, f.ann, gone.ID); err != nil {
		t.Fatal(err)
	}
	cfg := func(public bool, ids ...string) dashboards.Widget {
		raw, _ := json.Marshal(mediaConfig{SourceIDs: ids, Layout: "list", Size: "M", ShowIfPublic: public})
		return mediaWidget(string(raw))
	}
	if err := f.save(
		cfg(false, fm4.ID),
		cfg(true, jazz.ID, gone.ID, fm4.ID, bobs.ID),
		cfg(true, gone.ID, bobs.ID),
	); err != nil {
		t.Fatal(err)
	}
	wantData := canonical(t, `{"sources":[`+
		`{"id":"`+jazz.ID+`","kind":"stream","title":"Jazz","url":"https://stream.example/Jazz","icon":null},`+
		`{"id":"`+fm4.ID+`","kind":"stream","title":"FM4","url":"https://stream.example/FM4","icon":{"type":"symbol","name":"music"}}]}`)

	owner := f.lookup(f.ann)
	if len(owner) != 3 {
		t.Fatalf("owner sees %d widgets, want 3", len(owner))
	}
	if data, _ := json.Marshal(owner[1]["data"]); string(data) != wantData {
		t.Errorf("owner data =\n%s\nwant\n%s", data, wantData)
	}
	if data, _ := json.Marshal(owner[2]["data"]); string(data) != `{"sources":[]}` {
		t.Errorf("all deleted data = %s", data)
	}

	// Visitors get only the public widget with existing sources, and only
	// its sources.
	for _, viewer := range []string{"", f.bob} {
		visitor := f.lookup(viewer)
		if len(visitor) != 1 {
			t.Fatalf("visitor %q sees %d widgets, want 1", viewer, len(visitor))
		}
		data, _ := json.Marshal(visitor[0]["data"])
		if string(data) != wantData {
			t.Errorf("visitor %q data =\n%s\nwant\n%s", viewer, data, wantData)
		}
		if strings.Contains(string(data), "Secret") || strings.Contains(string(data), "position") {
			t.Errorf("visitor %q gets more than the widget shows: %s", viewer, data)
		}
	}
}
