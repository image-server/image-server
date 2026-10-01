package server_test

import (
	"encoding/binary"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/image-server/image-server/core"
	fetcher "github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
)

const cropHash = "6e0072682e66287b662827da75b244a3"

// cropServer serves a local 100×200 upright original (stored sideways with
// EXIF orientation 6, red on top) and no remote store
func cropServer(t *testing.T) http.Handler {
	t.Helper()
	base := t.TempDir()
	sc := &core.ServerConfiguration{LocalBasePath: base, DefaultQuality: 90}
	p := &paths.Paths{LocalBasePath: base}
	sc.Adapters = &core.Adapters{Fetcher: &fetcher.Fetcher{}, Paths: p}

	original := p.LocalOriginalPath("test_namespace", cropHash)
	Ok(t, os.MkdirAll(filepath.Dir(original), 0700))
	WriteOrientedJPEG(t, original, 200, 100, 6, binary.BigEndian)
	return server.NewRouter(sc)
}

func getCrop(router http.Handler, filename string) *httptest.ResponseRecorder {
	uri := "/test_namespace/6e0/072/682/e66287b662827da75b244a3/" + filename
	request, _ := http.NewRequest("GET", uri, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestResizeHandlerCrop(t *testing.T) {
	response := getCrop(cropServer(t), "c0.000_0.000_1.000_0.500-w50-q90.jpg")
	Equals(t, http.StatusOK, response.Code)

	out, err := jpeg.Decode(response.Body)
	Ok(t, err)
	Equals(t, 50, out.Bounds().Dx())
	Equals(t, 50, out.Bounds().Dy())
	r, _, b, _ := out.At(25, 25).RGBA()
	Assert(t, r > 0xC000 && b < 0x4000, "upper half of the upright image should be red")
}

func TestResizeHandlerRejectsInvalidCrop(t *testing.T) {
	router := cropServer(t)
	for _, filename := range []string{
		"c0.390_0.000_0.050_0.500.jpg",  // x0 > x1
		"c0.05_0.000_1.000_0.500.jpg",   // two decimals
		"c0.000_0.000_1.500_0.500.jpg",  // out of range
		"c0.000_0.000_+1.500_1.000.jpg", // sign
		"c0.000_0.000_1e0_1.000.jpg",    // exponent
	} {
		response := getCrop(router, filename)
		Equals(t, http.StatusBadRequest, response.Code)
	}
}

func TestResizeHandlerRejectsCropSmallerThanAPixel(t *testing.T) {
	// 0.001 of a 100px wide image is less than one pixel
	response := getCrop(cropServer(t), "c0.500_0.000_0.501_1.000.jpg")
	Equals(t, http.StatusBadRequest, response.Code)
	Matches(t, "invalid crop", response.Body.String())
}

func postProcess(router http.Handler, outputs string) *httptest.ResponseRecorder {
	uri := "/test_namespace/6e0/072/682/e66287b662827da75b244a3/process?outputs=" + outputs
	request, _ := http.NewRequest("POST", uri, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestResizeManyHandlerRejectsInvalidCrop(t *testing.T) {
	router := cropServer(t)
	Equals(t, http.StatusBadRequest, postProcess(router, "w50.jpg,c0.390_0.000_0.050_0.500.jpg").Code)
	// Only detectable once the original's size is known
	Equals(t, http.StatusBadRequest, postProcess(router, "c0.500_0.000_0.501_1.000.jpg").Code)
	Equals(t, http.StatusOK, postProcess(router, "c0.000_0.000_1.000_0.500-w50.jpg").Code)
}

func TestNewImageHandlerRejectsInvalidCropOutput(t *testing.T) {
	request, err := newUploadRequest("/test_namespace?outputs=x300.jpg,c0.390_0.000_0.050_0.500.jpg", "../test/images/a.jpg")
	Ok(t, err)
	response := httptest.NewRecorder()
	cropServer(t).ServeHTTP(response, request)
	Equals(t, http.StatusBadRequest, response.Code)
}
