package core

import (
	"errors"
	"fmt"
)

// ErrInvalidCrop is returned for crop boxes that are malformed or select no pixels
var ErrInvalidCrop = errors.New("invalid crop")

// CropBox is a rectangle in thousandths of the upright image (0..1000),
// given as left, top, right and bottom edges
type CropBox struct {
	X0, Y0, X1, Y1 int
}

// Validate checks the box is inside the image and not inverted
func (c CropBox) Validate() error {
	for _, v := range []int{c.X0, c.Y0, c.X1, c.Y1} {
		if v < 0 || v > 1000 {
			return fmt.Errorf("%w: coordinates must be between 0 and 1", ErrInvalidCrop)
		}
	}
	if c.X0 >= c.X1 || c.Y0 >= c.Y1 {
		return fmt.Errorf("%w: x0 must be less than x1 and y0 less than y1", ErrInvalidCrop)
	}
	return nil
}

// PixelRect converts the box to a pixel rectangle on an image of the given
// size. Edges are floor(coord * size), so a box ending at 1.000 reaches the edge.
func (c CropBox) PixelRect(width, height int) (left, top, w, h int, err error) {
	if err := c.Validate(); err != nil {
		return 0, 0, 0, 0, err
	}
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("%w: image has no pixels (%dx%d)", ErrInvalidCrop, width, height)
	}

	edge := func(milli, size int) int { return int(int64(milli) * int64(size) / 1000) }
	left = edge(c.X0, width)
	top = edge(c.Y0, height)
	w = edge(c.X1, width) - left
	h = edge(c.Y1, height) - top
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("%w: crop selects no pixels on a %dx%d image", ErrInvalidCrop, width, height)
	}
	return left, top, w, h, nil
}
