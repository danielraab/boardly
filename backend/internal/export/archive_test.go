package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// readArchive returns the entries of a ZIP archive by name.
func readArchive(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[f.Name] = content
	}
	return files
}

func TestWriteArchive(t *testing.T) {
	f := newFixture(t)
	s := f.seed()
	png := []byte("\x89PNG fake image")
	pdf := []byte("%PDF fake invoice")
	if err := f.s.ImageFiles.Write(s.image, png); err != nil {
		t.Fatal(err)
	}
	// The preview's file is missing on disk.
	if err := f.s.AttachmentFiles.Write(s.attachment, pdf); err != nil {
		t.Fatal(err)
	}

	doc, err := f.s.Document(f.ctx, f.ann.ID)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.s.WriteArchive(&buf, doc); err != nil {
		t.Fatal(err)
	}
	files := readArchive(t, buf.Bytes())

	imagePath := "images/" + s.image + ".png"
	attachmentPath := "attachments/" + s.attachment + "/invoice.pdf"
	if len(files) != 3 {
		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		t.Fatalf("archive has %v, want boardly.json, %s and %s", names, imagePath, attachmentPath)
	}
	if !bytes.Equal(files[imagePath], png) {
		t.Errorf("%s = %q, want the stored bytes", imagePath, files[imagePath])
	}
	if !bytes.Equal(files[attachmentPath], pdf) {
		t.Errorf("%s = %q, want the stored bytes", attachmentPath, files[attachmentPath])
	}

	var got Document
	if err := json.Unmarshal(files["boardly.json"], &got); err != nil {
		t.Fatal(err)
	}
	if !got.WithFiles {
		t.Error("withFiles = false in the archive")
	}
	for _, img := range got.Images {
		switch img.ID {
		case s.image:
			if img.File == nil || *img.File != imagePath {
				t.Errorf("image file = %v, want %s", img.File, imagePath)
			}
		case s.preview:
			if img.File != nil {
				t.Errorf("missing preview file = %q, want null", *img.File)
			}
		}
	}
	if len(got.Attachments) != 1 || got.Attachments[0].File == nil || *got.Attachments[0].File != attachmentPath {
		t.Errorf("attachments = %+v, want file %s", got.Attachments, attachmentPath)
	}
}

func TestWriteArchiveUnsafeName(t *testing.T) {
	f := newFixture(t)
	s := f.seed()
	f.exec(`UPDATE attachments SET name = '../../etc/passwd' WHERE id = ?`, s.attachment)
	if err := f.s.AttachmentFiles.Write(s.attachment, []byte("x")); err != nil {
		t.Fatal(err)
	}
	doc, err := f.s.Document(f.ctx, f.ann.ID)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.s.WriteArchive(&buf, doc); err != nil {
		t.Fatal(err)
	}
	prefix := "attachments/" + s.attachment + "/"
	found := false
	for name := range readArchive(t, buf.Bytes()) {
		if strings.HasPrefix(name, "/") || slices.Contains(strings.Split(name, "/"), "..") {
			t.Errorf("archive path %q leaves the archive root", name)
		}
		if strings.HasPrefix(name, prefix) {
			found = true
			if strings.Contains(strings.TrimPrefix(name, prefix), "/") {
				t.Errorf("attachment path %q has a subdirectory", name)
			}
		}
	}
	if !found {
		t.Errorf("no file inside %s", prefix)
	}
}

func TestCleanFileName(t *testing.T) {
	long := strings.Repeat("ä", 150) + ".txt"
	for in, want := range map[string]string{
		"invoice.pdf":      "invoice.pdf",
		"../../etc/passwd": "_.._etc_passwd",
		`C:\temp\a.txt`:    "C__temp_a.txt",
		"a\x00b\nc":        "a_b_c",
		"  ..  ":           "file",
		"":                 "file",
		"what?.md":         "what_.md",
	} {
		if got := CleanFileName(in); got != want {
			t.Errorf("CleanFileName(%q) = %q, want %q", in, got, want)
		}
	}
	got := CleanFileName(long)
	if len(got) > maxNameBytes || !utf8.ValidString(got) {
		t.Errorf("CleanFileName(long) = %d bytes, valid UTF-8 %v", len(got), utf8.ValidString(got))
	}
}

func TestFileName(t *testing.T) {
	day := time.Date(2026, 11, 2, 23, 30, 0, 0, time.FixedZone("CET", 3600))
	slug := "ann"
	if got := FileName(&slug, day, false); got != "boardly-export-ann-2026-11-02.json" {
		t.Errorf("FileName = %q", got)
	}
	if got := FileName(nil, day.Add(2*time.Hour), true); got != "boardly-export-user-2026-11-03.zip" {
		t.Errorf("FileName without slug = %q", got)
	}
}
