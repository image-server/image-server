package migrate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/fetcher"
	fetcherhttp "github.com/image-server/image-server/fetcher/http"
	"github.com/image-server/image-server/paths"
	"github.com/image-server/image-server/server"
	. "github.com/image-server/image-server/test"
	"github.com/image-server/image-server/uploader/s3"
)

// A migrated tree is served from the bucket alone
func TestS3MigrateThenServe(t *testing.T) {
	store := NewS3Store(t)
	s3.Initialize(store.Bucket, store.Region, s3.Options{UsePathStyle: true, SkipACL: true})

	root := t.TempDir()
	data, err := os.ReadFile("../test/images/a.jpg")
	Ok(t, err)
	write(t, root, "acct/"+img+"/original", string(data))
	write(t, root, "acct/"+img+"/info.json", `{"content_type":"image/jpeg","width":574,"height":496}`)
	write(t, root, "acct/"+img+"/w300.jpg", "jpg")
	write(t, root, "tmp/x", "temporary")

	summary, err := Run(Options{Root: root, RemoteBase: "images"}, &s3.Uploader{})
	Ok(t, err)
	Equals(t, Summary{Images: 1, Uploaded: 2, Skipped: 1}, summary)
	keys := store.Keys(t, "")
	sort.Strings(keys)
	Equals(t, []string{"images/acct/" + img + "/info.json", "images/acct/" + img + "/original"}, keys)
	Equals(t, "image/jpeg", aws.ToString(store.Head(t, "images/acct/"+img+"/original").ContentType))
	Equals(t, "application/json", aws.ToString(store.Head(t, "images/acct/"+img+"/info.json").ContentType))

	summary, err = Run(Options{Root: root, RemoteBase: "images"}, &s3.Uploader{})
	Ok(t, err)
	Equals(t, 0, summary.Uploaded)
	Equals(t, 2, summary.Existing)

	// A server with an empty local directory reads everything from the bucket
	base := t.TempDir()
	sc := &core.ServerConfiguration{LocalBasePath: base, RemoteBasePath: "images", DefaultQuality: 90, UploaderType: "s3", S3Reads: true}
	sc.Adapters = &core.Adapters{Fetcher: &fetcherhttp.Fetcher{}, Paths: &paths.Paths{LocalBasePath: base, RemoteBasePath: "images"}}
	fetcher.Store = &s3.Uploader{}
	t.Cleanup(func() { fetcher.Store = nil })
	h := server.NewHandler(sc, base)
	for _, name := range []string{"original", "info.json", "w100.jpg"} {
		request, _ := http.NewRequest("GET", "/acct/"+img+"/"+name, nil)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		Equals(t, http.StatusOK, response.Code)
	}
	_, err = os.Stat(filepath.Join(base, "acct", filepath.FromSlash(img), "w100.jpg"))
	Ok(t, err)
}
