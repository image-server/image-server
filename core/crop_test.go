package core_test

import (
	"errors"
	"testing"

	"github.com/image-server/image-server/core"
	. "github.com/image-server/image-server/test"
)

func TestPixelRectFloorsEdges(t *testing.T) {
	box := core.CropBox{X0: 50, Y0: 640, X1: 390, Y1: 920}
	left, top, w, h, err := box.PixelRect(3024, 4032)
	Ok(t, err)
	// floor(0.05*3024)=151, floor(0.39*3024)=1179, floor(0.64*4032)=2580, floor(0.92*4032)=3709
	Equals(t, []int{151, 2580, 1028, 1129}, []int{left, top, w, h})
}

func TestPixelRectFullImage(t *testing.T) {
	box := core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000}
	left, top, w, h, err := box.PixelRect(560, 420)
	Ok(t, err)
	Equals(t, []int{0, 0, 560, 420}, []int{left, top, w, h})
}

func TestPixelRectEmpty(t *testing.T) {
	// 0.001 of a 100px image is less than one pixel
	box := core.CropBox{X0: 500, Y0: 0, X1: 501, Y1: 1000}
	_, _, _, _, err := box.PixelRect(100, 100)
	Assert(t, errors.Is(err, core.ErrInvalidCrop), "expected ErrInvalidCrop, got %v", err)
}

func TestValidate(t *testing.T) {
	Ok(t, core.CropBox{X0: 0, Y0: 0, X1: 1000, Y1: 1000}.Validate())
	invalid := []core.CropBox{
		{X0: 10, Y0: 0, X1: 10, Y1: 1000},
		{X0: 0, Y0: 500, X1: 1000, Y1: 400},
		{X0: -1, Y0: 0, X1: 1000, Y1: 1000},
		{X0: 0, Y0: 0, X1: 1001, Y1: 1000},
	}
	for _, box := range invalid {
		Assert(t, errors.Is(box.Validate(), core.ErrInvalidCrop), "expected %+v to be invalid", box)
	}
}
