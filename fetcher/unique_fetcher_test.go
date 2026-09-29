package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/fetcher"
	"github.com/image-server/image-server/paths"
	. "github.com/image-server/image-server/test"
)

func TestUniqueFetcherWithoutRemoteUsesLocalFile(t *testing.T) {
	local := filepath.Join(t.TempDir(), "w300.jpg")
	Ok(t, os.WriteFile(local, []byte("x"), 0644))

	downloaded, err := fetcher.NewUniqueFetcher("", local).Fetch()
	Ok(t, err)
	Equals(t, false, downloaded)
}

func TestUniqueFetcherWithoutRemoteMissingFile(t *testing.T) {
	local := filepath.Join(t.TempDir(), "w300.jpg")

	_, err := fetcher.NewUniqueFetcher("", local).Fetch()
	Equals(t, fetcher.ErrNoRemoteStore, err)
	_, statErr := os.Stat(filepath.Dir(local))
	Ok(t, statErr) // t.TempDir exists; nothing else created
}

// A missing variant with no remote store must not make any HTTP request:
// the caller goes straight to processing it from the local original.
func TestProcessedFetcherSkipsRemoteWithoutBaseURL(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &paths.Paths{LocalBasePath: t.TempDir(), RemoteBasePath: "p"}
	err := fetcher.NewProcessedFetcher(p).Fetch(testImageConfiguration())
	Equals(t, fetcher.ErrNoRemoteStore, err)
	Equals(t, int32(0), atomic.LoadInt32(&requests))

	// With a remote store the variant is looked up there first.
	p.RemoteBaseURL = srv.URL
	err = fetcher.NewProcessedFetcher(p).Fetch(testImageConfiguration())
	Assert(t, err != nil, "404 from the remote store should be an error")
	Equals(t, int32(1), atomic.LoadInt32(&requests))
}

func testImageConfiguration() *core.ImageConfiguration {
	return &core.ImageConfiguration{ID: "0123456789abcdef0123456789abcdef", Namespace: "ns", Filename: "w300.jpg", Width: 300, Format: "jpg"}
}
