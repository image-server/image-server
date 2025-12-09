package parser

import (
	"regexp"
	"strconv"

	"github.com/image-server/image-server/core"
)

var reR, reS, reW, reF, reFormat *regexp.Regexp
// Patterns without extension (default to jpg)
var reRNoExt, reSNoExt, reWNoExt, reFNoExt *regexp.Regexp

const defaultFormat = "jpg"

func init() {
	// Patterns with extension
	reR = regexp.MustCompile(`^([0-9]+)x([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reS = regexp.MustCompile(`^x([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reW = regexp.MustCompile(`^w([0-9]+)(?:-q([0-9]+))?\.(\w{3,5})$`)
	reF = regexp.MustCompile(`^full_size(?:-q([0-9]+))?\.(\w{3,5})$`)
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

	if reR.MatchString(filename) {
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

	return &core.ImageConfiguration{Width: width, Height: height, Quality: quality, Format: f, Filename: filename}, nil
}
