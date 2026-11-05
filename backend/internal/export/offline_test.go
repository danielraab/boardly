package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"boardly/internal/blobs"
	"boardly/internal/db"
	"boardly/internal/favourites"
)

// linkNode finds the link with the URL in the tree.
func linkNode(n *Node, url string) *Node {
	if n == nil {
		return nil
	}
	if n.Kind == "link" && n.URL == url {
		return n
	}
	for _, c := range n.Children {
		if found := linkNode(c, url); found != nil {
			return found
		}
	}
	return nil
}

func TestOfflineCopies(t *testing.T) {
	f := newFixture(t)
	now := db.FormatTime(f.now)
	favs := favourites.NewStore(f.db)
	links := map[string]string{}
	for _, u := range []string{"https://go.dev/tour", "https://example.org/pending", "https://example.org/failed", "https://example.org/none"} {
		n, err := favs.Create(f.ctx, f.ann.ID, favourites.NewNode{Kind: favourites.KindLink, URL: ptr(u)})
		if err != nil {
			t.Fatal(err)
		}
		links[u] = n.ID
	}
	article, shot, failedArticle := blobs.NewID(), blobs.NewID(), blobs.NewID()
	f.exec(`INSERT INTO offline_copies (node_id, user_id, url, status, title, article_id, screenshot_id, captured_at, updated_at)
		VALUES (?, ?, 'https://go.dev/tour', 'ready', 'A Tour of Go', ?, ?, '2026-10-01T08:00:00Z', ?)`, links["https://go.dev/tour"], f.ann.ID, article, shot, now)
	f.exec(`INSERT INTO offline_copies (node_id, user_id, url, status, updated_at) VALUES (?, ?, 'https://example.org/pending', 'pending', ?)`,
		links["https://example.org/pending"], f.ann.ID, now)
	// A failed refresh keeps the earlier article.
	f.exec(`INSERT INTO offline_copies (node_id, user_id, url, status, error, article_id, captured_at, updated_at)
		VALUES (?, ?, 'https://example.org/failed', 'failed', 'fetch_failed', ?, '2026-09-01T08:00:00Z', ?)`, links["https://example.org/failed"], f.ann.ID, failedArticle, now)
	for id, data := range map[string]string{article: "<p>Tour</p>", shot: "\xff\xd8jpeg", failedArticle: "<p>Old</p>"} {
		if err := f.s.OfflineFiles.Write(id, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}

	doc, err := f.s.Document(f.ctx, f.ann.ID)
	if err != nil {
		t.Fatal(err)
	}
	var plain bytes.Buffer
	if err := WriteJSON(&plain, doc); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), "fetch_failed") || strings.Contains(plain.String(), "pending") && strings.Contains(plain.String(), `"status"`) {
		t.Errorf("document contains the state of offline copies:\n%s", plain.String())
	}
	var withoutFiles Document
	if err := json.Unmarshal(plain.Bytes(), &withoutFiles); err != nil {
		t.Fatal(err)
	}
	tour := linkNode(withoutFiles.Favourites, "https://go.dev/tour").OfflineCopy
	if tour == nil || tour.CapturedAt != "2026-10-01T08:00:00Z" || tour.Title == nil || *tour.Title != "A Tour of Go" || tour.Article != nil || tour.Screenshot != nil {
		t.Fatalf("tour without files = %+v", tour)
	}
	for _, u := range []string{"https://example.org/pending", "https://example.org/none"} {
		if c := linkNode(withoutFiles.Favourites, u).OfflineCopy; c != nil {
			t.Errorf("%s: offlineCopy = %+v, want null", u, c)
		}
	}
	if !strings.Contains(plain.String(), `"offlineCopy": null`) {
		t.Error("links without a copy have no offlineCopy: null")
	}

	var buf bytes.Buffer
	if err := f.s.WriteArchive(&buf, doc); err != nil {
		t.Fatal(err)
	}
	files := readArchive(t, buf.Bytes())
	tourID, failedID := links["https://go.dev/tour"], links["https://example.org/failed"]
	for name, want := range map[string]string{
		"offline/" + tourID + "/article.html":   "<p>Tour</p>",
		"offline/" + tourID + "/screenshot.jpg": "\xff\xd8jpeg",
		"offline/" + failedID + "/article.html": "<p>Old</p>",
	} {
		if string(files[name]) != want {
			t.Errorf("%s = %q, want %q", name, files[name], want)
		}
	}
	var got Document
	if err := json.Unmarshal(files[DocumentName], &got); err != nil {
		t.Fatal(err)
	}
	c := linkNode(got.Favourites, "https://go.dev/tour").OfflineCopy
	if c == nil || c.Article == nil || *c.Article != "offline/"+tourID+"/article.html" || c.Screenshot == nil || *c.Screenshot != "offline/"+tourID+"/screenshot.jpg" {
		t.Fatalf("tour in the archive = %+v", c)
	}
	if c := linkNode(got.Favourites, "https://example.org/failed").OfflineCopy; c == nil || c.Article == nil || c.Screenshot != nil {
		t.Fatalf("failed in the archive = %+v", c)
	}

	// A missing file leaves its path null.
	if err := f.s.OfflineFiles.Remove(shot); err != nil {
		t.Fatal(err)
	}
	doc, _ = f.s.Document(f.ctx, f.ann.ID)
	buf.Reset()
	if err := f.s.WriteArchive(&buf, doc); err != nil {
		t.Fatal(err)
	}
	if c := linkNode(doc.Favourites, "https://go.dev/tour").OfflineCopy; c.Screenshot != nil || c.Article == nil {
		t.Fatalf("missing screenshot = %+v", c)
	}
}
