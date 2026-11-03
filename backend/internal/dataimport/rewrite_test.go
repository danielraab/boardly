package dataimport

import (
	"encoding/json"
	"testing"
)

func TestRewriteIDs(t *testing.T) {
	ids := map[string]string{"old-folder": "new-folder", "old-note": "new-note"}
	in := json.RawMessage(`{"height":3,"config":{"folderId":"old-folder","noteIds":["old-note","other"],"title":"old-folder note","big":12345678901234567890,"old-folder":"key stays"}}`)
	out, err := rewriteIDs(in, ids)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Height int `json:"height"`
		Config struct {
			FolderID string          `json:"folderId"`
			NoteIDs  []string        `json:"noteIds"`
			Title    string          `json:"title"`
			Big      json.RawMessage `json:"big"`
			Key      string          `json:"old-folder"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	c := got.Config
	if got.Height != 3 || c.FolderID != "new-folder" || c.NoteIDs[0] != "new-note" || c.NoteIDs[1] != "other" ||
		c.Title != "old-folder note" || string(c.Big) != "12345678901234567890" || c.Key != "key stays" {
		t.Errorf("rewritten = %s", out)
	}

	same := json.RawMessage(`{"folderId": "unknown"}`)
	if out, err := rewriteIDs(same, ids); err != nil || string(out) != string(same) {
		t.Errorf("unchanged input = %s, %v", out, err)
	}
}

func TestStringsIn(t *testing.T) {
	var got []string
	stringsIn(json.RawMessage(`{"a":["x",{"b":"y"}],"n":1}`), func(s string) { got = append(got, s) })
	if len(got) != 2 {
		t.Errorf("strings = %v", got)
	}
}
