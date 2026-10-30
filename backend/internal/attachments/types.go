package attachments

import (
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
)

// Type is a kind of file that can be attached.
type Type struct {
	ContentType string
	Extensions  []string
	// ByContent types are recognised from the file's bytes; the others from
	// the extension of its name.
	ByContent bool
	// Inline types are served to be shown in the browser; the others are
	// served as downloads.
	Inline bool
}

// Group is the part of the content type before the slash, such as "image".
func (t Type) Group() string {
	g, _, _ := strings.Cut(t.ContentType, "/")
	return g
}

// Types are all known attachment types.
var Types = []Type{
	{"image/png", []string{"png"}, true, true},
	{"image/jpeg", []string{"jpg", "jpeg"}, true, true},
	{"image/gif", []string{"gif"}, true, true},
	{"image/webp", []string{"webp"}, true, true},
	{"image/avif", []string{"avif"}, true, true},
	{"image/bmp", []string{"bmp"}, true, true},
	{"image/vnd.microsoft.icon", []string{"ico"}, true, true},
	{"image/svg+xml", []string{"svg"}, false, false},
	{"application/pdf", []string{"pdf"}, true, true},
	{"audio/mpeg", []string{"mp3"}, true, true},
	{"audio/ogg", []string{"ogg", "oga"}, true, true},
	{"audio/wav", []string{"wav"}, true, true},
	{"video/mp4", []string{"mp4"}, true, true},
	{"video/webm", []string{"webm"}, true, true},
	{"text/plain", []string{"txt"}, false, false},
	{"text/markdown", []string{"md"}, false, false},
	{"text/csv", []string{"csv"}, false, false},
	{"application/json", []string{"json"}, false, false},
	{"text/html", []string{"html", "htm"}, false, false},
	{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", []string{"docx"}, false, false},
	{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []string{"xlsx"}, false, false},
	{"application/vnd.openxmlformats-officedocument.presentationml.presentation", []string{"pptx"}, false, false},
	{"application/vnd.oasis.opendocument.text", []string{"odt"}, false, false},
	{"application/vnd.oasis.opendocument.spreadsheet", []string{"ods"}, false, false},
	{"application/vnd.oasis.opendocument.presentation", []string{"odp"}, false, false},
	{"application/zip", []string{"zip"}, false, false},
	{"application/x-7z-compressed", []string{"7z"}, false, false},
	{"application/x-tar", []string{"tar"}, false, false},
	{"application/gzip", []string{"gz", "tgz"}, false, false},
}

// DefaultAllowed are the types allowed when BOARDLY_ATTACHMENT_TYPES isn't
// set: images except SVG, and PDF.
func DefaultAllowed() []string {
	var allowed []string
	for _, t := range Types {
		if (t.Group() == "image" && t.ContentType != "image/svg+xml") || t.ContentType == "application/pdf" {
			allowed = append(allowed, t.ContentType)
		}
	}
	return allowed
}

// ParseAllowed reads a comma-separated list of content types and groups
// ("image/*") and returns the content types it allows, in table order.
func ParseAllowed(list string) ([]string, error) {
	want := map[string]bool{}
	for entry := range strings.SplitSeq(list, ",") {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		found := false
		for _, t := range Types {
			if t.ContentType == entry || entry == t.Group()+"/*" {
				want[t.ContentType] = true
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("%q is not a known attachment type", entry)
		}
	}
	if len(want) == 0 {
		return DefaultAllowed(), nil
	}
	var allowed []string
	for _, t := range Types {
		if want[t.ContentType] {
			allowed = append(allowed, t.ContentType)
		}
	}
	return allowed, nil
}

// TypeOf returns the known type with the content type.
func TypeOf(contentType string) (Type, bool) {
	i := slices.IndexFunc(Types, func(t Type) bool { return t.ContentType == contentType })
	if i < 0 {
		return Type{}, false
	}
	return Types[i], true
}

// DetectType returns the type of a file: from its content when that is
// recognised, otherwise from the extension of its name. An extension of a
// type recognised by content whose content doesn't match gives no type.
func DetectType(name string, data []byte) (Type, bool) {
	if len(data) == 0 {
		return Type{}, false
	}
	if t, ok := TypeOf(sniff(data)); ok {
		return t, true
	}
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	for _, t := range Types {
		if slices.Contains(t.Extensions, ext) {
			if t.ByContent {
				return Type{}, false
			}
			return t, true
		}
	}
	return Type{}, false
}

// sniff returns the content type of data when it is one of the types
// recognised by content, and "" otherwise.
func sniff(data []byte) string {
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		switch string(data[8:12]) {
		case "avif", "avis":
			return "image/avif"
		}
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0 {
		return "audio/mpeg" // MPEG audio frame without an ID3 tag
	}
	switch t := http.DetectContentType(data); t {
	case "image/x-icon":
		return "image/vnd.microsoft.icon"
	case "audio/wave":
		return "audio/wav"
	case "application/ogg":
		return "audio/ogg"
	default:
		if tt, ok := TypeOf(t); ok && tt.ByContent {
			return t
		}
	}
	return ""
}
