package images

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boardly/internal/favourites"
	"boardly/internal/jobs"
)

type previewFixture struct {
	t    *testing.T
	ctx  context.Context
	p    *Previews
	favs *favourites.Store
	user string
}

func newPreviewFixture(t *testing.T, allowPrivate bool) *previewFixture {
	t.Helper()
	s, user := newStore(t)
	p, err := NewPreviews(s, jobs.New(s.DB), NewFavicons(s, allowPrivate).Client, 1<<20,
		Converter{MaxPixels: 25_000_000, Width: 800})
	if err != nil {
		t.Fatal(err)
	}
	return &previewFixture{t: t, ctx: context.Background(), p: p, favs: favourites.NewStore(s.DB), user: user}
}

func (f *previewFixture) link(rawURL string, description *string) string {
	f.t.Helper()
	n, err := f.favs.Create(f.ctx, f.user, favourites.NewNode{Kind: favourites.KindLink, URL: &rawURL, Description: description})
	if err != nil {
		f.t.Fatal(err)
	}
	return n.ID
}

// node returns a link's description and preview image ID.
func (f *previewFixture) node(id string) (description, preview *string) {
	f.t.Helper()
	if err := f.p.Images.DB.QueryRow(`SELECT description, preview_image_id FROM favourite_nodes WHERE id = ?`, id).
		Scan(&description, &preview); err != nil {
		f.t.Fatal(err)
	}
	return description, preview
}

// runJobs runs the pending jobs of kind with handler, oldest first, and
// deletes them, as the queue's workers would.
func (f *previewFixture) runJobs(kind jobs.Kind, handler jobs.Handler) int {
	f.t.Helper()
	rows, err := f.p.Images.DB.Query(`SELECT id, payload FROM jobs WHERE kind = ? ORDER BY id`, kind)
	if err != nil {
		f.t.Fatal(err)
	}
	var pending []jobs.Job
	for rows.Next() {
		var j jobs.Job
		var payload string
		if err := rows.Scan(&j.ID, &payload); err != nil {
			f.t.Fatal(err)
		}
		j.Kind, j.Payload = kind, json.RawMessage(payload)
		pending = append(pending, j)
	}
	rows.Close()
	for _, j := range pending {
		if err := handler(f.ctx, j); err != nil {
			f.t.Fatalf("%s job %s: %v", kind, j.Payload, err)
		}
		if _, err := f.p.Images.DB.Exec(`DELETE FROM jobs WHERE id = ?`, j.ID); err != nil {
			f.t.Fatal(err)
		}
	}
	return len(pending)
}

func (f *previewFixture) fetch(nodeID, rawURL string) {
	f.t.Helper()
	payload, _ := json.Marshal(jobs.PreviewFetchPayload{UserID: f.user, NodeID: nodeID, URL: rawURL})
	if err := f.p.fetch(f.ctx, jobs.Job{Kind: jobs.KindPreviewFetch, Payload: payload}); err != nil {
		f.t.Fatal(err)
	}
}

func (f *previewFixture) imageSize(id *string) image.Point {
	f.t.Helper()
	if id == nil {
		f.t.Fatal("no preview")
	}
	img, file, err := f.p.Images.Open(f.ctx, *id)
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()
	if img.Source != SourcePreview || img.ContentType != "image/jpeg" {
		f.t.Fatalf("preview is %s %s, want a preview JPEG", img.Source, img.ContentType)
	}
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		f.t.Fatal(err)
	}
	return image.Pt(cfg.Width, cfg.Height)
}

func (f *previewFixture) rawFiles() int {
	f.t.Helper()
	entries, err := os.ReadDir(f.p.rawDir)
	if err != nil {
		f.t.Fatal(err)
	}
	return len(entries)
}

func TestPreviewFetchOpenGraph(t *testing.T) {
	f := newPreviewFixture(t, true)
	card := encodePNG(t, halves(1200, 630))
	site := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/article":      htmlPage(`<head><meta property="og:description" content="A tour of Go"><meta property="og:image" content="/img/card.png"></head>`),
		"/img/card.png": file(card),
	})
	id := f.link(site.URL+"/article", nil)

	f.fetch(id, site.URL+"/article")
	if d, _ := f.node(id); d == nil || *d != "A tour of Go" {
		t.Fatalf("description = %v", d)
	}
	if f.rawFiles() != 1 {
		t.Fatalf("raw files = %d, want 1", f.rawFiles())
	}
	if n := f.runJobs(jobs.KindPreviewConvert, f.p.convert); n != 1 {
		t.Fatalf("convert jobs = %d, want 1", n)
	}
	_, preview := f.node(id)
	if size := f.imageSize(preview); size != image.Pt(800, 420) {
		t.Fatalf("preview size = %v, want 800×420", size)
	}
	if f.rawFiles() != 0 {
		t.Fatalf("raw files after conversion = %d, want 0", f.rawFiles())
	}
	var owner string
	if err := f.p.Images.DB.QueryRow(`SELECT user_id FROM images WHERE id = ?`, *preview).Scan(&owner); err != nil || owner != f.user {
		t.Fatalf("preview owner = %q, err %v", owner, err)
	}
}

func TestPreviewFetchKeepsDescription(t *testing.T) {
	f := newPreviewFixture(t, true)
	site := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/": htmlPage(`<meta property="og:description" content="Something else">`),
	})
	mine := "My notes"
	id := f.link(site.URL+"/", &mine)
	f.fetch(id, site.URL+"/")
	if d, _ := f.node(id); d == nil || *d != "My notes" {
		t.Fatalf("description = %v, want My notes", d)
	}
}

func TestPreviewFetchStaleURL(t *testing.T) {
	f := newPreviewFixture(t, true)
	site := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/old":   htmlPage(`<meta property="og:description" content="Old"><meta property="og:image" content="/i.png">`),
		"/i.png": file(encodePNG(t, halves(10, 10))),
	})
	id := f.link(site.URL+"/old", nil)
	if _, err := f.favs.Update(f.ctx, f.user, id, favourites.Patch{URL: ptr(site.URL + "/new")}); err != nil {
		t.Fatal(err)
	}
	f.fetch(id, site.URL+"/old")
	if d, _ := f.node(id); d != nil {
		t.Fatalf("description from the old URL stored: %q", *d)
	}
	if n := f.runJobs(jobs.KindPreviewConvert, f.p.convert); n != 0 {
		t.Fatalf("convert jobs = %d, want none for a changed link", n)
	}
}

func TestPreviewFetchRefusesBadImages(t *testing.T) {
	f := newPreviewFixture(t, true)
	big := append(encodePNG(t, halves(10, 10)), make([]byte, 1<<20)...)
	site := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/big":     htmlPage(`<meta property="og:image" content="/big.png">`),
		"/svg":     htmlPage(`<meta property="og:image" content="/x.svg">`),
		"/gone":    htmlPage(`<meta property="og:image" content="/missing.png">`),
		"/big.png": file(big),
		"/x.svg":   file(svgData),
	})
	for _, path := range []string{"/big", "/svg", "/gone", "/not-found"} {
		id := f.link(site.URL+path, nil)
		f.fetch(id, site.URL+path)
	}
	if n := f.runJobs(jobs.KindPreviewConvert, f.p.convert); n != 0 || f.rawFiles() != 0 {
		t.Fatalf("convert jobs = %d, raw files = %d; want none", n, f.rawFiles())
	}
}

func TestPreviewFetchBlocksPrivateAddresses(t *testing.T) {
	f := newPreviewFixture(t, false)
	private := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/":      htmlPage(`<meta property="og:description" content="Private">`),
		"/i.png": file(encodePNG(t, halves(10, 10))),
	})
	// The page itself is private.
	id := f.link(private.URL+"/", nil)
	f.fetch(id, private.URL+"/")
	if d, _ := f.node(id); d != nil {
		t.Fatalf("description from a private page: %q", *d)
	}

	// A public page that redirects to a private address.
	redirector := newSite(t, map[string]func(http.ResponseWriter, *http.Request){
		"/":     func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, private.URL+"/", http.StatusFound) },
		"/card": htmlPage(`<meta property="og:image" content="` + private.URL + `/i.png">`),
	})
	f.p.Client.Transport.(*http.Transport).DialContext = (&fakePublic{
		target: redirector.Listener.Addr().String(),
		guard:  guardedDialer(false),
	}).DialContext
	id = f.link("http://public.example/", nil)
	f.fetch(id, "http://public.example/")
	// A public page whose og:image is on a private address.
	id2 := f.link("http://public.example/card", nil)
	f.fetch(id2, "http://public.example/card")

	if d, _ := f.node(id); d != nil {
		t.Fatalf("description through a redirect: %q", *d)
	}
	if n := f.runJobs(jobs.KindPreviewConvert, f.p.convert); n != 0 {
		t.Fatalf("convert jobs = %d, want none", n)
	}
	if n := private.requests.Load(); n != 0 {
		t.Fatalf("private server got %d requests", n)
	}
}

func (f *previewFixture) queueFill(id, rawURL string, data []byte) {
	f.t.Helper()
	if err := f.p.queueConvert(f.ctx, jobs.PreviewConvertPayload{UserID: f.user, NodeID: id, URL: rawURL}, data); err != nil {
		f.t.Fatal(err)
	}
}

func previewCount(t *testing.T, p *Previews) int {
	t.Helper()
	var n int
	if err := p.Images.DB.QueryRow(`SELECT COUNT(*) FROM images WHERE source = 'preview'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPreviewConvertPrecedence(t *testing.T) {
	f := newPreviewFixture(t, true)
	const u = "https://example.org/a"
	small, wide := encodePNG(t, halves(100, 50)), encodePNG(t, halves(400, 100))

	// Fill into an empty link.
	id := f.link(u, nil)
	f.queueFill(id, u, small)
	f.runJobs(jobs.KindPreviewConvert, f.p.convert)
	_, first := f.node(id)
	if f.imageSize(first) != image.Pt(100, 50) {
		t.Fatal("fill didn't store the preview")
	}

	// A second fill finds the preview and is dropped, with its image.
	f.queueFill(id, u, wide)
	f.runJobs(jobs.KindPreviewConvert, f.p.convert)
	if _, p := f.node(id); *p != *first {
		t.Fatal("fill replaced an existing preview")
	}
	if n := previewCount(t, f.p); n != 1 {
		t.Fatalf("preview images = %d, want 1", n)
	}

	// A screenshot replaces it.
	if err := f.p.Replace(f.ctx, f.user, id, u, wide); err != nil {
		t.Fatal(err)
	}
	f.runJobs(jobs.KindPreviewConvert, f.p.convert)
	_, replaced := f.node(id)
	if *replaced == *first || f.imageSize(replaced) != image.Pt(400, 100) {
		t.Fatal("replace didn't store the screenshot")
	}

	// Any job for a changed or deleted link is dropped.
	f.queueFill(id, "https://example.org/old", small)
	if err := f.p.Replace(f.ctx, f.user, id, "https://example.org/old", small); err != nil {
		t.Fatal(err)
	}
	gone := f.link(u, nil)
	f.queueFill(gone, u, small)
	if err := f.favs.Delete(f.ctx, f.user, gone, favourites.DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	before := previewCount(t, f.p)
	f.runJobs(jobs.KindPreviewConvert, f.p.convert)
	if _, p := f.node(id); *p != *replaced {
		t.Fatal("a job for the old URL changed the preview")
	}
	if n := previewCount(t, f.p); n != before {
		t.Fatalf("preview images = %d, want %d (dropped jobs leave no image)", n, before)
	}
	if f.rawFiles() != 0 {
		t.Fatalf("raw files = %d, want 0", f.rawFiles())
	}
}

func TestPreviewConvertMissingAndBrokenFiles(t *testing.T) {
	f := newPreviewFixture(t, true)
	id := f.link("https://example.org/", nil)
	payload, _ := json.Marshal(jobs.PreviewConvertPayload{UserID: f.user, NodeID: id, URL: "https://example.org/", File: "6b1e2c4e-0000-4000-8000-000000000000"})
	if err := f.p.convert(f.ctx, jobs.Job{Payload: payload}); err != nil {
		t.Fatalf("missing raw file: %v", err)
	}
	f.queueFill(id, "https://example.org/", pngHeader(10, 10))
	f.runJobs(jobs.KindPreviewConvert, f.p.convert)
	if _, p := f.node(id); p != nil || f.rawFiles() != 0 {
		t.Fatalf("broken image: preview %v, raw files %d", p, f.rawFiles())
	}
}

func TestPreviewConvertKeepsRawFileWhenStopping(t *testing.T) {
	f := newPreviewFixture(t, true)
	id := f.link("https://example.org/", nil)
	f.queueFill(id, "https://example.org/", encodePNG(t, halves(10, 10)))
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	var payload string
	if err := f.p.Images.DB.QueryRow(`SELECT payload FROM jobs`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	_ = f.p.convert(ctx, jobs.Job{Payload: json.RawMessage(payload)})
	if f.rawFiles() != 1 {
		t.Fatalf("raw files = %d, want the file kept for the rerun", f.rawFiles())
	}
}

func TestReplaceChecksLimits(t *testing.T) {
	f := newPreviewFixture(t, true)
	id := f.link("https://example.org/", nil)
	for name, tt := range map[string]struct {
		data []byte
		want error
	}{
		"too large":       {append(encodePNG(t, halves(10, 10)), make([]byte, 1<<20)...), ErrTooLarge},
		"svg":             {svgData, ErrUnsupported},
		"text":            {[]byte("hello"), ErrUnsupported},
		"too many pixels": {pngHeader(30000, 30000), ErrTooManyPixels},
	} {
		if err := f.p.Replace(f.ctx, f.user, id, "https://example.org/", tt.data); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tt.want)
		}
	}
	var n int
	if err := f.p.Images.DB.QueryRow(`SELECT COUNT(*) FROM jobs`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("jobs = %d, err %v; want none", n, err)
	}
	if f.rawFiles() != 0 {
		t.Fatalf("raw files = %d, want 0", f.rawFiles())
	}
}

func TestRawPathRejectsOtherNames(t *testing.T) {
	f := newPreviewFixture(t, true)
	for _, name := range []string{"../boardly.db", "", "x", strings.Repeat("a", 36)} {
		if got := f.p.rawPath(name); filepath.Base(got) != "invalid" {
			t.Errorf("rawPath(%q) = %q", name, got)
		}
	}
}

func ptr(s string) *string { return &s }
