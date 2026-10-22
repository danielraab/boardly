package trpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSuperjsonOutput(t *testing.T) {
	at := Date(time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("CET", 3600)))
	type content struct {
		CrawledAt *Date `json:"crawledAt"`
		Skipped   *Date `json:"skipped,omitempty"`
		Hidden    Date  `json:"-"`
	}
	type bookmark struct {
		CreatedAt Date    `json:"createdAt"`
		Content   content `json:"content"`
	}
	type cursor struct {
		CreatedAt Date
	}
	type page struct {
		cursor
		Bookmarks  []bookmark      `json:"bookmarks"`
		NextCursor *cursor         `json:"nextCursor"`
		Stats      Map             `json:"stats"`
		ByKey      map[string]Date `json:"byKey"`
		Plain      string          `json:"plain"`
	}
	out := page{
		cursor:    cursor{at},
		Bookmarks: []bookmark{{CreatedAt: at}, {CreatedAt: at, Content: content{CrawledAt: &at}}},
		Stats:     Map{{"a", 3}, {"b", 2}},
		ByKey:     map[string]Date{"x.y": at},
	}
	for name, tc := range map[string]struct {
		v        any
		json     string
		wantMeta string
	}{
		"nested": {out,
			`{"CreatedAt":"2026-03-01T11:00:00.000Z","bookmarks":[{"createdAt":"2026-03-01T11:00:00.000Z","content":{"crawledAt":null}},{"createdAt":"2026-03-01T11:00:00.000Z","content":{"crawledAt":"2026-03-01T11:00:00.000Z"}}],"nextCursor":null,"stats":[["a",3],["b",2]],"byKey":{"x.y":"2026-03-01T11:00:00.000Z"},"plain":""}`,
			`{"values":{"CreatedAt":["Date"],"bookmarks.0.createdAt":["Date"],"bookmarks.1.content.crawledAt":["Date"],"bookmarks.1.createdAt":["Date"],"byKey.x\\.y":["Date"],"stats":["map"]}}`},
		"pointer to struct": {&cursor{at}, `{"CreatedAt":"2026-03-01T11:00:00.000Z"}`, `{"values":{"CreatedAt":["Date"]}}`},
		"no dates":          {map[string]any{"success": true}, `{"success":true}`, `null`},
		"nil":               {nil, `null`, `null`},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := json.Marshal(tc.v)
			if err != nil || string(got) != tc.json {
				t.Errorf("json = %s (%v)\nwant %s", got, err, tc.json)
			}
			m, _ := json.Marshal(meta(tc.v))
			if string(m) != tc.wantMeta {
				t.Errorf("meta = %s\nwant %s", m, tc.wantMeta)
			}
		})
	}
}

func TestResultCarriesMeta(t *testing.T) {
	s := newServer()
	s.Procedures["when"] = Procedure{Kind: Query, Public: true, Handle: func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]Date{"at": Date(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))}, nil
	}}
	status, body := do(t, s, "GET", query("when", `{}`), "", false)
	want := `[{"result":{"data":{"json":{"at":"2026-03-01T12:00:00.000Z"},"meta":{"values":{"at":["Date"]}}}}}]`
	if status != 200 || body != want {
		t.Errorf("got %d %s\nwant %s", status, body, want)
	}
}
