package server

import (
	"net/http"
	"regexp"
)

// imageFilePath matches stored files and variants: /{ns}/{h0-3}/{h3-6}/{h6-9}/{h9-32}/{file}
var imageFilePath = regexp.MustCompile(`^/([a-z0-9_-]+)/[a-f0-9]{3}/[a-f0-9]{3}/[a-f0-9]{3}/[a-f0-9]{23}/[^/]+$`)

// CacheHeaders sets Cache-Control and a cache tag header (the namespace) on
// successful image reads. Errors are never marked cacheable.
type CacheHeaders struct {
	CacheControl string
	TagHeader    string
}

func (c *CacheHeaders) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		next(w, r)
		return
	}
	m := imageFilePath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		next(w, r)
		return
	}
	next(&cacheHeaderWriter{ResponseWriter: w, headers: c, namespace: m[1]}, r)
}

// cacheHeaderWriter adds the headers when the status is known to be a success
type cacheHeaderWriter struct {
	http.ResponseWriter
	headers     *CacheHeaders
	namespace   string
	wroteHeader bool
}

func (w *cacheHeaderWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		switch status {
		case http.StatusOK, http.StatusPartialContent, http.StatusNotModified:
			if w.headers.CacheControl != "" {
				w.Header().Set("Cache-Control", w.headers.CacheControl)
			}
			if w.headers.TagHeader != "" {
				w.Header().Set(w.headers.TagHeader, w.namespace)
			}
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *cacheHeaderWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
