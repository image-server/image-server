package namespaces

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

const img1 = "6e0/072/682/e66287b662827da75b244a3"
const img2 = "31e/8b3/187/a9f63f26d58c88bf09a7bbd"

func remoteStorage(t *testing.T, keys ...string) (Storage, *memRemote) {
	t.Helper()
	m := newMemRemote(keys...)
	return Storage{Root: t.TempDir(), Remote: m, RemoteBase: "images"}, m
}

func expectKeys(t *testing.T, m *memRemote, want ...string) {
	t.Helper()
	got := append([]string{}, m.keys()...)
	want = append([]string{}, want...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys:\n got %v\nwant %v", got, want)
	}
}

func TestRemoteRenameMovesEveryObject(t *testing.T) {
	s, m := remoteStorage(t,
		"images/a/"+img1+"/original", "images/a/"+img1+"/w100.jpg", "images/a/"+img2+"/original",
		"images/ab/"+img1+"/original") // a longer name with the same prefix
	// A local cache of the source
	image(t, s.Root, "a")

	status, err := s.Rename("a", "b")
	if err != nil || status != "renamed" {
		t.Fatalf("got %q, %v", status, err)
	}
	expectKeys(t, m, "images/ab/"+img1+"/original",
		"images/b/"+img1+"/original", "images/b/"+img1+"/w100.jpg", "images/b/"+img2+"/original")
	if exists(filepath.Join(s.Root, "a")) {
		t.Fatal("local cache of the source not removed")
	}
	if m.objects["images/b/"+img1+"/original"] != "data:images/a/"+img1+"/original" {
		t.Fatal("content not copied")
	}

	// A retry after success: nothing to move
	status, err = s.Rename("a", "b")
	if err != nil || status != "missing" {
		t.Fatalf("retry got %q, %v", status, err)
	}
}

func TestRemoteRenameMissingSource(t *testing.T) {
	s, m := remoteStorage(t, "images/b/"+img1+"/original")
	status, err := s.Rename("a", "b")
	if err != nil || status != "missing" {
		t.Fatalf("got %q, %v", status, err)
	}
	expectKeys(t, m, "images/b/"+img1+"/original")
}

// Both names hold images: never overwrite the target
func TestRemoteRenameConflict(t *testing.T) {
	keys := []string{"images/a/" + img1 + "/original", "images/b/" + img2 + "/original"}
	s, m := remoteStorage(t, keys...)
	if _, err := s.Rename("a", "b"); !errors.Is(err, ErrExists) {
		t.Fatalf("got %v, want ErrExists", err)
	}
	expectKeys(t, m, keys...)

	// Only a local copy of the target is enough to refuse
	s, m = remoteStorage(t, "images/a/"+img1+"/original")
	image(t, s.Root, "b")
	if _, err := s.Rename("a", "b"); !errors.Is(err, ErrExists) {
		t.Fatalf("local target: got %v, want ErrExists", err)
	}
	expectKeys(t, m, "images/a/"+img1+"/original")
}

// A rename that stopped part way is finished by retrying the same rename
func TestRemoteRenameResumesAfterPartialCopy(t *testing.T) {
	s, m := remoteStorage(t,
		"images/a/"+img1+"/original", "images/a/"+img1+"/w100.jpg", "images/a/"+img2+"/original")
	m.failCopyAfter = 1

	if _, err := s.Rename("a", "b"); !errors.Is(err, errStore) {
		t.Fatalf("got %v, want the store error", err)
	}
	// Source and target both hold objects, and the target carries the marker
	if _, ok := m.objects["images/b/"+renameMarker]; !ok {
		t.Fatal("no marker on the partial target")
	}

	m.failCopyAfter = 0
	copiesBefore := m.copies
	status, err := s.Rename("a", "b")
	if err != nil || status != "renamed" {
		t.Fatalf("retry got %q, %v; want renamed, not a conflict", status, err)
	}
	if n := m.copies - copiesBefore; n != 2 {
		t.Fatalf("retry copied %d objects, want only the 2 not yet copied", n)
	}
	expectKeys(t, m, "images/b/"+img1+"/original", "images/b/"+img1+"/w100.jpg", "images/b/"+img2+"/original")
}

// Everything moved, but the marker was not removed: a retry removes it
func TestRemoteRenameFinishesAfterFailedCleanup(t *testing.T) {
	s, m := remoteStorage(t, "images/b/"+img1+"/original", "images/b/"+renameMarker)
	m.objects["images/b/"+renameMarker] = `{"from":"a"}`

	status, err := s.Rename("a", "b")
	if err != nil || status != "renamed" {
		t.Fatalf("got %q, %v", status, err)
	}
	expectKeys(t, m, "images/b/"+img1+"/original")
}

// Another rename's marker does not let this one write into the target
func TestRemoteRenameDoesNotResumeAnotherRename(t *testing.T) {
	s, m := remoteStorage(t, "images/a/"+img1+"/original", "images/b/"+renameMarker)
	m.objects["images/b/"+renameMarker] = `{"from":"c"}`

	if _, err := s.Rename("a", "b"); !errors.Is(err, ErrExists) {
		t.Fatalf("got %v, want ErrExists", err)
	}
}

// Renaming back and forth works, as for hiding and restoring a namespace
func TestRemoteRenameBackAndForth(t *testing.T) {
	s, m := remoteStorage(t, "images/a/"+img1+"/original")
	for _, step := range [][2]string{{"a", "hidden_1"}, {"hidden_1", "a"}, {"a", "hidden_2"}} {
		if status, err := s.Rename(step[0], step[1]); err != nil || status != "renamed" {
			t.Fatalf("%v: got %q, %v", step, status, err)
		}
	}
	expectKeys(t, m, "images/hidden_2/"+img1+"/original")
}

// A background upload into the source, started by a request that held the
// namespace, finishes before the rename lists the source
func TestRemoteRenameWaitsForBackgroundUploads(t *testing.T) {
	s, m := remoteStorage(t, "images/a/"+img1+"/original")

	unlock := ReadLock("a")
	uploadDone := TrackUpload("a")
	unlock()

	done := make(chan error, 1)
	go func() {
		_, err := s.Rename("a", "b")
		done <- err
	}()
	select {
	case <-done:
		t.Fatal("rename did not wait for the background upload")
	case <-time.After(100 * time.Millisecond):
	}
	m.PutBytes("images/a/"+img1+"/w300.jpg", []byte("x"), "image/jpeg")
	uploadDone()

	if err := <-done; err != nil {
		t.Fatal(err)
	}
	expectKeys(t, m, "images/b/"+img1+"/original", "images/b/"+img1+"/w300.jpg")
}

func TestRemoteDelete(t *testing.T) {
	s, m := remoteStorage(t,
		"images/a/"+img1+"/original", "images/a/"+img1+"/w100.jpg", "images/a/"+img2+"/original",
		"images/ab/"+img1+"/original")
	image(t, s.Root, "a")

	result, err := s.Delete("a")
	if err != nil {
		t.Fatal(err)
	}
	if result != (Result{Status: "deleted", Images: 2, Files: 3}) {
		t.Fatalf("got %+v", result)
	}
	expectKeys(t, m, "images/ab/"+img1+"/original")
	if exists(filepath.Join(s.Root, "a")) {
		t.Fatal("local cache not removed")
	}

	result, err = s.Delete("a")
	if err != nil || result.Status != "missing" {
		t.Fatalf("retry got %+v, %v", result, err)
	}
}

func TestRemoteDeleteRetryAfterFailure(t *testing.T) {
	s, m := remoteStorage(t, "images/a/"+img1+"/original")
	image(t, s.Root, "a")
	m.failDelete = true
	if _, err := s.Delete("a"); !errors.Is(err, errStore) {
		t.Fatalf("got %v, want the store error", err)
	}
	if !exists(filepath.Join(s.Root, "a")) {
		t.Fatal("local cache removed although the bucket still holds the namespace")
	}

	m.failDelete = false
	result, err := s.Delete("a")
	if err != nil || result.Status != "deleted" {
		t.Fatalf("retry got %+v, %v", result, err)
	}
	expectKeys(t, m)
}

// Only a local copy left (e.g. a failed earlier delete): it is removed
func TestRemoteDeleteLocalOnly(t *testing.T) {
	s, _ := remoteStorage(t)
	image(t, s.Root, "a")
	result, err := s.Delete("a")
	if err != nil || result.Status != "deleted" {
		t.Fatalf("got %+v, %v", result, err)
	}
	if _, err := os.Lstat(filepath.Join(s.Root, "a")); !os.IsNotExist(err) {
		t.Fatal("local copy not removed")
	}
}
