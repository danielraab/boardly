package blobs

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadRemove(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "files"))
	if err != nil {
		t.Fatal(err)
	}
	id := NewID()
	if err := d.Write(id, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(d.Path(id))
	if err != nil || string(got) != "hello" {
		t.Fatalf("read %q, %v", got, err)
	}
	if err := d.Remove(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(d.Path(id)); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := d.Remove(id); err != nil {
		t.Fatalf("removing a missing file: %v", err)
	}
}

func TestPathRejectsNonIDs(t *testing.T) {
	d := Dir("/data/files")
	for _, id := range []string{"../etc/passwd", "", "abc", "550E8400-E29B-41D4-A716-446655440000"} {
		if got := d.Path(id); got != filepath.Join("/data/files", "invalid") {
			t.Errorf("Path(%q) = %q", id, got)
		}
	}
}

func TestRemoveOrphans(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	kept, orphan, recent := NewID(), NewID(), NewID()
	for _, id := range []string{kept, orphan, recent} {
		if err := d.Write(id, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	tmp := filepath.Join(string(d), tmpPrefix+"leftover")
	if err := os.WriteFile(tmp, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(string(d), "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	for _, p := range []string{d.Path(kept), d.Path(orphan), tmp} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	err = d.RemoveOrphans(context.Background(), time.Now(), func(_ context.Context, id string) (bool, error) {
		return id == kept, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{
		d.Path(kept): true, d.Path(orphan): false, d.Path(recent): true, tmp: false,
		filepath.Join(string(d), "sub"): true,
	} {
		_, err := os.Stat(p)
		if exists := err == nil; exists != want {
			t.Errorf("%s exists = %v, want %v", filepath.Base(p), exists, want)
		}
	}
}

func TestSetHeaders(t *testing.T) {
	h := http.Header{}
	SetHeaders(h, "image/png")
	if h.Get("Content-Type") != "image/png" || h.Get("X-Content-Type-Options") != "nosniff" ||
		h.Get("Content-Security-Policy") != "sandbox" {
		t.Fatalf("headers = %v", h)
	}
}
