package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	"github.com/image-server/image-server/fetcher"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
)

const imagePath = "/test_namespace/6e0/072/682/e66287b662827da75b244a3"

// imageAdminServer is adminServer with image delete enabled
func imageAdminServer(t *testing.T) (http.Handler, *core.ServerConfiguration) {
	t.Helper()
	router, sc := adminServer(t, "")
	sc.NamespaceAdmin.ImageDelete = true
	return router, sc
}

// quietLogger ignores every metric; rejectionRecorder overrides the admin
// ones, so tests that also process images record only those
type quietLogger struct{}

func (quietLogger) ImagePosted()                                      {}
func (quietLogger) ImagePostingFailed()                               {}
func (quietLogger) ImageProcessed(*core.ImageConfiguration)           {}
func (quietLogger) ImageAlreadyProcessed(*core.ImageConfiguration)    {}
func (quietLogger) ImageProcessedWithErrors(*core.ImageConfiguration) {}
func (quietLogger) AllImagesAlreadyProcessed(string, string, string)  {}
func (quietLogger) SourceDownloaded()                                 {}
func (quietLogger) OriginalDownloaded(string, string)                 {}
func (quietLogger) OriginalDownloadFailed(string)                     {}
func (quietLogger) OriginalDownloadSkipped(string)                    {}
func (quietLogger) RequestLatency(string, time.Time)                  {}
func (quietLogger) OriginalUploaded(*core.ImageProperties, string)    {}
func (quietLogger) NamespaceAdmin(string, string, string)             {}

func recordAdminMetricsQuietly(t *testing.T) chan string {
	events := recordAdminMetrics(t)
	logger.Loggers = []core.Logger{&rejectionRecorder{Logger: quietLogger{}, events: events}}
	return events
}

func imageExists(sc *core.ServerConfiguration) bool {
	_, err := os.Stat(filepath.Join(sc.LocalBasePath, imagePath))
	return err == nil
}

func TestDeleteImage(t *testing.T) {
	router, sc := imageAdminServer(t)
	events := recordAdminMetricsQuietly(t)

	// A variant is generated and its decoded source cached
	Equals(t, http.StatusOK, getCrop(router, "w50.jpg").Code)
	// Another image in the namespace stays
	other := filepath.Join(sc.LocalBasePath, "test_namespace", "31e", "8b3", "187", "a9f63f26d58c88bf09a7bbd")
	Ok(t, os.MkdirAll(other, 0700))
	Ok(t, os.WriteFile(filepath.Join(other, "original"), []byte("x"), 0644))

	r := serve(router, "DELETE", signed("DELETE", imagePath, adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"namespace": "test_namespace"`, r.Body.String())
	Matches(t, `"hash": "6e0072682e66287b662827da75b244a3"`, r.Body.String())
	Matches(t, `"status": "deleted"`, r.Body.String())
	Matches(t, `"files_deleted": 2`, r.Body.String())
	Equals(t, "delete_image deleted ", nextEvent(t, events))
	Equals(t, false, imageExists(sc))
	Equals(t, true, nsExists(sc, "test_namespace"))
	_, err := os.Stat(filepath.Join(other, "original"))
	Ok(t, err)

	// Nothing serves the image afterwards, even with files served from disk
	static := server.NewHandler(sc, sc.LocalBasePath)
	for _, name := range []string{"original", "w50.jpg", "w60.jpg", "info.json"} {
		Equals(t, http.StatusNotFound, getCrop(router, name).Code)
		Equals(t, http.StatusNotFound, serve(static, "GET", imagePath+"/"+name).Code)
	}

	// A retry is safe
	r = serve(router, "DELETE", signed("DELETE", imagePath, adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "missing"`, r.Body.String())
	Matches(t, `"files_deleted": 0`, r.Body.String())
	Equals(t, "delete_image missing ", nextEvent(t, events))
}

func TestDeleteImageTrailingSlash(t *testing.T) {
	router, sc := imageAdminServer(t)
	r := serve(router, "DELETE", signed("DELETE", imagePath+"/", adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "deleted"`, r.Body.String())
	Equals(t, false, imageExists(sc))
}

func TestDeleteImageMissing(t *testing.T) {
	router, _ := imageAdminServer(t)
	r := serve(router, "DELETE", signed("DELETE", "/other_namespace/6e0/072/682/e66287b662827da75b244a3", adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "missing"`, r.Body.String())
	Matches(t, `"files_deleted": 0`, r.Body.String())
}

func TestDeleteImageDisabled(t *testing.T) {
	router, sc := adminServer(t, ".*")
	events := recordAdminMetrics(t)

	// Namespace admin on, image delete off
	Equals(t, http.StatusNotFound, serve(router, "DELETE", signed("DELETE", imagePath, adminSecret)).Code)
	Equals(t, "delete_image rejected disabled", nextEvent(t, events))

	sc.NamespaceAdmin = nil
	Equals(t, http.StatusNotFound, serve(router, "DELETE", signed("DELETE", imagePath, adminSecret)).Code)
	Equals(t, "delete_image rejected disabled", nextEvent(t, events))
	Equals(t, true, imageExists(sc))
}

func TestDeleteImageRequiresAdminSignature(t *testing.T) {
	router, sc := imageAdminServer(t)
	sc.SignatureConfig = &signature.Config{Enabled: true, Secrets: []string{"upload-secret"}, MaxTTL: time.Hour}
	events := recordAdminMetrics(t)

	Equals(t, http.StatusUnauthorized, serve(router, "DELETE", imagePath).Code)
	Equals(t, "delete_image rejected bad_signature", nextEvent(t, events))
	Equals(t, http.StatusUnauthorized, serve(router, "DELETE", signed("DELETE", imagePath, "upload-secret")).Code)
	Equals(t, "delete_image rejected wrong_secret", nextEvent(t, events))

	// A namespace signature does not cover its images: paths must match exactly
	uri := signed("DELETE", "/test_namespace", adminSecret)
	uri = strings.Replace(uri, "/test_namespace?", imagePath+"?", 1)
	Equals(t, http.StatusUnauthorized, serve(router, "DELETE", uri).Code)
	Equals(t, "delete_image rejected bad_signature", nextEvent(t, events))
	// Nor does a signature for a read of the same image
	uri = signed("GET", imagePath, adminSecret)
	Equals(t, http.StatusUnauthorized, serve(router, "DELETE", uri).Code)
	Equals(t, "delete_image rejected bad_signature", nextEvent(t, events))

	Equals(t, true, imageExists(sc))
}

func TestDeleteImageInvalidNamespace(t *testing.T) {
	router, sc := imageAdminServer(t)
	events := recordAdminMetrics(t)
	path := "/tmp/6e0/072/682/e66287b662827da75b244a3"
	Equals(t, http.StatusBadRequest, serve(router, "DELETE", signed("DELETE", path, adminSecret)).Code)
	Equals(t, "delete_image rejected invalid_name", nextEvent(t, events))

	// Not an image path: no route
	for _, path := range []string{
		"/test_namespace/6e0/072/682/e66287b662827da75b244a",
		"/test_namespace/6E0/072/682/e66287b662827da75b244a3",
		"/test_namespace/6e0/072/682/e66287b662827da75b244a3/original",
	} {
		if r := serve(router, "DELETE", signed("DELETE", path, adminSecret)); r.Code == http.StatusOK {
			t.Errorf("%s: got 200", path)
		}
	}
	Equals(t, true, imageExists(sc))
}

// bucketStore reads files from a bucket, as with --s3_reads
type bucketStore struct{ b *bucket }

func (s bucketStore) Download(key, destination string) error {
	data, found, _ := s.b.GetBytes(key)
	if !found {
		return errors.New("not found")
	}
	Must(os.MkdirAll(filepath.Dir(destination), 0700))
	return os.WriteFile(destination, data, 0644)
}

// With the S3 uploader, the image's objects are deleted from the bucket
func TestDeleteImageWithRemoteStorage(t *testing.T) {
	router, sc := imageAdminServer(t)
	sc.UploaderType = "s3"
	sc.RemoteBasePath = "images"
	sc.Adapters.Paths = &paths.Paths{LocalBasePath: sc.LocalBasePath, RemoteBasePath: "images"}
	original, err := os.ReadFile(filepath.Join(sc.LocalBasePath, imagePath, "original"))
	Ok(t, err)
	b := &bucket{objects: map[string][]byte{
		"images" + imagePath + "/original":                                    original,
		"images" + imagePath + "/info.json":                                   []byte(`{"width":200}`),
		"images" + imagePath + "/w50.jpg":                                     []byte("x"),
		"images/test_namespace/31e/8b3/187/a9f63f26d58c88bf09a7bbd/original":  []byte("x"),
		"images/test_namespace2/6e0/072/682/e66287b662827da75b244a3/original": []byte("x"),
	}}
	t.Cleanup(server.SetNamespaceRemote(b))
	fetcher.Store = bucketStore{b}
	t.Cleanup(func() { fetcher.Store = nil })

	r := serve(router, "DELETE", signed("DELETE", imagePath, adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "deleted"`, r.Body.String())
	Matches(t, `"files_deleted": 3`, r.Body.String())
	keys, _ := b.ListKeys("images/")
	Equals(t, []string{
		"images/test_namespace/31e/8b3/187/a9f63f26d58c88bf09a7bbd/original",
		"images/test_namespace2/6e0/072/682/e66287b662827da75b244a3/original",
	}, keys)
	Equals(t, false, imageExists(sc))

	// Not read back from the bucket
	for _, name := range []string{"original", "w50.jpg", "info.json"} {
		Equals(t, http.StatusNotFound, getCrop(router, name).Code)
	}

	r = serve(router, "DELETE", signed("DELETE", imagePath, adminSecret))
	Matches(t, `"status": "missing"`, r.Body.String())
}

// failingBucket fails every delete, like a store that is down
type failingBucket struct{ *bucket }

func (failingBucket) DeleteKeys([]string) error { return errors.New("store unavailable") }

func TestDeleteImageRemoteError(t *testing.T) {
	router, sc := imageAdminServer(t)
	sc.UploaderType = "s3"
	b := failingBucket{&bucket{objects: map[string][]byte{"test_namespace/6e0/072/682/e66287b662827da75b244a3/original": []byte("x")}}}
	t.Cleanup(server.SetNamespaceRemote(b))

	Equals(t, http.StatusInternalServerError, serve(router, "DELETE", signed("DELETE", imagePath, adminSecret)).Code)
	Equals(t, true, imageExists(sc))
}

func TestSignatureMiddlewareSkipsDeleteImage(t *testing.T) {
	m := server.NewSignatureMiddleware(&signature.Config{Enabled: true, Secrets: []string{"upload-secret"}, MaxTTL: time.Hour})
	for _, c := range []struct {
		method, path string
		skipped      bool
	}{
		{"DELETE", imagePath, true},
		{"DELETE", imagePath + "/", true},
		{"DELETE", imagePath + "/original", false},
		{"POST", imagePath + "/original", false},
	} {
		called := false
		request, _ := http.NewRequest(c.method, signed(c.method, c.path, adminSecret), nil)
		m.ServeHTTP(httptest.NewRecorder(), request, func(http.ResponseWriter, *http.Request) { called = true })
		if called != c.skipped {
			t.Errorf("%s %s: next called=%v, want %v", c.method, c.path, called, c.skipped)
		}
	}
}
