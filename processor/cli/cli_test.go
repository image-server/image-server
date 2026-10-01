package cli_test

import (
	"encoding/binary"
	"errors"
	"fmt"
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

	expected := []string{"-auto-orient", "-strip", "-format", "jpg", "-flatten", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/full_size.jpg"}
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

	expected := []string{"-auto-orient", "-strip", "-format", "jpg", "-flatten", "-resize", "600", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/w600.jpg"}

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

	expected := []string{"-auto-orient", "-strip", "-format", "jpg", "-flatten", "-extent", "600x500", "-gravity", "center", "-background", "rgba(255,255,255,1)", "-quality", "85", "public/test/00/of/rA/original", "public/test/00/of/rA/600x500.jpg"}

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
	Equals(t, "ImageMagick failed to process the image: convert -auto-orient -strip -format jpg -flatten -resize 600 -background rgba(255,255,255,1) -quality 85 test/images/empty.jpg public/test/00/of/rA/empty.jpg", errorMsg)
}

func TestImageWithCrop(t *testing.T) {
	ic := &core.ImageConfiguration{Width: 100, Format: "jpg", Quality: 90, Filename: "c0.050_0.640_0.390_0.920-w100-q90.jpg",
		Crop: &core.CropBox{X0: 50, Y0: 640, X1: 390, Y1: 920}}
	id := &info.ImageProperties{Width: 3024, Height: 4032}

	expected := []string{"-auto-orient", "-strip", "-format", "jpg", "-flatten", "-crop", "1028x1129+151+2580", "+repage", "-resize", "100>", "-background", "rgba(255,255,255,1)", "-quality", "90", "original", "crop.jpg"}

	p := cli.Processor{
		Source:             "original",
		Destination:        "crop.jpg",
		ImageConfiguration: ic,
		ImageDetails:       id,
	}
	Equals(t, expected, p.CommandArgs())
}

// Runs ImageMagick for real: the crop must be measured on the upright image
func TestCropAppliesOrientationFirst(t *testing.T) {
	if _, err := exec.LookPath("convert"); err != nil {
		t.Skip("ImageMagick not available")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "rotated.jpg")
	dest := filepath.Join(dir, "crop.jpg")
	// Stored 200×100 (left red, right blue), orientation 6: upright 100×200, red on top
	WriteOrientedJPEG(t, source, 200, 100, 6, binary.BigEndian)

	p := cli.Processor{
		Source:      source,
		Destination: dest,
		ImageConfiguration: &core.ImageConfiguration{Width: 1000, Format: "jpg", Quality: 90,
			Crop: &core.CropBox{X0: 0, Y0: 250, X1: 1000, Y1: 750}},
		ImageDetails: &info.ImageProperties{Width: 100, Height: 200},
	}
	Ok(t, p.CreateImage())

	f, err := os.Open(dest)
	Ok(t, err)
	defer f.Close()
	out, err := jpeg.Decode(f)
	Ok(t, err)
	// Not enlarged to 1000
	Equals(t, 100, out.Bounds().Dx())
	Equals(t, 100, out.Bounds().Dy())
	r, _, b, _ := out.At(50, 10).RGBA()
	Assert(t, r > 0xC000 && b < 0x4000, "top should be red")
	r, _, b, _ = out.At(50, 90).RGBA()
	Assert(t, b > 0xC000 && r < 0x4000, "bottom should be blue")
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
