package parser

import (
	"testing"

	"github.com/image-server/image-server/core"
)

func ensureImageConfiguration(t *testing.T, ic *core.ImageConfiguration, w int, h int, q uint, f string) {
	if ic.Width != w {
		t.Errorf("expected %v to be %v", ic.Width, w)
	}
	if ic.Height != h {
		t.Errorf("expected %v to be %v", ic.Width, h)
	}
	if ic.Quality != q {
		t.Errorf("expected %v to be %v", ic.Quality, q)
	}
	if ic.Format != f {
		t.Errorf("expected %v to be %v", ic.Format, f)
	}
}

var sc *core.ServerConfiguration

func init() {
	sc = &core.ServerConfiguration{
		DefaultQuality: 75,
	}
}

// Use the default quality

func TestRectangle(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "300x400.jpg")
	ensureImageConfiguration(t, ic, 300, 400, 75, "jpg")
}

func TestSquare(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "x300.jpg")
	ensureImageConfiguration(t, ic, 300, 300, 75, "jpg")
}

func TestWidth(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "w300.jpg")
	ensureImageConfiguration(t, ic, 300, 0, 75, "jpg")
}

func TestFullSize(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "full_size.jpg")
	ensureImageConfiguration(t, ic, 0, 0, 75, "jpg")
}

func TestUnsupported(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "random.jpg")
	ensureImageConfiguration(t, ic, 0, 0, 0, "jpg")
}

// Quality is Provided

func TestRectangleWithQuality(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "300x400-q10.jpg")
	ensureImageConfiguration(t, ic, 300, 400, 10, "jpg")
}

func TestSquareWithQuality(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "x300-q10.jpg")
	ensureImageConfiguration(t, ic, 300, 300, 10, "jpg")
}

func TestWidthWithQuality(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "w300-q10.jpg")
	ensureImageConfiguration(t, ic, 300, 0, 10, "jpg")
}

func TestFullSizeWithQuality(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "full_size-q10.jpg")
	ensureImageConfiguration(t, ic, 0, 0, 10, "jpg")
}

// Without extension (defaults to jpg)

func TestRectangleNoExtension(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "512x512")
	ensureImageConfiguration(t, ic, 512, 512, 75, "jpg")
}

func TestSquareNoExtension(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "x300")
	ensureImageConfiguration(t, ic, 300, 300, 75, "jpg")
}

func TestWidthNoExtension(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "w300")
	ensureImageConfiguration(t, ic, 300, 0, 75, "jpg")
}

func TestFullSizeNoExtension(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "full_size")
	ensureImageConfiguration(t, ic, 0, 0, 75, "jpg")
}

func TestRectangleNoExtensionWithQuality(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "300x400-q50")
	ensureImageConfiguration(t, ic, 300, 400, 50, "jpg")
}
