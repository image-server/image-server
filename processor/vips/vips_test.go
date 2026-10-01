package vips_test

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/processor/vips"
	. "github.com/image-server/image-server/test"
)

func TestVipsAvailable(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}
	Equals(t, true, vips.Available)
}

func TestSupportedOutputFormats(t *testing.T) {
	Equals(t, true, vips.SupportedFormat("jpg"))
	Equals(t, true, vips.SupportedFormat("jpeg"))
	Equals(t, true, vips.SupportedFormat("png"))
	Equals(t, true, vips.SupportedFormat("webp"))
	Equals(t, true, vips.SupportedFormat("gif"))
	Equals(t, false, vips.SupportedFormat("pdf"))
	Equals(t, false, vips.SupportedFormat("svg"))
}

func TestSupportedInputFormats(t *testing.T) {
	Equals(t, true, vips.SupportedInputFormat("image/jpeg"))
	Equals(t, true, vips.SupportedInputFormat("image/png"))
	Equals(t, true, vips.SupportedInputFormat("image/webp"))
	Equals(t, true, vips.SupportedInputFormat("image/gif"))
	Equals(t, true, vips.SupportedInputFormat("application/pdf"))
	Equals(t, false, vips.SupportedInputFormat("image/svg+xml"))
}

func TestVipsResizeJpeg(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "resized.jpg")

	ic := &core.ImageConfiguration{Width: 100, Height: 100, Format: "jpg", Quality: 85}
	id := &info.ImageProperties{Width: 560, Height: 420}

	p := vips.Processor{
		Source:             "../../test/images/wine.jpg",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)

	// Verify output dimensions
	i := info.Info{Path: dest}
	details, err := i.ImageDetails()
	Ok(t, err)
	Equals(t, 100, details.Width)
	Equals(t, 100, details.Height)
}

func TestVipsResizePng(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "resized.png")

	ic := &core.ImageConfiguration{Width: 50, Height: 50, Format: "png", Quality: 0}
	id := &info.ImageProperties{Width: 800, Height: 600}

	p := vips.Processor{
		Source:             "../../test/images/a.png",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)
}

func TestVipsResizeWebp(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "resized.webp")

	ic := &core.ImageConfiguration{Width: 100, Height: 0, Format: "webp", Quality: 80}
	id := &info.ImageProperties{Width: 550, Height: 368}

	p := vips.Processor{
		Source:             "../../test/images/a.webp",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)
}

func TestVipsFormatConversion(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "converted.webp")

	// Convert JPEG to WebP
	ic := &core.ImageConfiguration{Width: 200, Height: 150, Format: "webp", Quality: 75}
	id := &info.ImageProperties{Width: 560, Height: 420}

	p := vips.Processor{
		Source:             "../../test/images/wine.jpg",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists and is WebP
	_, err = os.Stat(dest)
	Ok(t, err)
}

func TestVipsWidthOnlyResize(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "w200.jpg")

	ic := &core.ImageConfiguration{Width: 200, Height: 0, Format: "jpg", Quality: 85}
	id := &info.ImageProperties{Width: 560, Height: 420}

	p := vips.Processor{
		Source:             "../../test/images/wine.jpg",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)

	// Verify width is correct (height should maintain aspect ratio)
	i := info.Info{Path: dest}
	details, err := i.ImageDetails()
	Ok(t, err)
	Equals(t, 200, details.Width)
}

func TestVipsFullSize(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "full_size.jpg")

	ic := &core.ImageConfiguration{Width: 0, Height: 0, Format: "jpg", Quality: 85}
	id := &info.ImageProperties{Width: 560, Height: 420}

	p := vips.Processor{
		Source:             "../../test/images/wine.jpg",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)
}

func TestVipsUnsupportedFormat(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "output.tiff")

	ic := &core.ImageConfiguration{Width: 100, Height: 100, Format: "tiff", Quality: 85}
	id := &info.ImageProperties{Width: 560, Height: 420}

	p := vips.Processor{
		Source:             "../../test/images/wine.jpg",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Assert(t, err != nil, "expected error for unsupported format")
}

func TestVipsPdfToJpeg(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "pdf_converted.jpg")

	// Convert PDF to JPEG
	ic := &core.ImageConfiguration{Width: 200, Height: 0, Format: "jpg", Quality: 85}
	id := &info.ImageProperties{Width: 612, Height: 792, ContentType: "application/pdf"}

	p := vips.Processor{
		Source:             "../../test/images/test.pdf",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	Ok(t, err)

	// Verify it's a valid JPEG
	i := info.Info{Path: dest}
	details, err := i.ImageDetails()
	Ok(t, err)
	Equals(t, "image/jpeg", details.ContentType)
}

func TestVipsPdfToPng(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "pdf_converted.png")

	// Convert PDF to PNG
	ic := &core.ImageConfiguration{Width: 300, Height: 400, Format: "png", Quality: 0}
	id := &info.ImageProperties{Width: 612, Height: 792, ContentType: "application/pdf"}

	p := vips.Processor{
		Source:             "../../test/images/test.pdf",
		Destination:        dest,
		ImageConfiguration: ic,
		ImageDetails:       id,
	}

	err := p.CreateImage()
	Ok(t, err)

	// Verify output file exists and dimensions
	_, err = os.Stat(dest)
	Ok(t, err)

	i := info.Info{Path: dest}
	details, err := i.ImageDetails()
	Ok(t, err)
	Equals(t, "image/png", details.ContentType)
	Equals(t, 300, details.Width)
	Equals(t, 400, details.Height)
}

// A sideways phone photo: pixels stored 200×100 with EXIF orientation 6
// (display turns 90° clockwise). Metadata is stripped on export, so the
// rotation must be baked into the pixels: left (red) half ends up on top.
func TestVipsAppliesExifOrientation(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "rotated.jpg")
	dest := filepath.Join(tmpDir, "w50.jpg")
	WriteOrientedJPEG(t, source, 200, 100, 6, binary.BigEndian)

	p := vips.Processor{
		Source:             source,
		Destination:        dest,
		ImageConfiguration: &core.ImageConfiguration{Width: 50, Format: "jpg", Quality: 90},
		ImageDetails:       &info.ImageProperties{Width: 100, Height: 200},
	}
	Ok(t, p.CreateImage())

	f, err := os.Open(dest)
	Ok(t, err)
	defer f.Close()
	out, err := jpeg.Decode(f)
	Ok(t, err)
	Equals(t, 50, out.Bounds().Dx())
	Equals(t, 100, out.Bounds().Dy())
	top, _, topBlue, _ := out.At(25, 10).RGBA()
	_, _, bottom, _ := out.At(25, 90).RGBA()
	Assert(t, top > 0xC000 && topBlue < 0x4000, "top should be red")
	Assert(t, bottom > 0xC000, "bottom should be blue")
}

func cropWithVips(t *testing.T, source string, ic *core.ImageConfiguration, details *info.ImageProperties) image.Image {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "crop.jpg")
	p := vips.Processor{Source: source, Destination: dest, ImageConfiguration: ic, ImageDetails: details}
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

// Stored 200×100 (left red, right blue) with orientation 6 is upright 100×200,
// red on top. The crop is measured on the upright image.
func TestVipsCropsUprightImage(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	source := filepath.Join(t.TempDir(), "rotated.jpg")
	WriteOrientedJPEG(t, source, 200, 100, 6, binary.BigEndian)
	details := &info.ImageProperties{Width: 100, Height: 200}

	top := cropWithVips(t, source, &core.ImageConfiguration{Format: "jpg", Quality: 90,
		Crop: &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500}}, details)
	Equals(t, 100, top.Bounds().Dx())
	Equals(t, 100, top.Bounds().Dy())
	Assert(t, isRed(top.At(50, 10)) && isRed(top.At(50, 90)), "upper half should be red")

	// Straddles the boundary: 100×100 starting at y=50
	middle := cropWithVips(t, source, &core.ImageConfiguration{Format: "jpg", Quality: 90,
		Crop: &core.CropBox{X0: 0, Y0: 250, X1: 1000, Y1: 750}}, details)
	Equals(t, 100, middle.Bounds().Dy())
	Assert(t, isRed(middle.At(50, 10)), "top of middle crop should be red")
	Assert(t, isBlue(middle.At(50, 90)), "bottom of middle crop should be blue")
}

// Orientation 7 (transverse) puts the stored left half at the bottom
func TestVipsCropsMirroredOrientation(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	source := filepath.Join(t.TempDir(), "transverse.jpg")
	WriteOrientedJPEG(t, source, 200, 100, 7, binary.LittleEndian)

	top := cropWithVips(t, source, &core.ImageConfiguration{Format: "jpg", Quality: 90,
		Crop: &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500}}, &info.ImageProperties{Width: 100, Height: 200})
	Equals(t, 100, top.Bounds().Dx())
	Equals(t, 100, top.Bounds().Dy())
	Assert(t, isBlue(top.At(50, 50)), "upper half should be blue")
}

// Same cases as the ImageMagick processor: the first half of the upright
// image along its long side. Stored 200×100 is left red, right blue.
func TestVipsCropAllOrientations(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	for orientation, wantRed := range map[int]bool{1: true, 2: false, 3: false, 4: true, 5: true, 6: true, 7: false, 8: false} {
		source := filepath.Join(t.TempDir(), "oriented.jpg")
		WriteOrientedJPEG(t, source, 200, 100, orientation, binary.BigEndian)

		crop := &core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 500}
		details := &info.ImageProperties{Width: 100, Height: 200}
		if orientation <= 4 {
			crop = &core.CropBox{X0: 0, Y0: 0, X1: 500, Y1: 1000}
			details = &info.ImageProperties{Width: 200, Height: 100}
		}
		out := cropWithVips(t, source, &core.ImageConfiguration{Format: "jpg", Quality: 90, Crop: crop}, details)
		Equals(t, 100, out.Bounds().Dx())
		Equals(t, 100, out.Bounds().Dy())
		Assert(t, isRed(out.At(50, 50)) == wantRed && isBlue(out.At(50, 50)) == !wantRed,
			"orientation %d: expected red=%v", orientation, wantRed)
	}
}

func TestVipsCropWidthShrinksOnly(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	crop := &core.CropBox{X0: 0, Y0: 0, X1: 500, Y1: 500} // 400×300 of wine.jpg
	details := &info.ImageProperties{Width: 800, Height: 600}

	shrunk := cropWithVips(t, "../../test/images/wine.jpg",
		&core.ImageConfiguration{Width: 140, Format: "jpg", Quality: 90, Crop: crop}, details)
	Equals(t, 140, shrunk.Bounds().Dx())
	Equals(t, 105, shrunk.Bounds().Dy())

	notEnlarged := cropWithVips(t, "../../test/images/wine.jpg",
		&core.ImageConfiguration{Width: 1024, Format: "jpg", Quality: 90, Crop: crop}, details)
	Equals(t, 400, notEnlarged.Bounds().Dx())
	Equals(t, 300, notEnlarged.Bounds().Dy())
}

func TestVipsRejectsEmptyCrop(t *testing.T) {
	if !vips.Available {
		t.Skip("vips not available, skipping tests")
	}

	p := vips.Processor{
		Source:      "../../test/images/wine.jpg",
		Destination: filepath.Join(t.TempDir(), "empty.jpg"),
		ImageConfiguration: &core.ImageConfiguration{Format: "jpg", Quality: 90,
			Crop: &core.CropBox{X0: 500, Y0: 0, X1: 501, Y1: 1000}}, // 0.8px wide
		ImageDetails: &info.ImageProperties{Width: 800, Height: 600},
	}
	err := p.CreateImage()
	Assert(t, errors.Is(err, core.ErrInvalidCrop), "expected ErrInvalidCrop, got %v", err)
}
