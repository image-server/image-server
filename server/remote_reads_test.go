package server_test

import (
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	"github.com/image-server/image-server/fetcher"
	fetcherhttp "github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
)

// fakeStore serves files by key, like a bucket read with credentials
type fakeStore struct {
	mu    sync.Mutex
	files map[string][]byte
	reads []string
}

func (s *fakeStore) Download(key, destination string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = append(s.reads, key)
	data, ok := s.files[key]
	if !ok {
		return errors.New("not found")
	}
	Must(os.MkdirAll(filepath.Dir(destination), 0700))
	return os.WriteFile(destination, data, 0644)
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// remoteOnlyServer has an image that exists only in the store: nothing is
// on local disk, as after local files expire
func remoteOnlyServer(t *testing.T, sc *core.ServerConfiguration) (http.Handler, *fakeStore) {
	t.Helper()
	base := t.TempDir()
	sc.LocalBasePath = base
	sc.DefaultQuality = 90
	sc.RemoteBasePath = "images"
	p := &paths.Paths{LocalBasePath: base, RemoteBasePath: "images"}
	sc.Adapters = &core.Adapters{Fetcher: &fetcherhttp.Fetcher{}, Paths: p}

	original := filepath.Join(t.TempDir(), "original")
	WriteOrientedJPEG(t, original, 200, 100, 6, binary.BigEndian)
	data, err := os.ReadFile(original)
	Ok(t, err)

	store := &fakeStore{files: map[string][]byte{
		p.RemoteOriginalPath("test_namespace", cropHash): data,
		p.RemoteInfoPath("test_namespace", cropHash):     []byte(`{"width":100}`),
	}}
	fetcher.Store = store
	t.Cleanup(func() { fetcher.Store = nil })
	return server.NewHandler(sc, base), store
}

func get(h http.Handler, method, uri string) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(method, uri, nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	return response
}

func TestReadsMissingFilesFromStore(t *testing.T) {
	h, store := remoteOnlyServer(t, &core.ServerConfiguration{})

	Equals(t, http.StatusOK, get(h, "GET", imageDir+"/w50.jpg").Code)
	Equals(t, http.StatusOK, get(h, "GET", imageDir+"/original").Code)
	r := get(h, "GET", imageDir+"/info.json")
	Equals(t, http.StatusOK, r.Code)
	Equals(t, `{"width":100}`, r.Body.String())
	Equals(t, "application/json", r.Header().Get("Content-Type"))

	// Read by key under the remote base path
	Assert(t, len(store.reads) > 0, "store not used")
	for _, key := range store.reads {
		Assert(t, filepath.HasPrefix(key, "images/test_namespace/6e0/072/682/e66287b662827da75b244a3/"), "unexpected key %s", key)
	}

	Equals(t, http.StatusNotFound, get(h, "GET", "/test_namespace/000/000/000/00000000000000000000000/w50.jpg").Code)
}

// info.json is served from storage whatever --extensions allows, and is
// never generated
func TestInfoJSONIsNotGenerated(t *testing.T) {
	h, store := remoteOnlyServer(t, &core.ServerConfiguration{AllowedExtensions: []string{"jpg"}})
	Equals(t, http.StatusOK, get(h, "GET", imageDir+"/info.json").Code)

	// Not stored: 404, with the original still available
	h, store = remoteOnlyServer(t, &core.ServerConfiguration{})
	delete(store.files, "images/test_namespace/6e0/072/682/e66287b662827da75b244a3/info.json")
	Equals(t, http.StatusNotFound, get(h, "GET", imageDir+"/info.json").Code)
}

func TestCacheHeadersOnSuccessfulReads(t *testing.T) {
	sc := &core.ServerConfiguration{CacheControl: "public, max-age=31536000, immutable", CacheTagHeader: "Cache-Tag"}
	h, _ := remoteOnlyServer(t, sc)

	for _, name := range []string{"w50.jpg", "original", "info.json"} {
		// First read goes through the router, the second is a stored file
		for i := 0; i < 2; i++ {
			for _, method := range []string{"GET", "HEAD"} {
				r := get(h, method, imageDir+"/"+name)
				Equals(t, http.StatusOK, r.Code)
				Equals(t, "public, max-age=31536000, immutable", r.Header().Get("Cache-Control"))
				Equals(t, "test_namespace", r.Header().Get("Cache-Tag"))
			}
		}
	}

	// Errors and other routes are never marked cacheable
	for _, r := range []*httptest.ResponseRecorder{
		get(h, "GET", "/test_namespace/000/000/000/00000000000000000000000/w50.jpg"),
		get(h, "GET", imageDir+"/c0.500_0.000_0.501_1.000.jpg"),
		get(h, "GET", "/status_check"),
		get(h, "POST", imageDir+"/process?outputs=w60.jpg"),
	} {
		Equals(t, "", r.Header().Get("Cache-Control"))
		Equals(t, "", r.Header().Get("Cache-Tag"))
	}
}

func TestCacheHeadersNotSetOnRejectedReads(t *testing.T) {
	sc := &core.ServerConfiguration{
		CacheControl:    "public, max-age=60",
		CacheTagHeader:  "Cache-Tag",
		SignatureConfig: &signature.Config{Enabled: true, RequireForReads: true, Secrets: []string{readSecret}, MaxTTL: time.Hour},
	}
	h, _ := remoteOnlyServer(t, sc)

	r := get(h, "GET", imageDir+"/original")
	Equals(t, http.StatusUnauthorized, r.Code)
	Equals(t, "", r.Header().Get("Cache-Control"))

	r = get(h, "GET", signedRead("GET", imageDir+"/original"))
	Equals(t, http.StatusOK, r.Code)
	Equals(t, "public, max-age=60", r.Header().Get("Cache-Control"))
}

func TestNoCacheHeadersByDefault(t *testing.T) {
	h, _ := remoteOnlyServer(t, &core.ServerConfiguration{})
	r := get(h, "GET", imageDir+"/original")
	Equals(t, http.StatusOK, r.Code)
	Equals(t, "", r.Header().Get("Cache-Control"))
	Equals(t, "", r.Header().Get("Cache-Tag"))
}
