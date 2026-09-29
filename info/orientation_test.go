package info

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

// tiffEntry builds a TIFF header with one IFD0 entry.
type tiffEntry struct {
	order  binary.ByteOrder
	magic  uint16
	tag    uint16
	typ    uint16
	count  uint32
	value  uint16
	offset uint32 // IFD0 offset; 0 means 8
}

func (e tiffEntry) bytes() []byte {
	order := e.order
	if order == nil {
		order = binary.BigEndian
	}
	var b bytes.Buffer
	if order == binary.LittleEndian {
		b.WriteString("II")
	} else {
		b.WriteString("MM")
	}
	magic := e.magic
	if magic == 0 {
		magic = 42
	}
	offset := e.offset
	if offset == 0 {
		offset = 8
	}
	binary.Write(&b, order, magic)
	binary.Write(&b, order, offset)
	binary.Write(&b, order, uint16(1))
	binary.Write(&b, order, e.tag)
	binary.Write(&b, order, e.typ)
	binary.Write(&b, order, e.count)
	binary.Write(&b, order, e.value)
	binary.Write(&b, order, uint16(0))
	binary.Write(&b, order, uint32(0))
	return b.Bytes()
}

func valid(v uint16) tiffEntry { return tiffEntry{tag: 0x0112, typ: 3, count: 1, value: v} }

func TestTiffOrientation(t *testing.T) {
	cases := map[string]struct {
		tiff []byte
		want int
	}{
		"big endian 6":    {valid(6).bytes(), 6},
		"little endian 8": {tiffEntry{order: binary.LittleEndian, tag: 0x0112, typ: 3, count: 1, value: 8}.bytes(), 8},
		"bad magic":       {tiffEntry{magic: 43, tag: 0x0112, typ: 3, count: 1, value: 6}.bytes(), 1},
		"ASCII type":      {tiffEntry{tag: 0x0112, typ: 2, count: 1, value: 6}.bytes(), 1},
		"LONG type":       {tiffEntry{tag: 0x0112, typ: 4, count: 1, value: 6}.bytes(), 1},
		"count 0":         {tiffEntry{tag: 0x0112, typ: 3, count: 0, value: 6}.bytes(), 1},
		"count 2":         {tiffEntry{tag: 0x0112, typ: 3, count: 2, value: 6}.bytes(), 1},
		"value 0":         {valid(0).bytes(), 1},
		"value 9":         {valid(9).bytes(), 1},
		"other tag":       {tiffEntry{tag: 0x0110, typ: 3, count: 1, value: 6}.bytes(), 1},
		"IFD past end":    {tiffEntry{tag: 0x0112, typ: 3, count: 1, value: 6, offset: 0xFFFFFFF0}.bytes(), 1},
		"IFD in header":   {tiffEntry{tag: 0x0112, typ: 3, count: 1, value: 6, offset: 4}.bytes(), 1},
		"truncated entry": {valid(6).bytes()[:16], 1},
		"empty":           {nil, 1},
		"not TIFF":        {[]byte("hello world, not tiff"), 1},
	}
	for name, c := range cases {
		if got := tiffOrientation(c.tiff); got != c.want {
			t.Errorf("%s: got %d, want %d", name, got, c.want)
		}
	}
}

// jpegWithExif returns a tiny JPEG carrying tiff as its EXIF payload.
func jpegWithExif(t *testing.T, tiff []byte) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, image.NewGray(image.Rect(0, 0, 20, 10)), nil); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	var out bytes.Buffer
	out.Write(encoded.Bytes()[:2])
	out.Write([]byte{0xFF, 0xE1})
	binary.Write(&out, binary.BigEndian, uint16(len(payload)+2))
	out.Write(payload)
	out.Write(encoded.Bytes()[2:])
	return out.Bytes()
}

func TestJpegOrientationSurvivesGarbage(t *testing.T) {
	good := jpegWithExif(t, valid(6).bytes())
	if got := jpegOrientation(bytes.NewReader(good)); got != 6 {
		t.Fatalf("valid JPEG: got %d, want 6", got)
	}
	// Every truncation of a valid file must return without panicking.
	for n := 0; n < len(good); n++ {
		jpegOrientation(bytes.NewReader(good[:n]))
	}
	for _, input := range [][]byte{nil, {0xFF}, {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x01}, {0xFF, 0xD8, 0xFF, 0xFF, 0xFF}} {
		if got := jpegOrientation(bytes.NewReader(input)); got != 1 {
			t.Errorf("%x: got %d, want 1", input, got)
		}
	}
}

// The reader must agree with libvips, which rotates the outputs.
func TestJpegOrientationAgreesWithVips(t *testing.T) {
	entries := map[string]tiffEntry{
		"valid 6":    valid(6),
		"valid 8 LE": {order: binary.LittleEndian, tag: 0x0112, typ: 3, count: 1, value: 8},
		"ASCII type": {tag: 0x0112, typ: 2, count: 1, value: 6},
		"count 0":    {tag: 0x0112, typ: 3, count: 0, value: 6},
		"bad magic":  {magic: 43, tag: 0x0112, typ: 3, count: 1, value: 6},
	}
	for name, e := range entries {
		data := jpegWithExif(t, e.bytes())
		path := filepath.Join(t.TempDir(), "x.jpg")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
		img, err := vips.NewImageFromFile(path)
		if err != nil {
			t.Fatalf("%s: vips: %v", name, err)
		}
		want := img.Orientation()
		img.Close()
		if want == 0 {
			want = 1
		}
		if got := jpegOrientation(bytes.NewReader(data)); got != want {
			t.Errorf("%s: reader %d, vips %d", name, got, want)
		}
	}
}
