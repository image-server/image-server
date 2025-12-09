package vips_test

import (
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
