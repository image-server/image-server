package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/image-server/image-server/core"
)

var reR, reS, reW, reF, reC, reCropLike, reCoord, reFormat *regexp.Regexp
// Patterns without extension (default to jpg)
var reRNoExt, reSNoExt, reWNoExt, reFNoExt *regexp.Regexp

const defaultFormat = "jpg"

func init() {
	// Patterns with extension
	reR = regexp.MustCompile(`^([0-9]+)x([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reS = regexp.MustCompile(`^x([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reW = regexp.MustCompile(`^w([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reF = regexp.MustCompile(`^full_size(?:-q([0-9]+))?\.(\w{3,5})$`)
	// Crop: c<x0>_<y0>_<x1>_<y1>[-w<N>][-q<N>].<ext>. Width and quality have no
	// leading zeros so each crop has a single file name (and cache entry).
	reC = regexp.MustCompile(`^c([0-9.]+)_([0-9.]+)_([0-9.]+)_([0-9.]+)(?:-w([1-9][0-9]{0,4}))?(?:-q([1-9][0-9]{0,2}))?\.(\w{3,5})$`)
	// Names that look like a crop (c, a number-like start, four fields) but
	// don't match reC are rejected rather than served uncropped as custom
	// file names
	reCropLike = regexp.MustCompile(`^c[-+.0-9][^_]*_[^_]*_[^_]*_`)
	// Exactly three decimals, so each box has one file name (and one cache entry)
	reCoord = regexp.MustCompile(`^(?:0\.[0-9]{3}|1\.000)$`)
	// Custom file name i.e. original.png, some-image-name.png, my-file.png
	reFormat = regexp.MustCompile(`^.+\.(\w{3,5})$`)

	// Patterns without extension (default to jpg)
	reRNoExt = regexp.MustCompile(`^([0-9]+)x([0-9]+)(?:-q([0-9]+))?$`)
	reSNoExt = regexp.MustCompile(`^x([0-9]+)(?:-q([0-9]+))?$`)
	reWNoExt = regexp.MustCompile(`^w([0-9]+)(?:-q([0-9]+))?$`)
	reFNoExt = regexp.MustCompile(`^full_size(?:-q([0-9]+))?$`)
}

func NameToConfiguration(sc *core.ServerConfiguration, filename string) (*core.ImageConfiguration, error) {
	var w, h, q, f string
	var quality uint
	var crop *core.CropBox

	if reC.MatchString(filename) {
		m := reC.FindStringSubmatch(filename)
		c, err := parseCrop(m[1:5])
		if err != nil {
			return nil, err
		}
		if q, _ := strconv.Atoi(m[6]); q > 100 {
			return nil, fmt.Errorf("%w: quality must be between 1 and 100", core.ErrInvalidCrop)
		}
		crop = c
		w, h, q, f = m[5], "0", m[6], m[7]
	} else if reCropLike.MatchString(filename) {
		return nil, fmt.Errorf("%w: expected c<x0>_<y0>_<x1>_<y1>[-w<width>][-q<quality>].<ext>", core.ErrInvalidCrop)
	} else if reR.MatchString(filename) {
		m := reR.FindStringSubmatch(filename)
		w, h, q, f = m[1], m[2], m[3], m[4]
	} else if reS.MatchString(filename) {
		m := reS.FindStringSubmatch(filename)
		w, h, q, f = m[1], m[1], m[2], m[3]
	} else if reW.MatchString(filename) {
		m := reW.FindStringSubmatch(filename)
		w, h, q, f = m[1], "0", m[2], m[3]
	} else if reF.MatchString(filename) {
		m := reF.FindStringSubmatch(filename)
		w, h, q, f = "0", "0", m[1], m[2]
	} else if reRNoExt.MatchString(filename) {
		// WxH without extension - default to jpg
		m := reRNoExt.FindStringSubmatch(filename)
		w, h, q, f = m[1], m[2], m[3], defaultFormat
	} else if reSNoExt.MatchString(filename) {
		// xN (square) without extension - default to jpg
		m := reSNoExt.FindStringSubmatch(filename)
		w, h, q, f = m[1], m[1], m[2], defaultFormat
	} else if reWNoExt.MatchString(filename) {
		// wN (width only) without extension - default to jpg
		m := reWNoExt.FindStringSubmatch(filename)
		w, h, q, f = m[1], "0", m[2], defaultFormat
	} else if reFNoExt.MatchString(filename) {
		// full_size without extension - default to jpg
		m := reFNoExt.FindStringSubmatch(filename)
		w, h, q, f = "0", "0", m[1], defaultFormat
	} else {
		if reFormat.MatchString(filename) {
			f = reFormat.FindStringSubmatch(filename)[1]
		}

		return &core.ImageConfiguration{Filename: filename, Format: f}, nil
	}

	width, _ := strconv.Atoi(w)
	height, _ := strconv.Atoi(h)
	quality64, _ := strconv.ParseUint(q, 10, 0)

	if quality64 > 0 {
		quality = uint(quality64)
	} else {
		quality = sc.DefaultQuality
	}

	return &core.ImageConfiguration{Width: width, Height: height, Quality: quality, Format: f, Filename: filename, Crop: crop}, nil
}

// parseCrop reads the four crop coordinates as thousandths
func parseCrop(coords []string) (*core.CropBox, error) {
	var milli [4]int
	for i, c := range coords {
		if !reCoord.MatchString(c) {
			return nil, fmt.Errorf("%w: coordinate %q must be written as 0.ddd or 1.000", core.ErrInvalidCrop, c)
		}
		n, _ := strconv.Atoi(c[:1] + c[2:])
		milli[i] = n
	}

	box := &core.CropBox{X0: milli[0], Y0: milli[1], X1: milli[2], Y1: milli[3]}
	if err := box.Validate(); err != nil {
		return nil, err
	}
	return box, nil
}
