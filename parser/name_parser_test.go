package parser

import (
	"errors"
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

// Crop

func TestCrop(t *testing.T) {
	ic, err := NameToConfiguration(sc, "c0.050_0.640_0.390_0.920-w1024-q90.jpg")
	if err != nil {
		t.Fatal(err)
	}
	ensureImageConfiguration(t, ic, 1024, 0, 90, "jpg")
	if *ic.Crop != (core.CropBox{X0: 50, Y0: 640, X1: 390, Y1: 920}) {
		t.Errorf("unexpected crop %+v", *ic.Crop)
	}
}

func TestCropWithoutWidthOrQuality(t *testing.T) {
	ic, err := NameToConfiguration(sc, "c0.000_0.000_1.000_1.000.webp")
	if err != nil {
		t.Fatal(err)
	}
	ensureImageConfiguration(t, ic, 0, 0, 75, "webp")
	if *ic.Crop != (core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000}) {
		t.Errorf("unexpected crop %+v", *ic.Crop)
	}
}

func TestOtherVariantsHaveNoCrop(t *testing.T) {
	ic, _ := NameToConfiguration(sc, "w300.jpg")
	if ic.Crop != nil {
		t.Errorf("expected no crop, got %+v", *ic.Crop)
	}
}

func TestInvalidCrops(t *testing.T) {
	names := []string{
		"c0.05_0.640_0.390_0.920-w1024.jpg", // too few decimals
		"c0.0500_0.640_0.390_0.920.jpg",     // too many decimals
		"c.050_0.640_0.390_0.920.jpg",       // no leading zero
		"c0_0.640_0.390_0.920.jpg",          // no decimals
		"c0.050_0.640_1.001_0.920.jpg",      // above 1
		"c0.050_0.640_2.000_0.920.jpg",      // above 1
		"c0.390_0.640_0.050_0.920.jpg",      // x0 > x1
		"c0.050_0.640_0.050_0.920.jpg",      // x0 == x1
		"c0.050_0.920_0.390_0.920.jpg",      // y0 == y1
		"c0.050_0.920_0.390_0.640-w100.jpg", // y0 > y1
	}
	for _, name := range names {
		_, err := NameToConfiguration(sc, name)
		if !errors.Is(err, core.ErrInvalidCrop) {
			t.Errorf("%s: expected ErrInvalidCrop, got %v", name, err)
		}
	}
}

func TestCustomNamesStartingWithCAreNotCrops(t *testing.T) {
	for _, name := range []string{"cat_dog.jpg", "cover.png", "c1_2.jpg", "cat_dog_x_y.jpg", "c_a_b_c.jpg"} {
		ic, err := NameToConfiguration(sc, name)
		if err != nil {
			t.Errorf("%s: unexpected error %v", name, err)
			continue
		}
		if ic.Crop != nil || ic.Filename != name {
			t.Errorf("%s: expected a custom file name, got %+v", name, ic)
		}
	}
}
