package server_test

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/core/signature"
	fetcher "github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
)

const (
	readSecret = "read-secret"
	imageDir   = "/test_namespace/6e0/072/682/e66287b662827da75b244a3"
)

// storedImageHandler serves an image whose original, info.json and a
// rendered variant are already on disk, through the full middleware stack
func storedImageHandler(t *testing.T, sig *signature.Config) http.Handler {
	t.Helper()
	base := t.TempDir()
	sc := &core.ServerConfiguration{LocalBasePath: base, DefaultQuality: 90, SignatureConfig: sig}
	p := &paths.Paths{LocalBasePath: base}
	sc.Adapters = &core.Adapters{Fetcher: &fetcher.Fetcher{}, Paths: p}

	original := p.LocalOriginalPath("test_namespace", cropHash)
	Ok(t, os.MkdirAll(filepath.Dir(original), 0700))
	WriteOrientedJPEG(t, original, 200, 100, 6, binary.BigEndian)
	Ok(t, os.WriteFile(filepath.Join(filepath.Dir(original), "info.json"), []byte(`{"width":100}`), 0644))
	Ok(t, os.WriteFile(filepath.Join(filepath.Dir(original), "x200.jpg"), []byte("rendered"), 0644))

	return server.NewHandler(sc, base)
}

func requestStatus(h http.Handler, method, uri string) int {
	request, _ := http.NewRequest(method, uri, nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	return response.Code
}

func signedRead(method, path string) string {
	return signature.NewSigner(readSecret, "").SignURL(method, path, 5*time.Minute)
}

func readSignatures() *signature.Config {
	return &signature.Config{Enabled: true, RequireForReads: true, Secrets: []string{readSecret}, MaxTTL: time.Hour}
}

// Files already on disk used to be served before the signature check, so
// once rendered a URL worked forever without its signature
func TestStoredFilesRequireReadSignature(t *testing.T) {
	h := storedImageHandler(t, readSignatures())

	for _, name := range []string{"original", "info.json", "x200.jpg"} {
		path := imageDir + "/" + name
		for _, method := range []string{"GET", "HEAD"} {
			Equals(t, http.StatusUnauthorized, requestStatus(h, method, path))
			Equals(t, http.StatusOK, requestStatus(h, method, signedRead(method, path)))
		}
	}

	// A signature for one file does not cover another one
	tampered := strings.Replace(signedRead("GET", imageDir+"/w100.jpg"), "/w100.jpg?", "/x200.jpg?", 1)
	Equals(t, http.StatusUnauthorized, requestStatus(h, "GET", tampered))
	tampered = strings.Replace(signedRead("GET", imageDir+"/w100.jpg"), "/w100.jpg?", "/original?", 1)
	Equals(t, http.StatusUnauthorized, requestStatus(h, "GET", tampered))
}

// Without signatures for reads, stored files are served as before, HEAD
// included
func TestStoredFilesServedWithoutReadSignature(t *testing.T) {
	uploadsOnly := &signature.Config{Enabled: true, Secrets: []string{readSecret}, MaxTTL: time.Hour}
	for _, sig := range []*signature.Config{nil, uploadsOnly} {
		h := storedImageHandler(t, sig)
		for _, name := range []string{"original", "info.json", "x200.jpg"} {
			for _, method := range []string{"GET", "HEAD"} {
				Equals(t, http.StatusOK, requestStatus(h, method, imageDir+"/"+name))
			}
		}
		// Variants not on disk are still rendered by the router
		Equals(t, http.StatusOK, requestStatus(h, "GET", imageDir+"/w50.jpg"))
	}
}

// The admin port has no signature check, so it must not serve stored files
func TestAdminServerDoesNotServeStoredFiles(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	Ok(t, err)
	Ok(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(wd) })

	// The admin server used to serve ./public, the default storage path
	Ok(t, os.MkdirAll(filepath.Join(dir, "public", "test_namespace"), 0700))
	Ok(t, os.WriteFile(filepath.Join(dir, "public", "test_namespace", "original"), []byte("x"), 0644))

	h := server.NewAdminServerHandler()
	Equals(t, http.StatusNotFound, requestStatus(h, "GET", "/test_namespace/original"))
	Equals(t, http.StatusOK, requestStatus(h, "GET", "/probe/live"))
}
