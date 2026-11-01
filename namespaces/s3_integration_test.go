package namespaces

import (
	"errors"
	"testing"

	. "github.com/image-server/image-server/test"
	"github.com/image-server/image-server/uploader/s3"
)

// failingCopies fails copies after the first n, like a store or network
// that gives out part way through a rename
type failingCopies struct {
	*s3.Uploader
	n int
}

func (f *failingCopies) Copy(src, dst string) error {
	if f.n == 0 {
		return errStore
	}
	f.n--
	return f.Uploader.Copy(src, dst)
}

func TestS3RenameAndDelete(t *testing.T) {
	store := NewS3Store(t)
	s3.Initialize(store.Bucket, store.Region, s3.Options{UsePathStyle: true, SkipACL: true})
	u := &s3.Uploader{}
	for _, k := range []string{img1 + "/original", img1 + "/w100.jpg", img2 + "/original", img2 + "/info.json"} {
		store.Put(t, "images/acct/"+k, []byte(k), "image/jpeg")
	}
	store.Put(t, "images/acct2/"+img1+"/original", []byte("x"), "image/jpeg")
	s := Storage{Root: t.TempDir(), Remote: &failingCopies{Uploader: u, n: 2}, RemoteBase: "images"}

	// The first attempt stops after two copies; a retry finishes it
	if _, err := s.Rename("acct", "hidden_1"); !errors.Is(err, errStore) {
		t.Fatalf("got %v, want the copy error", err)
	}
	s.Remote = u
	status, err := s.Rename("acct", "hidden_1")
	Ok(t, err)
	Equals(t, "renamed", status)
	Equals(t, 0, len(store.Keys(t, "images/acct/")))
	Equals(t, 4, len(store.Keys(t, "images/hidden_1/")))
	Equals(t, 1, len(store.Keys(t, "images/acct2/")))

	// Both exist: refused, nothing changes
	_, err = s.Rename("acct2", "hidden_1")
	Assert(t, errors.Is(err, ErrExists), "got %v, want ErrExists", err)

	// Restore, then delete
	status, err = s.Rename("hidden_1", "acct")
	Ok(t, err)
	Equals(t, "renamed", status)
	result, err := s.Delete("acct")
	Ok(t, err)
	Equals(t, Result{Status: "deleted", Images: 2, Files: 4}, result)
	Equals(t, 0, len(store.Keys(t, "images/acct/")))
	result, err = s.Delete("acct")
	Ok(t, err)
	Equals(t, "missing", result.Status)
}
