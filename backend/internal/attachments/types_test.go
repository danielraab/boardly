package attachments

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDetectType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string // "" for no type
	}{
		{"scan.txt", pngBytes(t), "image/png"}, // recognised by content
		{"Offer.DOCX", []byte("PK\x03\x04rest"), "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"photo.jpg", []byte("just some text"), ""}, // fake image
		{"report.pdf", []byte("%PDF-1.7\n"), "application/pdf"},
		{"pic.avif", append([]byte("\x00\x00\x00\x1cftypavif"), make([]byte, 16)...), "image/avif"},
		{"favicon.ico", []byte("\x00\x00\x01\x00\x01\x00"), "image/vnd.microsoft.icon"},
		{"sound.wav", []byte("RIFF\x00\x00\x00\x00WAVEfmt "), "audio/wav"},
		{"sound.ogg", []byte("OggS\x00\x02rest"), "audio/ogg"},
		{"song.mp3", []byte("ID3\x03\x00\x00\x00rest"), "audio/mpeg"},
		{"raw.mp3", []byte{0xFF, 0xFB, 0x90, 0x00}, "audio/mpeg"},
		{"clip.mp4", append([]byte("\x00\x00\x00\x18ftypmp42"), make([]byte, 16)...), "video/mp4"},
		{"logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), "image/svg+xml"},
		{"notes.md", []byte("# Notes"), "text/markdown"},
		{"setup.exe", []byte("MZ\x90\x00"), ""},
		{"noext", []byte("hello"), ""},
		{"empty.txt", nil, ""},
	}
	for _, c := range cases {
		got, ok := DetectType(c.name, c.data)
		if (c.want == "") == ok || got.ContentType != c.want {
			t.Errorf("DetectType(%q) = %q, %v; want %q", c.name, got.ContentType, ok, c.want)
		}
	}
}

func TestParseAllowed(t *testing.T) {
	def, err := ParseAllowed("")
	if err != nil || !slices.Equal(def, DefaultAllowed()) {
		t.Fatalf("empty = %v, %v", def, err)
	}
	if slices.Contains(def, "image/svg+xml") || !slices.Contains(def, "application/pdf") || !slices.Contains(def, "image/png") {
		t.Errorf("default = %v", def)
	}

	got, err := ParseAllowed(" image/* , video/*")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, "image/svg+xml") || !slices.Contains(got, "video/mp4") || slices.Contains(got, "application/pdf") {
		t.Errorf("groups = %v", got)
	}

	for _, bad := range []string{"application/x-msdownload", "font/*"} {
		if _, err := ParseAllowed(bad); err == nil {
			t.Errorf("ParseAllowed(%q) succeeded", bad)
		}
	}
}

// TestEnvExampleListsAllTypes keeps .env.example in step with the table.
func TestEnvExampleListsAllTypes(t *testing.T) {
	data, err := os.ReadFile("../../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, ty := range Types {
		line := regexp.MustCompile(`(?m)^#\s+` + regexp.QuoteMeta(ty.ContentType) + `\s+(.*)$`).FindStringSubmatch(text)
		if line == nil {
			t.Errorf(".env.example doesn't list %s", ty.ContentType)
			continue
		}
		for _, ext := range ty.Extensions {
			if !strings.Contains(line[1], "."+ext) {
				t.Errorf(".env.example doesn't list .%s for %s", ext, ty.ContentType)
			}
		}
	}
	for _, risky := range []string{"image/svg+xml", "text/html"} {
		if !regexp.MustCompile(`(?m)^#\s+` + regexp.QuoteMeta(risky) + `\s+.*risky`).MatchString(text) {
			t.Errorf(".env.example doesn't mark %s as risky", risky)
		}
	}
	def := "BOARDLY_ATTACHMENT_TYPES=" + strings.Join(DefaultAllowed(), ",")
	if !strings.Contains(text, def) {
		t.Errorf(".env.example doesn't show the default %s", def)
	}
}
