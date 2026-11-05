package offline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"boardly/internal/blobs"
	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/jobs"
	"boardly/internal/users"
)

type fixture struct {
	t    *testing.T
	ctx  context.Context
	s    *Store
	c    *Capturer
	favs *favourites.Store
	user string
}

func newFixture(t *testing.T, allowPrivate bool) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	s, err := NewStore(database, filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	u, err := users.NewStore(database).EnsureByEmail(context.Background(), database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	queue := jobs.New(database)
	favs := favourites.NewStore(database)
	favs.Jobs = queue
	c := &Capturer{
		Store: s, Jobs: queue,
		Client:    &http.Client{Transport: images.GuardedTransport(allowPrivate)},
		MaxBytes:  1 << 20,
		Converter: images.Converter{MaxPixels: 25_000_000, Width: ScreenshotWidth},
	}
	return &fixture{t: t, ctx: context.Background(), s: s, c: c, favs: favs, user: u.ID}
}

func (f *fixture) link(rawURL string) string {
	f.t.Helper()
	n, err := f.favs.Create(f.ctx, f.user, favourites.NewNode{Kind: favourites.KindLink, URL: &rawURL})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

// runCaptures runs the pending capture jobs and deletes them, as the
// queue's workers would.
func (f *fixture) runCaptures() int {
	f.t.Helper()
	rows, err := f.s.DB.Query(`SELECT id, payload FROM jobs WHERE kind = 'offline_capture' ORDER BY id`)
	if err != nil {
		f.t.Fatal(err)
	}
	var pending []jobs.Job
	for rows.Next() {
		j := jobs.Job{Kind: jobs.KindOfflineCapture}
		var payload string
		if err := rows.Scan(&j.ID, &payload); err != nil {
			f.t.Fatal(err)
		}
		j.Payload = json.RawMessage(payload)
		pending = append(pending, j)
	}
	rows.Close()
	for _, j := range pending {
		if err := f.c.capture(f.ctx, j); err != nil {
			f.t.Fatalf("capture %s: %v", j.Payload, err)
		}
		if _, err := f.s.DB.Exec(`DELETE FROM jobs WHERE id = ?`, j.ID); err != nil {
			f.t.Fatal(err)
		}
	}
	return len(pending)
}

func (f *fixture) get(id string) Copy {
	f.t.Helper()
	c, err := f.s.Get(f.ctx, f.user, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) read(id string, part Part) string {
	f.t.Helper()
	file, err := f.s.Open(f.ctx, f.user, id, part)
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

var paragraph = strings.Repeat("Go is an open source programming language that makes it simple to build secure, scalable systems. ", 4)

// articlePage is a page with a readable article, a relative image, a script
// and an event handler.
func articlePage(head string) string {
	return `<!doctype html><html><head><title>Go &amp; you</title>` + head + `</head><body>
<nav><a href="/">Home</a></nav>
<article><h1>Generics</h1>
<p>` + paragraph + `</p>
<p onclick="alert(1)">` + paragraph + `<a href="/doc">docs</a> <a href="javascript:alert(2)">bad</a></p>
<img src="/img/diagram.png" alt="Diagram" onerror="alert(3)">
<script>alert(4)</script>
<iframe src="https://evil.example"></iframe>
<form action="/x"><input name="q"></form>
<p>` + paragraph + `</p>
</article></body></html>`
}

// site serves pages by path and counts the requests for /img/.
type site struct {
	*httptest.Server
	images atomic.Int32
}

func newSite(t *testing.T, pages map[string]func(w http.ResponseWriter)) *site {
	s := &site{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/img/") {
			s.images.Add(1)
		}
		if h, ok := pages[r.URL.Path]; ok {
			h(w)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func htmlPage(body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}
}

func TestCaptureArticle(t *testing.T) {
	f := newFixture(t, true)
	srv := newSite(t, map[string]func(http.ResponseWriter){"/post": htmlPage(articlePage(""))})
	id := f.link(srv.URL + "/post")
	if c := f.get(id); c.Status != StatusPending {
		t.Fatalf("status before the capture = %q, want pending", c.Status)
	}
	if n := f.runCaptures(); n != 1 {
		t.Fatalf("captures = %d", n)
	}
	c := f.get(id)
	if c.Status != StatusReady || !c.HasArticle || c.HasScreenshot || c.CapturedAt == nil || c.Error != nil {
		t.Fatalf("copy = %+v", c)
	}
	if c.Title == nil || *c.Title != "Go & you" {
		t.Fatalf("title = %v", c.Title)
	}
	article := f.read(id, PartArticle)
	for _, want := range []string{
		`<img src="` + srv.URL + `/img/diagram.png"`, `referrerpolicy="no-referrer"`, `alt="Diagram"`,
		`<meta name="referrer" content="no-referrer">`, "Generics", "open source programming language",
		`href="` + srv.URL + `/doc"`, `target="_blank"`,
	} {
		if !strings.Contains(article, want) {
			t.Errorf("article lacks %q:\n%s", want, article)
		}
	}
	for _, bad := range []string{"<script", "onclick", "onerror", "javascript:", "<iframe", "<form", "<input", "alert("} {
		if strings.Contains(article, bad) {
			t.Errorf("article contains %q", bad)
		}
	}
	if n := srv.images.Load(); n != 0 {
		t.Errorf("image requests = %d, want none", n)
	}
	if _, err := f.s.Open(f.ctx, f.user, id, PartScreenshot); !errors.Is(err, ErrNotFound) {
		t.Errorf("screenshot = %v, want ErrNotFound", err)
	}
}

func TestCaptureReasons(t *testing.T) {
	big := strings.Repeat("<p>"+paragraph+"</p>", 3000)
	srv := newSite(t, map[string]func(http.ResponseWriter){
		"/pdf": func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = io.WriteString(w, "%PDF-1.7")
		},
		"/big":   htmlPage("<html><body><article>" + big + "</article></body></html>"),
		"/short": htmlPage("<html><head><title>Hi</title></head><body><p>Just one sentence.</p></body></html>"),
	})
	for path, want := range map[string]string{
		"/missing": ReasonFetchFailed,
		"/pdf":     ReasonNotHTML,
		"/big":     ReasonTooLarge,
		"/short":   ReasonNoContent,
	} {
		f := newFixture(t, true)
		id := f.link(srv.URL + path)
		f.runCaptures()
		c := f.get(id)
		if c.Status != StatusFailed || c.Error == nil || *c.Error != want || c.HasArticle || c.CapturedAt != nil {
			t.Errorf("%s: copy = %+v, error %v; want failed with %s", path, c, c.Error, want)
		}
	}

	// Private addresses aren't loaded, by the server or the browser.
	f := newFixture(t, false)
	browser := &fakeBrowser{}
	f.c.Browser = browser
	id := f.link(srv.URL + "/short")
	f.runCaptures()
	if c := f.get(id); c.Status != StatusFailed || *c.Error != ReasonBlockedAddress {
		t.Errorf("private address: copy = %+v", c)
	}
	if browser.calls != 0 {
		t.Errorf("browser calls for a blocked page = %d", browser.calls)
	}
}

func TestStoreGet(t *testing.T) {
	f := newFixture(t, true)
	folder, err := f.favs.Create(f.ctx, f.user, favourites.NewNode{Kind: favourites.KindFolder, Title: ptr("Work")})
	if err != nil {
		t.Fatal(err)
	}
	other, err := users.NewStore(f.s.DB).EnsureByEmail(f.ctx, f.s.DB, "bob@example.org")
	if err != nil {
		t.Fatal(err)
	}
	id := f.link("https://example.org/a")
	for name, tt := range map[string]struct{ user, id string }{
		"folder":       {f.user, folder.ID},
		"unknown":      {f.user, "missing"},
		"other's link": {other.ID, id},
	} {
		if _, err := f.s.Get(f.ctx, tt.user, tt.id); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}

	// Imported links have no copy.
	if _, err := f.favs.Import(f.ctx, f.user, folder.ID, favourites.ImportTree{Root: favourites.ImportFolder{
		Items: []favourites.ImportItem{{Link: &favourites.ImportLink{URL: "https://example.org/imported"}}},
	}}, true); err != nil {
		t.Fatal(err)
	}
	var imported string
	if err := f.s.DB.QueryRow(`SELECT id FROM favourite_nodes WHERE url = 'https://example.org/imported'`).Scan(&imported); err != nil {
		t.Fatal(err)
	}
	if c := f.get(imported); c.Status != StatusNone || c.HasArticle || c.Error != nil {
		t.Errorf("imported link: %+v", c)
	}

	// A pending copy whose job was dropped reads as failed.
	if _, err := f.s.DB.Exec(`DELETE FROM jobs WHERE kind = 'offline_capture'`); err != nil {
		t.Fatal(err)
	}
	if c := f.get(id); c.Status != StatusFailed || c.Error == nil || *c.Error != ReasonFetchFailed {
		t.Errorf("dropped job: %+v", c)
	}
}

func TestCaptureKeepsPartsOnFailedRefresh(t *testing.T) {
	f := newFixture(t, true)
	up := true
	srv := newSite(t, map[string]func(http.ResponseWriter){"/post": func(w http.ResponseWriter) {
		if !up {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		htmlPage(articlePage(""))(w)
	}})
	id := f.link(srv.URL + "/post")
	f.runCaptures()
	first := f.get(id)
	article := f.read(id, PartArticle)

	up = false
	if err := f.favs.QueueOfflineCapture(f.ctx, f.s.DB, f.user, id, srv.URL+"/post", false); err != nil {
		t.Fatal(err)
	}
	if c := f.get(id); c.Status != StatusPending || !c.HasArticle {
		t.Fatalf("pending refresh = %+v, want pending with the old article", c)
	}
	f.runCaptures()
	c := f.get(id)
	if c.Status != StatusFailed || *c.Error != ReasonFetchFailed || !c.HasArticle || !c.CapturedAt.Equal(*first.CapturedAt) {
		t.Fatalf("after failed refresh = %+v", c)
	}
	if got := f.read(id, PartArticle); got != article {
		t.Fatal("article changed by a failed refresh")
	}
}

func TestCaptureDiscardedAfterChange(t *testing.T) {
	f := newFixture(t, true)
	srv := newSite(t, map[string]func(http.ResponseWriter){
		"/old": htmlPage(articlePage("")),
		"/new": htmlPage("<html><body><p>short</p></body></html>"),
	})
	id := f.link(srv.URL + "/old")

	// The URL changes while the capture for the old one runs.
	stored, err := f.s.Record(f.ctx, id, srv.URL+"/old", Result{Article: []byte("x")})
	if err != nil || !stored {
		t.Fatalf("record = %v, %v", stored, err)
	}
	if _, err := f.favs.Update(f.ctx, f.user, id, favourites.Patch{URL: ptr(srv.URL + "/new")}); err != nil {
		t.Fatal(err)
	}
	if stored, err := f.s.Record(f.ctx, id, srv.URL+"/old", Result{Article: []byte("late")}); err != nil || stored {
		t.Fatalf("late result for the old URL stored = %v, %v", stored, err)
	}
	if c := f.get(id); c.Status != StatusPending || c.HasArticle {
		t.Fatalf("after URL change = %+v", c)
	}
	f.runCaptures()
	if c := f.get(id); c.Status != StatusFailed || *c.Error != ReasonNoContent {
		t.Fatalf("copy of the new URL = %+v", c)
	}
	entries, _ := filepath.Glob(filepath.Join(string(f.s.Files), "*"))
	if len(entries) != 1 {
		t.Fatalf("files = %v, want only the first article (removed later by the cleanup)", entries)
	}

	// A link deleted before its capture ran gets nothing.
	gone := f.link(srv.URL + "/old")
	if err := f.favs.Delete(f.ctx, f.user, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	f.runCaptures()
	var rows int
	if err := f.s.DB.QueryRow(`SELECT COUNT(*) FROM offline_copies WHERE node_id = ?`, gone).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("rows of the deleted link = %d, %v", rows, err)
	}
}

// fakeBrowser returns a PNG of the viewport, or err.
type fakeBrowser struct {
	err   error
	calls int
	urls  []string
}

func (b *fakeBrowser) Screenshot(_ context.Context, pageURL string) ([]byte, error) {
	b.calls++
	b.urls = append(b.urls, pageURL)
	if b.err != nil {
		return nil, b.err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, ScreenshotWidth, ScreenshotHeight))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fakePreviews records the links it was asked to fill.
type fakePreviews struct{ filled []string }

func (p *fakePreviews) Fill(_ context.Context, _, nodeID, _ string, data []byte) error {
	if _, err := images.PreviewType(data); err != nil {
		return err
	}
	p.filled = append(p.filled, nodeID)
	return nil
}

func TestCaptureScreenshot(t *testing.T) {
	f := newFixture(t, true)
	browser, previews := &fakeBrowser{}, &fakePreviews{}
	f.c.Browser, f.c.Previews = browser, previews
	srv := newSite(t, map[string]func(http.ResponseWriter){
		"/plain": htmlPage(articlePage("")),
		"/og":    htmlPage(articlePage(`<meta property="og:image" content="/img/card.png">`)),
		"/short": htmlPage("<html><body><p>short</p></body></html>"),
	})
	plain, og, short := f.link(srv.URL+"/plain"), f.link(srv.URL+"/og"), f.link(srv.URL+"/short")
	f.runCaptures()

	for _, id := range []string{plain, og, short} {
		if c := f.get(id); c.Status != StatusReady || !c.HasScreenshot {
			t.Errorf("copy %s = %+v, want ready with a screenshot", id, c)
		}
	}
	if c := f.get(short); c.HasArticle {
		t.Error("short page has an article")
	}
	shot := f.read(plain, PartScreenshot)
	cfg, format, err := image.DecodeConfig(strings.NewReader(shot))
	if err != nil || format != "jpeg" || cfg.Width != ScreenshotWidth || cfg.Height != ScreenshotHeight {
		t.Fatalf("screenshot = %s %dx%d, %v", format, cfg.Width, cfg.Height, err)
	}
	// Pages without a preview image get the screenshot as preview.
	if len(previews.filled) != 2 || previews.filled[0] != plain || previews.filled[1] != short {
		t.Fatalf("filled previews = %v, want %s and %s", previews.filled, plain, short)
	}

	// A refresh never sets the preview.
	previews.filled = nil
	if err := f.favs.QueueOfflineCapture(f.ctx, f.s.DB, f.user, plain, srv.URL+"/plain", false); err != nil {
		t.Fatal(err)
	}
	f.runCaptures()
	if len(previews.filled) != 0 {
		t.Fatalf("refresh filled previews %v", previews.filled)
	}
}

func TestCaptureBrowserDown(t *testing.T) {
	f := newFixture(t, true)
	f.c.Browser = &fakeBrowser{err: errors.New("connection refused")}
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	srv := newSite(t, map[string]func(http.ResponseWriter){"/post": htmlPage(articlePage(""))})
	id := f.link(srv.URL + "/post")
	f.runCaptures()
	if c := f.get(id); c.Status != StatusReady || !c.HasArticle || c.HasScreenshot {
		t.Fatalf("copy = %+v, want an article without a screenshot", c)
	}
	if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "screenshot") {
		t.Fatalf("logs = %s, want a warning", logs.String())
	}
}

func TestSanitizeDocument(t *testing.T) {
	doc := []byte(`<html><head><script>alert(1)</script></head><body><main><p onclick="x()">Hello</p>
<script>alert(2)</script><img src="https://example.org/a.png"><a href="javascript:alert(3)">x</a>
<svg><script>alert(4)</script></svg><iframe srcdoc="<script>alert(5)</script>"></iframe>
<meta http-equiv="refresh" content="0;url=https://evil.example"></main></body></html>`)
	out, ok := SanitizeDocument(doc, "Title", "https://example.org/post")
	if !ok {
		t.Fatal("not ok")
	}
	s := string(out)
	for _, bad := range []string{"<script", "onclick", "javascript:", "<svg", "srcdoc", "<iframe", "http-equiv", "alert("} {
		if strings.Contains(s, bad) {
			t.Errorf("document contains %q:\n%s", bad, s)
		}
	}
	for _, want := range []string{"Hello", `src="https://example.org/a.png"`, "<title>Title</title>", `href="https://example.org/post"`} {
		if !strings.Contains(s, want) {
			t.Errorf("document lacks %q:\n%s", want, s)
		}
	}
	if _, ok := SanitizeDocument([]byte(`<html><body><script>x</script></body></html>`), "", ""); ok {
		t.Error("document without text is ok")
	}
	if out, _ := SanitizeDocument(doc, "T", "javascript:alert(1)"); strings.Contains(string(out), "javascript:") {
		t.Error("javascript: source URL kept")
	}
}

func TestExtractArticleRelativeURLs(t *testing.T) {
	base, _ := url.Parse("https://go.dev/blog/post")
	art, ok := ExtractArticle([]byte(articlePage("")), base)
	if !ok {
		t.Fatal("no article")
	}
	if !strings.Contains(string(art.HTML), `src="https://go.dev/img/diagram.png"`) {
		t.Fatalf("relative image not made absolute:\n%s", art.HTML)
	}
}

func ptr(s string) *string { return &s }

func TestSweep(t *testing.T) {
	f := newFixture(t, true)
	srv := newSite(t, map[string]func(http.ResponseWriter){"/post": htmlPage(articlePage(""))})
	id := f.link(srv.URL + "/post")
	f.runCaptures()
	first := f.get(id).articleID
	// A refresh replaces the article; a deleted link leaves its files behind.
	if err := f.favs.QueueOfflineCapture(f.ctx, f.s.DB, f.user, id, srv.URL+"/post", false); err != nil {
		t.Fatal(err)
	}
	f.runCaptures()
	current := f.get(id).articleID
	gone := f.link(srv.URL + "/post")
	f.runCaptures()
	goneFile := f.get(gone).articleID
	if err := f.favs.Delete(f.ctx, f.user, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	if first == current || first == "" || goneFile == "" {
		t.Fatalf("files %q %q %q", first, current, goneFile)
	}

	// Fresh files stay, so a capture that is about to record its parts isn't affected.
	if err := f.s.Sweep(f.ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{first, current, goneFile} {
		if _, err := os.Stat(f.s.Files.Path(id)); err != nil {
			t.Fatalf("fresh file %s removed: %v", id, err)
		}
	}
	f.s.Now = func() time.Time { return time.Now().Add(blobs.OrphanAge + time.Minute) }
	if err := f.s.Sweep(f.ctx); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{first: false, goneFile: false, current: true} {
		_, err := os.Stat(f.s.Files.Path(id))
		if exists := err == nil; exists != want {
			t.Errorf("file %s exists = %v, want %v", id, exists, want)
		}
	}
}
