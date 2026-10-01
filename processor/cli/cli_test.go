package cli_test

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/processor/cli"
	. "github.com/image-server/image-server/test"
)

func TestFullSizeImage(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 0, Height: 0, Format: "jpg", Quality: 85, Namespace: "test", ID: "ofrA", Filename: "full_size.jpg"}

	expected := []string{"-auto-orient", "+repage", "-strip", "-format", "jpg", "-flatten", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/full_size.jpg"}
	p := cli.Processor{
		Source:             "public/test/00/of/rA/original",
		Destination:        "public/test/00/of/rA/full_size.jpg",
		ImageConfiguration: ic,
	}
	command := p.CommandArgs()
	Equals(t, expected, command)
}

func TestImageWithWidth(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 600, Height: 0, Format: "jpg", Quality: 85, Namespace: "test", ID: "ofrA", Filename: "w600.jpg"}

	expected := []string{"-auto-orient", "+repage", "-strip", "-format", "jpg", "-flatten", "-resize", "600", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/w600.jpg"}

	p := cli.Processor{
		Source:             "public/test/00/of/rA/original",
		Destination:        "public/test/00/of/rA/w600.jpg",
		ImageConfiguration: ic,
	}
	command := p.CommandArgs()
	Equals(t, expected, command)
}

func TestImageWithWidthAndHeight(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 600, Height: 500, Format: "jpg", Quality: 85, Namespace: "test", ID: "ofrA", Filename: "600x500.jpg"}
	id := &info.ImageProperties{Width: 600, Height: 500}

	expected := []string{"-auto-orient", "+repage", "-strip", "-format", "jpg", "-flatten", "-extent", "600x500", "-gravity", "center", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/600x500.jpg"}

	p := cli.Processor{
		Source:             "public/test/00/of/rA/original",
		Destination:        "public/test/00/of/rA/600x500.jpg",
		ImageConfiguration: ic,
		ImageDetails:       id,
	}
	command := p.CommandArgs()
	Equals(t, expected, command)
}

func TestBlankImage(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 600, Height: 0, Format: "jpg", Quality: 85, Namespace: "test", ID: "ofrA", Filename: "w600.jpg"}

	p := cli.Processor{
		Source:             "test/images/empty.jpg",
		Destination:        "public/test/00/of/rA/empty.jpg",
		ImageConfiguration: ic,
	}

	err := p.CreateImage()
	errorMsg := fmt.Sprintf("%s", err)
	Equals(t, "ImageMagick failed to process the image: convert -auto-orient +repage -strip -format jpg -flatten -resize 600 -background rgba(255,255,255,1) -quality 85 test/images/empty.jpg public/test/00/of/rA/empty.jpg", errorMsg)
}

func TestImageWithCrop(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 100, Format: "jpg", Quality: 90, Filename: "c0.050_0.640_0.390_0.920-w100-q90.jpg",
		Crop: &core.CropBox{X0: 50, Y0: 640, X1: 390, Y1: 920}}
	id := &info.ImageProperties{Width: 3024, Height: 4032}

	expected := []string{"-auto-orient", "+repage", "-strip", "-format", "jpg", "-flatten", "-crop", "1028x1129+151+2580", "+repage", "-resize", "100>", "-background", "rgba(255,255,255,1)", "-quality", "90", "original", "crop.jpg"}

	p := cli.Processor{
		Source:             "original",
		Destination:        "crop.jpg",
		ImageConfiguration: ic,
		ImageDetails:       id,
	}
	Equals(t, expected, p.CommandArgs())
}

// convertCrop runs ImageMagick for real on a stored 200×100 JPEG (left red,
// right blue) with the given EXIF orientation, upright 100×200
func convertCrop(t *testing.T, orientation, width int, crop *core.CropBox) image.Image {
	return convertCropSized(t, orientation, width, crop, &info.ImageProperties{Width: 100, Height: 200})
}

func convertCropSized(t *testing.T, orientation, width int, crop *core.CropBox, details *info.ImageProperties) image.Image {
	t.Helper()
	return convertOriented(t, orientation, &core.ImageConfiguration{Width: width, Format: "jpg", Quality: 90, Crop: crop}, details)
}

// convertOriented runs ImageMagick for real on a stored 200×100 JPEG (left
// red, right blue) with the given EXIF orientation
func convertOriented(t *testing.T, orientation int, ic *core.ImageConfiguration, details *info.ImageProperties) image.Image {
	t.Helper()
	if _, err := exec.LookPath("convert"); err != nil {
		t.Skip("ImageMagick not available")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "rotated.jpg")
	dest := filepath.Join(dir, "crop.jpg")
	WriteOrientedJPEG(t, source, 200, 100, orientation, binary.BigEndian)

	p := cli.Processor{
		Source:             source,
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       details,
	}
	Ok(t, p.CreateImage())

	f, err := os.Open(dest)
	Ok(t, err)
	defer f.Close()
	out, err := jpeg.Decode(f)
	Ok(t, err)
	return out
}

func isRed(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return r > 0xC000 && b < 0x4000
}

func isBlue(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return b > 0xC000 && r < 0x4000
}

// Orientation 6 puts the stored left (red) half on top
func TestCropAppliesOrientationFirst(t *testing.T) {
	out := convertCrop(t, 6, 1000, &core.CropBox{X0: 0, Y0: 250, X1: 1000, Y1: 750})
	// Not enlarged to 1000
	Equals(t, 100, out.Bounds().Dx())
	Equals(t, 100, out.Bounds().Dy())
	Assert(t, isRed(out.At(50, 10)), "top should be red")
	Assert(t, isBlue(out.At(50, 90)), "bottom should be blue")
}

// Orientation 7 (transverse) puts the stored left (red) half at the bottom
func TestCropAppliesMirroredOrientation(t *testing.T) {
	out := convertCrop(t, 7, 0, &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500})
	Equals(t, 100, out.Bounds().Dx())
	Equals(t, 100, out.Bounds().Dy())
	Assert(t, isBlue(out.At(50, 50)), "upper half should be blue")
}

// The first half of the upright image along its long side, for every
// orientation. Stored 200×100 is left red, right blue.
func TestCropAllOrientations(t *testing.T) {
	for orientation, wantRed := range map[int]bool{1: true, 2: false, 3: false, 4: true, 5: true, 6: true, 7: false, 8: false} {
		var out image.Image
		if orientation <= 4 {
			// Upright 200×100: left half
			out = convertCropSized(t, orientation, 0, &core.CropBox{X0: 0, Y0: 0, X1: 500, Y1: 1000},
				&info.ImageProperties{Width: 200, Height: 100})
		} else {
			// Upright 100×200: top half
			out = convertCrop(t, orientation, 0, &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500})
		}
		Equals(t, 100, out.Bounds().Dx())
		Equals(t, 100, out.Bounds().Dy())
		Assert(t, isRed(out.At(50, 50)) == wantRed && isBlue(out.At(50, 50)) == !wantRed,
			"orientation %d: expected red=%v", orientation, wantRed)
	}
}

// A 100x100 center fill of every orientation. The first sample along the
// upright image's long side shows the color that ends up on top (or left).
func TestWidthAndHeightAllOrientations(t *testing.T) {
	for orientation, wantRed := range map[int]bool{1: true, 2: false, 3: false, 4: true, 5: true, 6: true, 7: false, 8: false} {
		ic := &core.ImageConfiguration{Width: 100, Height: 100, Format: "jpg", Quality: 90}
		details := &info.ImageProperties{Width: 100, Height: 200}
		first, last := image.Pt(50, 10), image.Pt(50, 90)
		if orientation <= 4 {
			details = &info.ImageProperties{Width: 200, Height: 100}
			first, last = image.Pt(10, 50), image.Pt(90, 50)
		}

		out := convertOriented(t, orientation, ic, details)
		Equals(t, 100, out.Bounds().Dx())
		Equals(t, 100, out.Bounds().Dy())
		Assert(t, isRed(out.At(first.X, first.Y)) == wantRed && isBlue(out.At(last.X, last.Y)) == wantRed,
			"orientation %d: expected red first=%v", orientation, wantRed)
	}
}

func TestCropShrinksToWidth(t *testing.T) {
	out := convertCrop(t, 6, 40, &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500})
	Equals(t, 40, out.Bounds().Dx())
	Equals(t, 40, out.Bounds().Dy())
	Assert(t, isRed(out.At(20, 20)), "should be red")
}

func TestCropRejectsEmptyBox(t *testing.T) {
	p := cli.Processor{
		Source:      "original",
		Destination: filepath.Join(t.TempDir(), "crop.jpg"),
		ImageConfiguration: &core.ImageConfiguration{Format: "jpg",
			Crop: &core.CropBox{X0: 500, Y0: 0, X1: 501, Y1: 1000}},
		ImageDetails: &info.ImageProperties{Width: 100, Height: 100},
	}
	err := p.CreateImage()
	Assert(t, errors.Is(err, core.ErrInvalidCrop), "expected ErrInvalidCrop, got %v", err)
}
