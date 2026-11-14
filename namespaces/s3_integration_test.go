package namespaces

import (
	"errors"
	"sync"
	"testing"

	. "github.com/image-server/image-server/test"
	"github.com/image-server/image-server/uploader/s3"
)

// failingCopies fails copies after the first n, like a store or network
// that gives out part way through a rename
type failingCopies struct {
	*s3.Uploader
	mu sync.Mutex
	n  int
}

func (f *failingCopies) Copy(src, dst string) error {
	f.mu.Lock()
	if f.n == 0 {
		f.mu.Unlock()
		return errStore
	}
	f.n--
	f.mu.Unlock()
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
	objects := func(ns string) int {
		n := 0
		for _, k := range store.Keys(t, "images/"+ns+"/") {
			if k != "images/"+ns+"/"+renameMarker {
				n++
			}
		}
		return n
	}

	// The first attempt stops after two copies; a retry finishes it
	if _, err := s.Rename("acct", "hidden_1"); !errors.Is(err, errStore) {
		t.Fatalf("got %v, want the copy error", err)
	}
	s.Remote = u
	status, err := s.Rename("acct", "hidden_1")
	Ok(t, err)
	Equals(t, "renamed", status)
	Equals(t, 0, len(store.Keys(t, "images/acct/")))
	Equals(t, 4, objects("hidden_1"))
	Equals(t, 1, objects("acct2"))
	Assert(t, store.Head(t, "images/hidden_1/"+renameMarker) != nil, "no done marker")

	// A retry after completion answers renamed
	status, err = s.Rename("acct", "hidden_1")
	Ok(t, err)
	Equals(t, "renamed", status)

	// Both exist: refused, nothing changes
	_, err = s.Rename("acct2", "hidden_1")
	Assert(t, errors.Is(err, ErrExists), "got %v, want ErrExists", err)
	Equals(t, 4, objects("hidden_1"))

	// Restore and hide again with the same names: real renames each time
	for _, step := range [][2]string{{"hidden_1", "acct"}, {"acct", "hidden_1"}} {
		status, err = s.Rename(step[0], step[1])
		Ok(t, err)
		Equals(t, "renamed", status)
		Equals(t, 0, len(store.Keys(t, "images/"+step[0]+"/")))
		Equals(t, 4, objects(step[1]))
	}

	// A done marker whose target lost objects is not taken as done
	Ok(t, u.DeleteKeys([]string{"images/hidden_1/" + img2 + "/info.json"}))
	_, err = s.Rename("acct", "hidden_1")
	Assert(t, errors.Is(err, ErrExists), "got %v, want ErrExists for a smaller target", err)

	// Delete removes the objects and the marker
	result, err := s.Delete("hidden_1")
	Ok(t, err)
	Equals(t, Result{Status: "deleted", Images: 2, Files: 3}, result)
	Equals(t, 0, len(store.Keys(t, "images/hidden_1/")))
	result, err = s.Delete("hidden_1")
	Ok(t, err)
	Equals(t, "missing", result.Status)
	status, err = s.Rename("acct", "hidden_1")
	Ok(t, err)
	Equals(t, "missing", status)
}

func TestS3DeleteImage(t *testing.T) {
	store := NewS3Store(t)
	s3.Initialize(store.Bucket, store.Region, s3.Options{UsePathStyle: true, SkipACL: true})
	for _, k := range []string{"acct/" + img1 + "/original", "acct/" + img1 + "/w100.jpg", "acct/" + img1 + "/info.json", "acct/" + img2 + "/original", "acct2/" + img1 + "/original"} {
		store.Put(t, "images/"+k, []byte(k), "image/jpeg")
	}
	s := Storage{Root: t.TempDir(), Remote: &s3.Uploader{}, RemoteBase: "images"}

	result, err := s.DeleteImage("acct", "6e0072682e66287b662827da75b244a3")
	Ok(t, err)
	Equals(t, ImageResult{Status: "deleted", Files: 3}, result)
	Equals(t, []string{"images/acct/" + img2 + "/original"}, store.Keys(t, "images/acct/"))
	Equals(t, 1, len(store.Keys(t, "images/acct2/")))

	result, err = s.DeleteImage("acct", "6e0072682e66287b662827da75b244a3")
	Ok(t, err)
	Equals(t, "missing", result.Status)
}
