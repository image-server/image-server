package namespaces

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const hash1 = "6e0072682e66287b662827da75b244a3"

func TestDeleteImageLocal(t *testing.T) {
	root := t.TempDir()
	dir := image(t, root, "a")
	other := filepath.Join(root, "a", filepath.FromSlash(img2))
	if err := os.MkdirAll(other, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "original"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DeleteImage(root, "a", hash1)
	if err != nil || result != (ImageResult{Status: "deleted", Files: 2}) {
		t.Fatalf("got %+v, %v", result, err)
	}
	if exists(dir) || exists(filepath.Join(root, "a", "6e0")) {
		t.Fatal("image directory or its empty parents left behind")
	}
	if !exists(filepath.Join(other, "original")) {
		t.Fatal("another image was deleted")
	}

	// A retry is safe
	result, err = DeleteImage(root, "a", hash1)
	if err != nil || result != (ImageResult{Status: "missing"}) {
		t.Fatalf("retry: got %+v, %v", result, err)
	}
}

func TestDeleteImageInvalid(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")
	for _, c := range []struct {
		namespace, hash string
		want            error
	}{
		{"tmp", hash1, ErrInvalidName},
		{"..", hash1, ErrInvalidName},
		{"a", "6e0072682e66287b662827da75b244a", ErrInvalidHash},
		{"a", "6E0072682E66287B662827DA75B244A3", ErrInvalidHash},
		{"a", "../../../../../../../../../../..", ErrInvalidHash},
	} {
		if _, err := DeleteImage(root, c.namespace, c.hash); !errors.Is(err, c.want) {
			t.Errorf("%s %s: got %v, want %v", c.namespace, c.hash, err, c.want)
		}
	}
}

func TestDeleteImageRemote(t *testing.T) {
	s, m := remoteStorage(t,
		"images/a/"+img1+"/original",
		"images/a/"+img1+"/w100.jpg",
		"images/a/"+img1+"/info.json",
		"images/a/"+img2+"/original",
		"images/ab/"+img1+"/original",
	)
	dir := image(t, s.Root, "a") // local cache copy

	result, err := s.DeleteImage("a", hash1)
	if err != nil || result != (ImageResult{Status: "deleted", Files: 3}) {
		t.Fatalf("got %+v, %v", result, err)
	}
	expectKeys(t, m, "images/a/"+img2+"/original", "images/ab/"+img1+"/original")
	if exists(dir) {
		t.Fatal("local copy left behind")
	}
	if len(m.deleteBatches) != 1 || len(m.deleteBatches[0]) != 3 {
		t.Fatalf("delete batches: %v", m.deleteBatches)
	}

	result, err = s.DeleteImage("a", hash1)
	if err != nil || result != (ImageResult{Status: "missing"}) {
		t.Fatalf("retry: got %+v, %v", result, err)
	}
}

// A failed remote delete keeps the local copy, so a retry finds it again
func TestDeleteImageRemoteFailure(t *testing.T) {
	s, m := remoteStorage(t, "images/a/"+img1+"/original")
	dir := image(t, s.Root, "a")
	m.failDelete = true

	if _, err := s.DeleteImage("a", hash1); !errors.Is(err, errStore) {
		t.Fatalf("got %v", err)
	}
	if !exists(dir) {
		t.Fatal("local copy deleted after a failed remote delete")
	}

	m.failDelete = false
	result, err := s.DeleteImage("a", hash1)
	if err != nil || result.Status != "deleted" {
		t.Fatalf("retry: got %+v, %v", result, err)
	}
	expectKeys(t, m)
}

// The delete waits for requests that hold the namespace read lock
func TestDeleteImageWaitsForReadLock(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")
	unlock := ReadLock("a")

	done := make(chan struct{})
	go func() {
		DeleteImage(root, "a", hash1)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("delete did not wait for the read lock")
	case <-time.After(50 * time.Millisecond):
	}
	unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("delete did not finish")
	}
}
