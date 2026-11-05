package dataimport

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"

	"boardly/internal/blobs"
	"boardly/internal/db"
	"boardly/internal/favourites"
)

// withOfflineCopy gives Ann's link url an offline copy with an article and a
// screenshot, captured on 2026-10-01.
func (f *fixture) withOfflineCopy(url string) string {
	f.t.Helper()
	link, err := f.fav.Create(f.ctx, f.ann, favourites.NewNode{Kind: favourites.KindLink, URL: &url, Title: ptr("Tour")})
	if err != nil {
		f.t.Fatal(err)
	}
	var shot bytes.Buffer
	if err := png.Encode(&shot, image.NewRGBA(image.Rect(0, 0, 1280, 800))); err != nil {
		f.t.Fatal(err)
	}
	article, screenshot := blobs.NewID(), blobs.NewID()
	for id, data := range map[string][]byte{
		article:    []byte(`<!doctype html><html><body><main><p>The tour of Go.</p></main></body></html>`),
		screenshot: shot.Bytes(),
	} {
		if err := f.s.OfflineFiles.Write(id, data); err != nil {
			f.t.Fatal(err)
		}
	}
	if _, err := f.db.ExecContext(f.ctx,
		`INSERT INTO offline_copies (node_id, user_id, url, status, title, article_id, screenshot_id, captured_at, updated_at)
		 VALUES (?, ?, ?, 'ready', 'A Tour of Go', ?, ?, '2026-10-01T08:00:00.000Z', ?)`,
		link.ID, f.ann, url, article, screenshot, db.FormatTime(f.fav.Now()),
	); err != nil {
		f.t.Fatal(err)
	}
	return link.ID
}

// offlineOf returns the offline copy of the user's link with the URL as
// "status captured title", and its article and screenshot files.
func (f *fixture) offlineOf(userID, url string) (row string, article, screenshot []byte) {
	f.t.Helper()
	var articleID, shotID *string
	err := f.db.QueryRowContext(f.ctx,
		`SELECT o.status || ' ' || o.captured_at || ' ' || COALESCE(o.title, ''), o.article_id, o.screenshot_id
		 FROM offline_copies o JOIN favourite_nodes n ON n.id = o.node_id WHERE n.user_id = ? AND n.url = ?`, userID, url,
	).Scan(&row, &articleID, &shotID)
	if err != nil {
		return "", nil, nil
	}
	read := func(id *string) []byte {
		if id == nil {
			return nil
		}
		data, err := os.ReadFile(f.s.OfflineFiles.Path(*id))
		if err != nil {
			f.t.Fatal(err)
		}
		return data
	}
	return row, read(articleID), read(shotID)
}

// rewriteZip replaces the entries of a ZIP by name.
func rewriteZip(t *testing.T, data []byte, replace map[string][]byte) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, file := range zr.File {
		w, err := zw.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if content, ok := replace[file.Name]; ok {
			_, err = w.Write(content)
		} else {
			var rc io.ReadCloser
			if rc, err = file.Open(); err == nil {
				_, err = io.Copy(w, rc)
				rc.Close()
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestImportOfflineCopies(t *testing.T) {
	f := newFixture(t)
	f.fav.Jobs = nil
	const url = "https://go.dev/tour"
	f.withOfflineCopy(url)
	zipped := f.exportOf(f.ann, true)

	report := f.importInto(f.bob, zipped, Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.bob), SkipDuplicates: true})
	row, article, shot := f.offlineOf(f.bob, url)
	if row != "ready 2026-10-01T08:00:00.000Z A Tour of Go" {
		t.Fatalf("Bob's copy = %q", row)
	}
	if !strings.Contains(string(article), "The tour of Go.") || !strings.Contains(string(article), `<meta name="referrer" content="no-referrer">`) {
		t.Fatalf("article = %s", article)
	}
	if cfg, format, err := image.DecodeConfig(bytes.NewReader(shot)); err != nil || format != "jpeg" || cfg.Width != 1280 {
		t.Fatalf("screenshot = %s %d, %v", format, cfg.Width, err)
	}
	if report.Files.OfflineCopies != 2 || report.Files.Skipped != 0 {
		t.Fatalf("files report = %+v", report.Files)
	}
	if n := f.count(`SELECT COUNT(*) FROM jobs WHERE kind = 'offline_capture'`); n != 0 {
		t.Fatalf("capture jobs = %d, want none", n)
	}

	// Into the same account: the link is a duplicate and keeps its own copy.
	before := f.count(`SELECT COUNT(*) FROM offline_copies`)
	files, _ := os.ReadDir(string(f.s.OfflineFiles))
	report = f.importInto(f.ann, zipped, Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.ann), SkipDuplicates: true})
	if n := f.count(`SELECT COUNT(*) FROM offline_copies`); n != before || report.Files.OfflineCopies != 0 {
		t.Fatalf("copies = %d (was %d), report %+v", n, before, report.Files)
	}
	if after, _ := os.ReadDir(string(f.s.OfflineFiles)); len(after) != len(files) {
		t.Fatalf("files of the skipped duplicate were left: %d, want %d", len(after), len(files))
	}
}

func TestImportOfflineCopyJSONAndOld(t *testing.T) {
	f := newFixture(t)
	f.withOfflineCopy("https://go.dev/tour")
	plain := f.exportOf(f.ann, false)
	if !strings.Contains(string(plain), `"capturedAt": "2026-10-01T08:00:00.000Z"`) {
		t.Fatalf("export lacks the copy: %s", plain)
	}
	report := f.importInto(f.bob, plain, Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.bob)})
	if row, _, _ := f.offlineOf(f.bob, "https://go.dev/tour"); row != "" || report.Files.Skipped != 0 {
		t.Fatalf("JSON import: copy %q, report %+v", row, report.Files)
	}

	// An export from before offline copies.
	old := strings.ReplaceAll(string(plain), `"offlineCopy"`, `"ignored"`)
	if _, err := f.s.Import(f.ctx, f.bob, []byte(old), Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.bob), SkipDuplicates: false}); err != nil {
		t.Fatalf("old export: %v", err)
	}
}

func TestImportOfflineCopyUntrustedFiles(t *testing.T) {
	f := newFixture(t)
	link := f.withOfflineCopy("https://go.dev/tour")
	zipped := rewriteZip(t, f.exportOf(f.ann, true), map[string][]byte{
		"offline/" + link + "/article.html":   []byte(`<html><body><main><p onclick="x()">Edited</p><script>alert(1)</script></main></body></html>`),
		"offline/" + link + "/screenshot.jpg": []byte("not an image"),
	})
	report := f.importInto(f.bob, zipped, Options{Parts: Parts{Favourites: true}, FolderID: f.root(f.bob)})
	row, article, shot := f.offlineOf(f.bob, "https://go.dev/tour")
	if !strings.HasPrefix(row, "ready ") || shot != nil {
		t.Fatalf("copy %q, screenshot %d bytes", row, len(shot))
	}
	if s := string(article); !strings.Contains(s, "Edited") || strings.Contains(s, "<script") || strings.Contains(s, "onclick") {
		t.Fatalf("article = %s", s)
	}
	if report.Files.Skipped != 1 || report.Files.OfflineCopies != 1 {
		t.Fatalf("files report = %+v", report.Files)
	}
}
