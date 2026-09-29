package cli_test

import (
	"fmt"
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
