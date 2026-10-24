package images

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"boardly/internal/dbtest"
	"boardly/internal/users"
)

// Minimal files of each type; enough for content sniffing.
var (
	pngData  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	jpegData = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 32)...)
	gifData  = append([]byte("GIF89a"), make([]byte, 32)...)
	webpData = append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)
	icoData  = append([]byte{0, 0, 1, 0, 1, 0}, make([]byte, 32)...)
	svgData  = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
)

func newStore(t *testing.T) (*Store, string) {
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
	return s, u.ID
}

func TestSniff(t *testing.T) {
	for want, data := range map[string][]byte{
		"image/png":    pngData,
		"image/jpeg":   jpegData,
		"image/gif":    gifData,
		"image/webp":   webpData,
		"image/x-icon": icoData,
	} {
		if got, err := Sniff(data); err != nil || got != want {
			t.Errorf("Sniff = %q, %v; want %q", got, err, want)
		}
	}
	for name, data := range map[string][]byte{
		"svg":   svgData,
		"html":  []byte("<html><body>hi</body></html>"),
		"empty": nil,
	} {
		if _, err := Sniff(data); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
	big := append(bytes.Clone(pngData), make([]byte, MaxSize)...)
	if _, err := Sniff(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("large: err = %v, want ErrTooLarge", err)
	}
}

func TestSaveOpenReplaceDelete(t *testing.T) {
	s, ann := newStore(t)
	ctx := context.Background()

	img, err := s.Save(ctx, SourceUpload, ann, pngData)
	if err != nil {
		t.Fatal(err)
	}
	if img.ContentType != "image/png" || img.Size != int64(len(pngData)) {
		t.Fatalf("saved = %+v", img)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, img.ID)); err != nil {
		t.Fatalf("file: %v", err)
	}

	got, f, err := s.Open(ctx, img.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if got.Source != SourceUpload || !bytes.Equal(data, pngData) {
		t.Fatalf("opened %+v with %d bytes", got, len(data))
	}

	if err := s.Replace(ctx, img.ID, gifData); err != nil {
		t.Fatal(err)
	}
	got, f, err = s.Open(ctx, img.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = io.ReadAll(f)
	f.Close()
	if got.ContentType != "image/gif" || !bytes.Equal(data, gifData) {
		t.Fatalf("replaced: %+v", got)
	}

	if err := s.Delete(ctx, img.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Open(ctx, img.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, img.ID)); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 0 {
		t.Fatalf("leftover files: %v", entries)
	}
}

func TestSaveRefusesBadImages(t *testing.T) {
	s, ann := newStore(t)
	ctx := context.Background()
	if _, err := s.Save(ctx, SourceUpload, ann, svgData); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("svg: %v", err)
	}
	if _, err := s.Save(ctx, SourceFavicon, "", append(bytes.Clone(pngData), make([]byte, MaxSize)...)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("large: %v", err)
	}
	if _, _, err := s.Open(ctx, "../../etc/passwd"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("path id: %v", err)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 0 {
		t.Fatalf("files were written: %v", entries)
	}
}

func TestLimits(t *testing.T) {
	big := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, MaxSize)...)
	if _, err := Sniff(big); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Sniff over MaxSize: err = %v, want ErrTooLarge", err)
	}
	if ct, err := SniffLimit(big, 2*MaxSize); err != nil || ct != "image/png" {
		t.Fatalf("SniffLimit: %q, %v", ct, err)
	}
	if _, err := ReadLimit(bytes.NewReader(make([]byte, 11)), 10); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("ReadLimit over the limit: err = %v, want ErrTooLarge", err)
	}
	if data, err := ReadLimit(bytes.NewReader(make([]byte, 10)), 10); err != nil || len(data) != 10 {
		t.Fatalf("ReadLimit at the limit: %d bytes, %v", len(data), err)
	}
}
