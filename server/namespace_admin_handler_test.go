package server_test

import (
	"bytes"
	"encoding/binary"
	"log"
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
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/namespaces"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
)

const adminSecret = "admin-secret"

// adminServer stores one image in test_namespace and enables namespace
// admin, deleting namespaces that match pattern
func adminServer(t *testing.T, pattern string) (http.Handler, *core.ServerConfiguration) {
	t.Helper()
	base := t.TempDir()
	sc := &core.ServerConfiguration{LocalBasePath: base, DefaultQuality: 90}
	p := &paths.Paths{LocalBasePath: base}
	sc.Adapters = &core.Adapters{Fetcher: &fetcher.Fetcher{}, Paths: p}
	sc.NamespaceAdmin = &core.NamespaceAdminConfiguration{Secrets: []string{adminSecret}, MaxTTL: time.Hour}
	if pattern != "" {
		compiled, err := namespaces.CompileDeletePattern(pattern)
		Ok(t, err)
		sc.NamespaceAdmin.DeletePattern = compiled
	}

	original := p.LocalOriginalPath("test_namespace", cropHash)
	Ok(t, os.MkdirAll(filepath.Dir(original), 0700))
	WriteOrientedJPEG(t, original, 200, 100, 6, binary.BigEndian)
	return server.NewRouter(sc), sc
}

// signed returns path signed for method with secret
func signed(method, path, secret string) string {
	return signature.NewSigner(secret, "").SignURL(method, path, 5*time.Minute)
}

func serve(router http.Handler, method, uri string) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(method, uri, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func nsExists(sc *core.ServerConfiguration, namespace string) bool {
	_, err := os.Stat(filepath.Join(sc.LocalBasePath, namespace))
	return err == nil
}

func TestRenameNamespace(t *testing.T) {
	router, sc := adminServer(t, "")

	path := "/test_namespace/rename/archive_ab12"
	r := serve(router, "POST", signed("POST", path, adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "renamed"`, r.Body.String())
	Matches(t, `"to": "archive_ab12"`, r.Body.String())
	Equals(t, false, nsExists(sc, "test_namespace"))

	// Variants are served under the new name only
	Equals(t, http.StatusOK, serve(router, "GET", "/archive_ab12/6e0/072/682/e66287b662827da75b244a3/w50.jpg").Code)
	Equals(t, http.StatusNotFound, getCrop(router, "w50.jpg").Code)
	Equals(t, false, nsExists(sc, "test_namespace"))

	// A retry is safe
	r = serve(router, "POST", signed("POST", path, adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "missing"`, r.Body.String())
}

func TestRenameNamespaceCollision(t *testing.T) {
	router, sc := adminServer(t, "")
	Ok(t, os.Mkdir(filepath.Join(sc.LocalBasePath, "taken"), 0700))

	r := serve(router, "POST", signed("POST", "/test_namespace/rename/taken", adminSecret))
	Equals(t, http.StatusConflict, r.Code)
	Equals(t, true, nsExists(sc, "test_namespace"))
}

func TestRenameNamespaceInvalidTarget(t *testing.T) {
	router, sc := adminServer(t, "")
	for _, to := range []string{"test_namespace", "tmp"} {
		path := "/test_namespace/rename/" + to
		Equals(t, http.StatusBadRequest, serve(router, "POST", signed("POST", path, adminSecret)).Code)
	}
	Equals(t, true, nsExists(sc, "test_namespace"))
}

func TestNamespaceAdminRejectsTraversal(t *testing.T) {
	router, sc := adminServer(t, ".*")
	for _, path := range []string{
		"/test_namespace/rename/..",
		"/test_namespace/rename/..%2Fescaped",
		"/test_namespace/rename/%2E%2E",
		"/..",
		"/%2E%2E",
		"/test_namespace/../test_namespace",
	} {
		for _, method := range []string{"POST", "DELETE"} {
			r := serve(router, method, signed(method, path, adminSecret))
			if r.Code == http.StatusOK {
				t.Errorf("%s %s: got 200", method, path)
			}
		}
	}
	Equals(t, true, nsExists(sc, "test_namespace"))
	entries, _ := os.ReadDir(sc.LocalBasePath)
	Equals(t, 1, len(entries))
}

func TestNamespaceAdminRequiresAdminSignature(t *testing.T) {
	router, sc := adminServer(t, ".*")
	rename := "/test_namespace/rename/other"

	// Unsigned, or signed with another secret
	Equals(t, http.StatusUnauthorized, serve(router, "POST", rename).Code)
	Equals(t, http.StatusUnauthorized, serve(router, "POST", signed("POST", rename, "upload-secret")).Code)
	Equals(t, http.StatusUnauthorized, serve(router, "DELETE", "/test_namespace").Code)

	// A namespace signature does not cover its rename: paths must match exactly
	uri := signature.NewSigner(adminSecret, "").SignURL("POST", "/test_namespace", 5*time.Minute)
	uri = strings.Replace(uri, "/test_namespace?", rename+"?", 1)
	Equals(t, http.StatusUnauthorized, serve(router, "POST", uri).Code)

	// A delete signature cannot be used for a rename
	uri = signed("DELETE", rename, adminSecret)
	Equals(t, http.StatusUnauthorized, serve(router, "POST", uri).Code)

	Equals(t, true, nsExists(sc, "test_namespace"))
}

func TestNamespaceAdminDisabled(t *testing.T) {
	router, sc := adminServer(t, ".*")
	sc.NamespaceAdmin = nil
	Equals(t, http.StatusNotFound, serve(router, "POST", signed("POST", "/test_namespace/rename/x", adminSecret)).Code)
	Equals(t, http.StatusNotFound, serve(router, "DELETE", signed("DELETE", "/test_namespace", adminSecret)).Code)
	Equals(t, true, nsExists(sc, "test_namespace"))
}

func TestNamespaceAdminRefusesRemoteStorage(t *testing.T) {
	router, sc := adminServer(t, ".*")
	sc.UploaderType = "s3"
	Equals(t, http.StatusNotImplemented, serve(router, "POST", signed("POST", "/test_namespace/rename/x", adminSecret)).Code)
	Equals(t, http.StatusNotImplemented, serve(router, "DELETE", signed("DELETE", "/test_namespace", adminSecret)).Code)
	Equals(t, true, nsExists(sc, "test_namespace"))
}

func TestDeleteNamespace(t *testing.T) {
	router, sc := adminServer(t, `archive_[a-z0-9]+`)
	Equals(t, http.StatusOK, serve(router, "POST", signed("POST", "/test_namespace/rename/archive_ab12", adminSecret)).Code)

	r := serve(router, "DELETE", signed("DELETE", "/archive_ab12", adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "deleted"`, r.Body.String())
	Matches(t, `"images_deleted": 1`, r.Body.String())
	Matches(t, `"files_deleted": 1`, r.Body.String())
	Equals(t, false, nsExists(sc, "archive_ab12"))

	r = serve(router, "DELETE", signed("DELETE", "/archive_ab12", adminSecret))
	Equals(t, http.StatusOK, r.Code)
	Matches(t, `"status": "missing"`, r.Body.String())
}

func TestDeleteNamespaceMustMatchWholePattern(t *testing.T) {
	router, sc := adminServer(t, "archive_.*")
	// "test_namespace" does not match; "old_archive_x" only matches in part
	Ok(t, os.Mkdir(filepath.Join(sc.LocalBasePath, "old_archive_x"), 0700))
	Equals(t, http.StatusForbidden, serve(router, "DELETE", signed("DELETE", "/test_namespace", adminSecret)).Code)
	Equals(t, http.StatusForbidden, serve(router, "DELETE", signed("DELETE", "/old_archive_x", adminSecret)).Code)
	Equals(t, true, nsExists(sc, "test_namespace"))
	Equals(t, true, nsExists(sc, "old_archive_x"))
}

func TestDeleteNamespaceWithoutPattern(t *testing.T) {
	router, sc := adminServer(t, "")
	Equals(t, http.StatusNotFound, serve(router, "DELETE", signed("DELETE", "/test_namespace", adminSecret)).Code)
	Equals(t, true, nsExists(sc, "test_namespace"))
}

// With upload signing on, the middleware must leave admin routes to their
// handlers: upload secrets are not accepted there, admin secrets are
func TestSignatureMiddlewareSkipsNamespaceAdmin(t *testing.T) {
	m := server.NewSignatureMiddleware(&signature.Config{Enabled: true, Secrets: []string{"upload-secret"}, MaxTTL: time.Hour})
	for _, c := range []struct {
		method, path string
		skipped      bool
	}{
		{"POST", "/ns/rename/other", true},
		{"DELETE", "/ns", true},
		{"POST", "/ns", false},
		{"POST", "/ns/6e0/072/682/e66287b662827da75b244a3/process", false},
	} {
		called := false
		request, _ := http.NewRequest(c.method, signed(c.method, c.path, adminSecret), nil)
		m.ServeHTTP(httptest.NewRecorder(), request, func(http.ResponseWriter, *http.Request) { called = true })
		if called != c.skipped {
			t.Errorf("%s %s: next called=%v, want %v", c.method, c.path, called, c.skipped)
		}
	}
}

// rejectionRecorder collects NamespaceAdmin metrics; the other logger calls
// these handlers make are ignored
type rejectionRecorder struct {
	core.Logger
	events chan string
}

func (r *rejectionRecorder) NamespaceAdmin(op, result, reason string) {
	r.events <- op + " " + result + " " + reason
}

func (r *rejectionRecorder) RequestLatency(string, time.Time) {}

func recordAdminMetrics(t *testing.T) chan string {
	events := make(chan string, 100)
	saved := logger.Loggers
	logger.Loggers = []core.Logger{&rejectionRecorder{events: events}}
	t.Cleanup(func() { logger.Loggers = saved })
	return events
}

func nextEvent(t *testing.T, events chan string) string {
	t.Helper()
	select {
	case e := <-events:
		return e
	case <-time.After(5 * time.Second):
		t.Fatal("no metric recorded")
		return ""
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestNamespaceAdminRejectionsAreCountedAndLogged(t *testing.T) {
	router, sc := adminServer(t, "archive_.*")
	sc.SignatureConfig = &signature.Config{Enabled: true, Secrets: []string{"upload-secret"}, MaxTTL: time.Hour}
	events := recordAdminMetrics(t)
	logs := captureLog(t)

	cases := []struct {
		method, uri, want string
	}{
		{"POST", "/test_namespace/rename/other", "rename rejected bad_signature"},
		{"POST", signed("POST", "/test_namespace/rename/other", "forged"), "rename rejected bad_signature"},
		{"POST", signed("POST", "/test_namespace/rename/other", "upload-secret"), "rename rejected wrong_secret"},
		{"POST", signed("POST", "/test_namespace/rename/tmp", adminSecret), "rename rejected invalid_name"},
		{"DELETE", signed("DELETE", "/test_namespace", adminSecret), "delete rejected pattern_mismatch"},
	}
	for _, c := range cases {
		request, _ := http.NewRequest(c.method, c.uri, nil)
		request.Header.Set("X-Request-Id", "req-42")
		request.RemoteAddr = "203.0.113.9:5555"
		router.ServeHTTP(httptest.NewRecorder(), request)
		Equals(t, c.want, nextEvent(t, events))
	}

	out := logs.String()
	Matches(t, `WARN namespace admin rejected op=delete reason=pattern_mismatch method=DELETE path="/test_namespace" namespace=test_namespace request_id="req-42" remote=203.0.113.9:5555`, out)
	Matches(t, `reason=wrong_secret`, out)
	if strings.Contains(out, "X-Signature") || strings.Contains(out, "X-Expires") {
		t.Fatalf("log contains the signature:\n%s", out)
	}

	// Disabled delete and disabled admin
	sc.NamespaceAdmin.DeletePattern = nil
	serve(router, "DELETE", signed("DELETE", "/test_namespace", adminSecret))
	Equals(t, "delete rejected disabled", nextEvent(t, events))
	sc.NamespaceAdmin = nil
	serve(router, "POST", signed("POST", "/test_namespace/rename/other", adminSecret))
	Equals(t, "rename rejected disabled", nextEvent(t, events))
}

func TestNamespaceAdminRejectionLogIsRateLimited(t *testing.T) {
	router, _ := adminServer(t, "")
	events := recordAdminMetrics(t)
	logs := captureLog(t)

	for i := 0; i < 50; i++ {
		serve(router, "POST", "/test_namespace/rename/other")
	}
	// Every rejection is counted
	for i := 0; i < 50; i++ {
		Equals(t, "rename rejected bad_signature", nextEvent(t, events))
	}
	if n := strings.Count(logs.String(), "WARN namespace admin rejected"); n > 20 {
		t.Fatalf("logged %d warnings in a minute, want at most 20", n)
	}
}
