package dataimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

const minimalDoc = `{"format":"boardly","version":1,"exportedAt":"2026-11-02T10:00:00.000Z","withFiles":false,
"profile":{"email":"ann@example.org"},"favourites":null,"tags":[],"dashboards":[],"noteCategories":[],
"notes":[],"mediaSources":[],"images":[],"attachments":[]}`

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ptr[T any](v T) *T { return &v }

func TestParseJSON(t *testing.T) {
	f, err := Parse([]byte(minimalDoc))
	if err != nil {
		t.Fatal(err)
	}
	if f.Doc.Profile.Email != "ann@example.org" {
		t.Errorf("profile = %+v", f.Doc.Profile)
	}
	if _, err := f.ReadFile(ptr("images/x.png"), 100); err == nil {
		t.Error("a JSON export returned a file")
	}
}

func TestParseZIP(t *testing.T) {
	f, err := Parse(zipOf(t, map[string]string{"boardly.json": minimalDoc, "images/a.png": "png!"}))
	if err != nil {
		t.Fatal(err)
	}
	data, err := f.ReadFile(ptr("images/a.png"), 100)
	if err != nil || string(data) != "png!" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	if _, err := f.ReadFile(ptr("images/a.png"), 3); !errors.Is(err, errFileTooLarge) {
		t.Errorf("over the limit: %v", err)
	}
	for _, path := range []*string{nil, ptr("images/missing.png"), ptr("../images/a.png"), ptr("/images/a.png")} {
		if _, err := f.ReadFile(path, 100); err == nil {
			t.Errorf("ReadFile(%v) found a file", path)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	for name, data := range map[string][]byte{
		"version 2":       []byte(strings.Replace(minimalDoc, `"version":1`, `"version":2`, 1)),
		"other format":    []byte(`{"bookmarks":[],"lists":[]}`),
		"not JSON":        []byte("<html>"),
		"broken ZIP":      []byte("PK\x03\x04 broken"),
		"ZIP without doc": zipOf(t, map[string]string{"images/a.png": "x"}),
		"wrong types":     []byte(`{"format":"boardly","version":1,"notes":"many"}`),
	} {
		if _, err := Parse(data); !errors.Is(err, ErrInvalidFile) {
			t.Errorf("%s: err = %v, want ErrInvalidFile", name, err)
		}
	}
}
