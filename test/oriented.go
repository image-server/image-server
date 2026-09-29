package test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

// WriteOrientedJPEG writes a w×h JPEG (left half red, right half blue) that
// carries an EXIF orientation tag, like an iPhone photo taken sideways.
func WriteOrientedJPEG(tb testing.TB, path string, w, h, orientation int, order binary.ByteOrder) {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{R: 255, A: 255}
			if x >= w/2 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 95}); err != nil {
		tb.Fatal(err)
	}

	// TIFF header + IFD0 with one entry: Orientation (0x0112), SHORT, count 1.
	var tiff bytes.Buffer
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	binary.Write(&tiff, order, uint16(42))
	binary.Write(&tiff, order, uint32(8))
	binary.Write(&tiff, order, uint16(1))
	binary.Write(&tiff, order, uint16(0x0112))
	binary.Write(&tiff, order, uint16(3))
	binary.Write(&tiff, order, uint32(1))
	binary.Write(&tiff, order, uint16(orientation))
	binary.Write(&tiff, order, uint16(0))
	binary.Write(&tiff, order, uint32(0)) // no next IFD

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var out bytes.Buffer
	out.Write(encoded.Bytes()[:2]) // SOI
	out.Write([]byte{0xFF, 0xE1})
	binary.Write(&out, binary.BigEndian, uint16(len(payload)+2))
	out.Write(payload)
	out.Write(encoded.Bytes()[2:])
	if err := os.WriteFile(path, out.Bytes(), 0644); err != nil {
		tb.Fatal(err)
	}
}
