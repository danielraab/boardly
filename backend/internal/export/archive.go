package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"boardly/internal/blobs"
)

// DocumentName is the name of the export document inside the archive.
const DocumentName = "boardly.json"

// WriteJSON writes the document as indented JSON.
func WriteJSON(w io.Writer, doc *Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// FileName is the download name of an export: boardly-export-<slug>-<date>
// with the date in UTC and ".zip" or ".json" depending on withFiles.
func FileName(slug *string, now time.Time, withFiles bool) string {
	name := "user"
	if slug != nil && *slug != "" {
		name = *slug
	}
	ext := ".json"
	if withFiles {
		ext = ".zip"
	}
	return "boardly-export-" + name + "-" + now.UTC().Format("2006-01-02") + ext
}

// WriteArchive writes the ZIP archive of doc to w: the image and attachment
// files first, then the document with the path of each file it could include.
// A file that can't be opened is left out and its File stays nil. An error
// means w holds an incomplete archive.
func (s *Service) WriteArchive(w io.Writer, doc *Document) error {
	zw := zip.NewWriter(w)
	doc.WithFiles = true

	for i := range doc.Images {
		img := &doc.Images[i]
		name := "images/" + img.ID + imageExtension(img.ContentType)
		ok, err := addFile(zw, s.ImageFiles, img.ID, name, img.ContentType, img.CreatedAt)
		if err != nil {
			return fmt.Errorf("adding image %s: %w", img.ID, err)
		}
		if ok {
			img.File = &name
		}
	}
	for i := range doc.Attachments {
		a := &doc.Attachments[i]
		name := "attachments/" + a.ID + "/" + CleanFileName(a.Name)
		ok, err := addFile(zw, s.AttachmentFiles, a.ID, name, a.ContentType, a.CreatedAt)
		if err != nil {
			return fmt.Errorf("adding attachment %s: %w", a.ID, err)
		}
		if ok {
			a.File = &name
		}
	}

	header := &zip.FileHeader{Name: DocumentName, Method: zip.Deflate}
	header.Modified, _ = time.Parse(time.RFC3339, doc.ExportedAt)
	f, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	if err := WriteJSON(f, doc); err != nil {
		return err
	}
	return zw.Close()
}

// addFile copies the blob id into the archive as name. It reports false
// without an error when the blob can't be opened.
func addFile(zw *zip.Writer, dir blobs.Dir, id, name, contentType, createdAt string) (bool, error) {
	src, err := os.Open(dir.Path(id))
	if err != nil {
		return false, nil
	}
	defer func() { _ = src.Close() }()

	method := zip.Deflate
	if compressed(contentType) {
		method = zip.Store
	}
	header := &zip.FileHeader{Name: name, Method: method}
	header.Modified, _ = time.Parse(time.RFC3339, createdAt)
	dst, err := zw.CreateHeader(header)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(dst, src); err != nil {
		return false, err
	}
	return true, nil
}

// compressed reports whether files of the content type are already
// compressed, so deflating them again would only cost time.
func compressed(contentType string) bool {
	switch {
	case contentType == "image/svg+xml":
		return false
	case strings.HasPrefix(contentType, "image/"),
		strings.HasPrefix(contentType, "video/"),
		strings.HasPrefix(contentType, "audio/"):
		return true
	}
	switch contentType {
	case "application/pdf", "application/zip", "application/gzip", "application/x-7z-compressed",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.spreadsheet":
		return true
	}
	return false
}

var imageExtensions = map[string]string{
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/gif":                ".gif",
	"image/webp":               ".webp",
	"image/svg+xml":            ".svg",
	"image/x-icon":             ".ico",
	"image/vnd.microsoft.icon": ".ico",
}

func imageExtension(contentType string) string {
	if ext, ok := imageExtensions[contentType]; ok {
		return ext
	}
	return ".bin"
}

// maxNameBytes is the longest file name CleanFileName returns.
const maxNameBytes = 200

// CleanFileName turns an attachment name into a single safe file name: no
// path separators, control characters or characters that Windows doesn't
// allow, no leading or trailing dots and spaces, at most maxNameBytes bytes.
// An empty result becomes "file".
func CleanFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == utf8.RuneError, r < 0x20, r == 0x7f, strings.ContainsRune(`/\:*?"<>|`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	clean := strings.Trim(b.String(), ". ")
	if len(clean) > maxNameBytes {
		cut := maxNameBytes
		for cut > 0 && !utf8.RuneStart(clean[cut]) {
			cut--
		}
		clean = strings.TrimRight(clean[:cut], ". ")
	}
	if clean == "" {
		return "file"
	}
	return clean
}
