package paths_test

import (
	"testing"

	"github.com/image-server/image-server/paths"
	. "github.com/image-server/image-server/test"
)

func TestRemoteURLsEmptyWithoutRemoteStore(t *testing.T) {
	p := &paths.Paths{LocalBasePath: "public", RemoteBasePath: "p"}
	Equals(t, "", p.RemoteImageURL("ns", "0123456789abcdef0123456789abcdef", "w300.jpg"))
	Equals(t, "", p.RemoteOriginalURL("ns", "0123456789abcdef0123456789abcdef"))
}

func TestRemoteURLsWithRemoteStore(t *testing.T) {
	p := &paths.Paths{RemoteBasePath: "p", RemoteBaseURL: "https://s3-us-east-1.amazonaws.com/bucket"}
	Equals(t, "https://s3-us-east-1.amazonaws.com/bucket/p/ns/012/345/678/9abcdef0123456789abcdef/w300.jpg",
		p.RemoteImageURL("ns", "0123456789abcdef0123456789abcdef", "w300.jpg"))
	Equals(t, "https://s3-us-east-1.amazonaws.com/bucket/p/ns/012/345/678/9abcdef0123456789abcdef/original",
		p.RemoteOriginalURL("ns", "0123456789abcdef0123456789abcdef"))
}
