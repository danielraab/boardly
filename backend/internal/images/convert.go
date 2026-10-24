package images

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // registers the GIF decoder; Decode returns the first frame
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

// ErrTooManyPixels is returned for an image whose header declares more pixels
// than a converter allows. It is not decoded.
var ErrTooManyPixels = errors.New("image has too many pixels")

// previewQuality is the JPEG quality of converted previews.
const previewQuality = 80

// Converter turns an image into a preview: a JPEG of at most Width pixels
// wide, upright, with transparent areas on white.
type Converter struct {
	// MaxPixels is the largest width times height that is decoded.
	MaxPixels int64
	// Width is the largest width of the result; smaller images keep their size.
	Width int
}

// PreviewType returns the content type of an image that can become a
// preview: PNG, JPEG, GIF or WebP. Everything else fails with ErrUnsupported.
func PreviewType(data []byte) (string, error) {
	switch t := http.DetectContentType(data); t {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return t, nil
	}
	return "", ErrUnsupported
}

// Convert returns the preview of data. It fails with ErrUnsupported for other
// types and broken files, and with ErrTooManyPixels before decoding an image
// that declares more than MaxPixels pixels.
func (c Converter) Convert(data []byte) ([]byte, error) {
	contentType, err := PreviewType(data)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrUnsupported
	}
	if int64(cfg.Width)*int64(cfg.Height) > c.MaxPixels {
		return nil, ErrTooManyPixels
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	orientation := 1
	if contentType == "image/jpeg" {
		orientation = jpegOrientation(data)
	}

	// Scale first and turn the small result: the width limit applies to the
	// upright image, whose sides are swapped for orientations 5–8.
	b := src.Bounds()
	uprightW, uprightH := b.Dx(), b.Dy()
	if swapsSides(orientation) {
		uprightW, uprightH = uprightH, uprightW
	}
	outW, outH := uprightW, uprightH
	if c.Width > 0 && outW > c.Width {
		outW = c.Width
		outH = max(1, (uprightH*c.Width+uprightW/2)/uprightW)
	}
	scaledW, scaledH := outW, outH
	if swapsSides(orientation) {
		scaledW, scaledH = outH, outW
	}

	dst := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if scaledW == b.Dx() && scaledH == b.Dy() {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	} else {
		xdraw.BiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	}
	dst = orient(dst, orientation)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: previewQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
