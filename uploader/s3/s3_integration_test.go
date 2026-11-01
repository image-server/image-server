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

func TestStoreListCopyDelete(t *testing.T) {
	store := useStore(t, true)
	u := &Uploader{}
	deleteBatch = 2
	t.Cleanup(func() { deleteBatch = 1000 })

	for _, k := range []string{"ns/a/original", "ns/a/w1.jpg", "ns/b/original", "ns/b/c0.100_0.100_0.500_0.500-w10.jpg", "nsx/a/original"} {
		Ok(t, u.Upload(writeFile(t, "data "+k), k, "image/png"))
	}

	keys, err := u.ListKeys("ns/")
	Ok(t, err)
	Equals(t, []string{"ns/a/original", "ns/a/w1.jpg", "ns/b/c0.100_0.100_0.500_0.500-w10.jpg", "ns/b/original"}, keys)

	Ok(t, u.Copy("ns/b/c0.100_0.100_0.500_0.500-w10.jpg", "moved/b/c0.100_0.100_0.500_0.500-w10.jpg"))
	Equals(t, "image/png", aws.ToString(store.Head(t, "moved/b/c0.100_0.100_0.500_0.500-w10.jpg").ContentType))

	// Five keys in batches of two, one of them missing
	Ok(t, u.DeleteKeys(append(keys, "ns/missing")))
	left, err := u.ListKeys("ns")
	Ok(t, err)
	Equals(t, []string{"nsx/a/original"}, left)

	Ok(t, u.PutBytes("ns/.marker", []byte(`{"from":"x"}`), "application/json"))
	data, found, err := u.GetBytes("ns/.marker")
	Ok(t, err)
	Equals(t, true, found)
	Equals(t, `{"from":"x"}`, string(data))
	_, found, err = u.GetBytes("ns/none")
	Ok(t, err)
	Equals(t, false, found)
}
