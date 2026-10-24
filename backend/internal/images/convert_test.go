package images

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

var (
	red  = color.RGBA{255, 0, 0, 255}
	blue = color.RGBA{0, 0, 255, 255}
)

// halves returns a w×h image whose left half is red and right half blue.
func halves(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if x < w/2 {
				img.Set(x, y, red)
			} else {
				img.Set(x, y, blue)
			}
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// withOrientation inserts an APP1 segment with the EXIF orientation after the
// JPEG's start marker, in little- or big-endian byte order.
func withOrientation(jpg []byte, orientation uint16, order binary.ByteOrder) []byte {
	tiff := make([]byte, 8+2+12+4)
	if order == binary.LittleEndian {
		copy(tiff, "II*\x00")
	} else {
		copy(tiff, "MM\x00*")
	}
	order.PutUint32(tiff[4:], 8)
	order.PutUint16(tiff[8:], 1)       // one entry
	order.PutUint16(tiff[10:], 0x0112) // Orientation
	order.PutUint16(tiff[12:], 3)      // SHORT
	order.PutUint32(tiff[14:], 1)      // one value
	order.PutUint16(tiff[18:], orientation)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(seg)+2))
	out := append([]byte{}, jpg[:2]...)
	out = append(out, app1...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || format != "jpeg" {
		t.Fatalf("result is %q, err %v; want a JPEG", format, err)
	}
	return img
}

// near reports whether c is close to want (JPEG is lossy).
func near(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	d := func(a uint32, b uint8) bool { x := int(a>>8) - int(b); return x > -40 && x < 40 }
	return d(r, want.R) && d(g, want.G) && d(b, want.B)
}

func TestJPEGOrientation(t *testing.T) {
	jpg := encodeJPEG(t, halves(16, 8))
	if o := jpegOrientation(jpg); o != 1 {
		t.Fatalf("without EXIF: %d, want 1", o)
	}
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for o := uint16(1); o <= 8; o++ {
			if got := jpegOrientation(withOrientation(jpg, o, order)); got != int(o) {
				t.Errorf("%v %d: got %d", order, o, got)
			}
		}
	}
	if o := jpegOrientation(withOrientation(jpg, 9, binary.BigEndian)); o != 1 {
		t.Fatalf("invalid value: %d, want 1", o)
	}
	if o := jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF}); o != 1 {
		t.Fatalf("truncated: %d, want 1", o)
	}
}

func TestConvertOrientation(t *testing.T) {
	c := Converter{MaxPixels: 1 << 20, Width: 800}
	jpg := encodeJPEG(t, halves(32, 16))
	for _, tt := range []struct {
		orientation uint16
		w, h        int
		redAt       image.Point // a point that must be red; the opposite side is blue
		blueAt      image.Point
	}{
		{1, 32, 16, image.Pt(4, 8), image.Pt(28, 8)},
		{3, 32, 16, image.Pt(28, 8), image.Pt(4, 8)},
		{6, 16, 32, image.Pt(8, 4), image.Pt(8, 28)}, // left side turns to the top
		{8, 16, 32, image.Pt(8, 28), image.Pt(8, 4)}, // left side turns to the bottom
	} {
		out, err := c.Convert(withOrientation(jpg, tt.orientation, binary.BigEndian))
		if err != nil {
			t.Fatalf("orientation %d: %v", tt.orientation, err)
		}
		img := decode(t, out)
		if b := img.Bounds(); b.Dx() != tt.w || b.Dy() != tt.h {
			t.Fatalf("orientation %d: %dx%d, want %dx%d", tt.orientation, b.Dx(), b.Dy(), tt.w, tt.h)
		}
		if !near(img.At(tt.redAt.X, tt.redAt.Y), red) || !near(img.At(tt.blueAt.X, tt.blueAt.Y), blue) {
			t.Errorf("orientation %d: %v at %v, %v at %v", tt.orientation,
				img.At(tt.redAt.X, tt.redAt.Y), tt.redAt, img.At(tt.blueAt.X, tt.blueAt.Y), tt.blueAt)
		}
	}
}

func TestConvertSizes(t *testing.T) {
	c := Converter{MaxPixels: 25_000_000, Width: 800}
	for name, tt := range map[string]struct {
		data []byte
		w, h int
	}{
		"large photo": {encodeJPEG(t, halves(3000, 2000)), 800, 533},
		"small png":   {encodePNG(t, halves(600, 315)), 600, 315},
		"exactly 800": {encodePNG(t, halves(800, 10)), 800, 10},
		"tall turned": {withOrientation(encodeJPEG(t, halves(2000, 1000)), 6, binary.LittleEndian), 800, 1600},
	} {
		out, err := c.Convert(tt.data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if b := decode(t, out).Bounds(); b.Dx() != tt.w || b.Dy() != tt.h {
			t.Errorf("%s: %dx%d, want %dx%d", name, b.Dx(), b.Dy(), tt.w, tt.h)
		}
	}
}

func TestConvertTransparencyOnWhite(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20)) // fully transparent
	for x := range 10 {
		for y := range 20 {
			img.Set(x, y, red)
		}
	}
	out, err := Converter{MaxPixels: 1 << 20, Width: 800}.Convert(encodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	res := decode(t, out)
	if !near(res.At(15, 10), color.RGBA{255, 255, 255, 255}) || !near(res.At(3, 10), red) {
		t.Fatalf("transparent area = %v, logo = %v; want white and red", res.At(15, 10), res.At(3, 10))
	}
}

func TestConvertAnimatedGIFUsesFirstFrame(t *testing.T) {
	pal := color.Palette{red, blue}
	frame := func(i uint8) *image.Paletted {
		p := image.NewPaletted(image.Rect(0, 0, 10, 10), pal)
		for j := range p.Pix {
			p.Pix[j] = i
		}
		return p
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{frame(0), frame(1)}, Delay: []int{10, 10}}); err != nil {
		t.Fatal(err)
	}
	out, err := Converter{MaxPixels: 1 << 20, Width: 800}.Convert(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if c := decode(t, out).At(5, 5); !near(c, red) {
		t.Fatalf("pixel = %v, want the first frame's red", c)
	}
}

func TestConvertWebP(t *testing.T) {
	// A 1×1 lossless WebP.
	webp, _ := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	out, err := Converter{MaxPixels: 1 << 20, Width: 800}.Convert(webp)
	if err != nil {
		t.Fatal(err)
	}
	if b := decode(t, out).Bounds(); b.Dx() != 1 || b.Dy() != 1 {
		t.Fatalf("size = %v, want 1×1", b)
	}
}

// pngHeader returns a PNG that declares w×h pixels but has no image data.
func pngHeader(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8 bit RGBA
	chunk := append([]byte("IHDR"), ihdr...)
	out := []byte("\x89PNG\r\n\x1a\n")
	out = binary.BigEndian.AppendUint32(out, 13)
	out = append(out, chunk...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(chunk))
}

func TestConvertRejects(t *testing.T) {
	c := Converter{MaxPixels: 25_000_000, Width: 800}
	if _, err := c.Convert(pngHeader(30000, 30000)); !errors.Is(err, ErrTooManyPixels) {
		t.Fatalf("30000×30000: err = %v, want ErrTooManyPixels", err)
	}
	for name, data := range map[string][]byte{
		"svg":    []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"ico":    append([]byte{0, 0, 1, 0, 1, 0}, make([]byte, 32)...),
		"text":   []byte("hello"),
		"empty":  nil,
		"broken": pngHeader(10, 10),
	} {
		if _, err := c.Convert(data); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
}
