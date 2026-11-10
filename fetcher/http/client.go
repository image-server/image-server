package http

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// A Client extends http.Client and patches incorrect URL path escaping
// for more information on bug: https://github.com/golang/go/issues/5684
type Client struct {
	http.Client
}

var slashReg *regexp.Regexp

func init() {
	slashReg = regexp.MustCompile("^/+")
}

// Get issues a GET to the specified URL.  If the response is one of the
// following redirect codes, Get follows the redirect after calling the
// Client's CheckRedirect function.
//
//	301 (Moved Permanently)
//	302 (Found)
//	303 (See Other)
//	307 (Temporary Redirect)
//
// An error is returned if the Client's CheckRedirect function fails
// or if there was an HTTP protocol error. A non-2xx response doesn't
// cause an error.
//
// When err is nil, resp always contains a non-nil resp.Body.
// Caller should close resp.Body when done reading from it.
func (c *Client) Get(urlStr string) (resp *http.Response, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	p := strings.Replace(u.Path, " ", "%20", -1)
	p = slashReg.ReplaceAllString(p, "/")
	u.Opaque = p

	header := make(http.Header)
	header.Add("user-agent", "image-server")
	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Host:       u.Host,
	}

	return c.Do(req)
}
