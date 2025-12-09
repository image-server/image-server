package vips

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
)

type Processor struct {
	ImageDetails       *info.ImageProperties
	ImageConfiguration *core.ImageConfiguration
	Source             string
	Destination        string
}

func (p *Processor) CreateImage() error {
	image, err := vips.NewImageFromFile(p.Source)
	if err != nil {
		return fmt.Errorf("vips failed to load image: %w", err)
	}
	defer image.Close()

	ic := p.ImageConfiguration

	// Flatten alpha channel with white background (equivalent to -flatten)
	if image.HasAlpha() {
		bg := &vips.Color{R: 255, G: 255, B: 255}
		if err := image.Flatten(bg); err != nil {
			return fmt.Errorf("vips failed to flatten image: %w", err)
		}
	}

	// Get actual image dimensions from loaded image
	cols := image.Width()
	rows := image.Height()

	// Resize logic matching ImageMagick behavior
	if ic.Height > 0 && ic.Width > 0 {
		if ic.Width != cols || ic.Height != rows {
			// Calculate scale factor to cover target dimensions
			w := float64(ic.Width) / float64(cols)
			h := float64(ic.Height) / float64(rows)
			scale := math.Max(w, h)

			if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
				return fmt.Errorf("vips failed to resize image: %w", err)
			}
		}

		// Crop or extend to exact dimensions (equivalent to -extent with -gravity center)
		currentWidth := image.Width()
		currentHeight := image.Height()

		if currentWidth != ic.Width || currentHeight != ic.Height {
			// Calculate offsets for center gravity
			left := (currentWidth - ic.Width) / 2
			top := (currentHeight - ic.Height) / 2

			if left >= 0 && top >= 0 {
				// Image is larger - crop
				if err := image.ExtractArea(left, top, ic.Width, ic.Height); err != nil {
					return fmt.Errorf("vips failed to crop image: %w", err)
				}
			} else {
				// Image is smaller - embed with white background
				embedLeft := 0
				embedTop := 0
				if left < 0 {
					embedLeft = -left
				}
				if top < 0 {
					embedTop = -top
				}
				bg := &vips.Color{R: 255, G: 255, B: 255}
				if err := image.EmbedBackground(embedLeft, embedTop, ic.Width, ic.Height, bg); err != nil {
					return fmt.Errorf("vips failed to embed image: %w", err)
				}
			}
		}
	} else if ic.Width > 0 {
		// Width-only resize (maintains aspect ratio)
		scale := float64(ic.Width) / float64(cols)
		if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
			return fmt.Errorf("vips failed to resize image: %w", err)
		}
	}

	// Strip metadata (equivalent to -strip)
	image.RemoveMetadata()

	// Export based on format
	format := strings.ToLower(ic.Format)
	quality := int(ic.Quality)
	if quality == 0 {
		quality = 75 // default quality
	}

	var buf []byte

	switch format {
	case "jpg", "jpeg":
		params := vips.NewJpegExportParams()
		params.Quality = quality
		params.StripMetadata = true
		buf, _, err = image.ExportJpeg(params)
	case "png":
		params := vips.NewPngExportParams()
		params.StripMetadata = true
		buf, _, err = image.ExportPng(params)
	case "webp":
		params := vips.NewWebpExportParams()
		params.Quality = quality
		params.StripMetadata = true
		buf, _, err = image.ExportWebp(params)
	case "gif":
		params := vips.NewGifExportParams()
		buf, _, err = image.ExportGIF(params)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("vips failed to export image: %w", err)
	}

	// Write buffer to destination file
	return os.WriteFile(p.Destination, buf, 0644)
}
