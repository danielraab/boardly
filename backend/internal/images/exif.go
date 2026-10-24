package images

import (
	"bytes"
	"encoding/binary"
	"image"
)

// jpegOrientation returns the EXIF orientation (1–8) of a JPEG file, read from
// the Orientation tag (0x0112) of IFD0 in its APP1 segment. Anything missing
// or unreadable counts as 1, the normal orientation.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan, end of image
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if seg := data[i+4 : i+2+size]; marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:4]) {
	case "II*\x00":
		order = binary.LittleEndian
	case "MM\x00*":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd:]))
	for e := ifd + 2; count > 0 && e+12 <= len(tiff); e, count = e+12, count-1 {
		if order.Uint16(tiff[e:]) != 0x0112 {
			continue
		}
		// Type SHORT, one value, stored in the first two bytes of the value field.
		if order.Uint16(tiff[e+2:]) != 3 {
			return 1
		}
		if o := int(order.Uint16(tiff[e+8:])); o >= 1 && o <= 8 {
			return o
		}
		return 1
	}
	return 1
}

// swapsSides reports whether an orientation turns the image by 90 degrees.
func swapsSides(orientation int) bool { return orientation >= 5 && orientation <= 8 }

// orient returns src turned and flipped so that it shows upright for the
// given EXIF orientation.
func orient(src *image.RGBA, orientation int) *image.RGBA {
	if orientation < 2 || orientation > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if swapsSides(orientation) {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range dh {
		for x := range dw {
			var sx, sy int
			switch orientation {
			case 2: // mirrored
				sx, sy = w-1-x, y
			case 3: // turned 180°
				sx, sy = w-1-x, h-1-y
			case 4: // flipped
				sx, sy = x, h-1-y
			case 5: // transposed
				sx, sy = y, x
			case 6: // needs 90° clockwise
				sx, sy = y, h-1-x
			case 7: // transversed
				sx, sy = w-1-y, h-1-x
			case 8: // needs 90° counter-clockwise
				sx, sy = w-1-y, x
			}
			so := src.PixOffset(sx+src.Rect.Min.X, sy+src.Rect.Min.Y)
			copy(dst.Pix[dst.PixOffset(x, y):][:4], src.Pix[so:so+4])
		}
	}
	return dst
}
