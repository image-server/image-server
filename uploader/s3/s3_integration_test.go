package s3

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"

	. "github.com/image-server/image-server/test"
)

// useStore points the package client at a real S3-compatible store, set up
// the way the server does it
func useStore(t *testing.T, skip bool) *S3Store {
	store := NewS3Store(t)
	Initialize(store.Bucket, store.Region, Options{UsePathStyle: true, SkipACL: skip})
	t.Cleanup(func() { skipACL = false })
	return store
}

func TestStoreUploadListDownload(t *testing.T) {
	for _, skip := range []bool{true, false} {
		store := useStore(t, skip)
		u := &Uploader{}

		Ok(t, u.Upload(writeFile(t, "original bytes"), "base/ns/abc/def/original", "image/heic"))
		Ok(t, u.Upload(writeFile(t, "variant bytes"), "base/ns/abc/def/w300.jpg", "image/jpeg"))

		names, err := u.ListDirectory("base/ns/abc/def")
		Ok(t, err)
		Equals(t, []string{"original", "w300.jpg"}, names)
		Equals(t, "image/heic", aws.ToString(store.Head(t, "base/ns/abc/def/original").ContentType))

		dest := filepath.Join(t.TempDir(), "original")
		Ok(t, u.Download("base/ns/abc/def/original", dest))
		data, err := os.ReadFile(dest)
		Ok(t, err)
		Equals(t, "original bytes", string(data))

		err = u.Download("base/ns/abc/def/w999.jpg", filepath.Join(t.TempDir(), "w999.jpg"))
		Assert(t, errors.Is(err, ErrNotFound), "got %v, want ErrNotFound", err)
	}
}
