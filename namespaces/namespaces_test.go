package namespaces

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// image writes an original and a variant for one image in namespace
func image(t *testing.T, root, namespace string) string {
	t.Helper()
	dir := filepath.Join(root, namespace, "6e0", "072", "682", "e66287b662827da75b244a3")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"original", "w100.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestRenameMovesNamespace(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")

	status, err := Rename(root, "a", "b")
	if err != nil || status != "renamed" {
		t.Fatalf("got %q, %v", status, err)
	}
	if exists(filepath.Join(root, "a")) || !exists(filepath.Join(root, "b", "6e0", "072", "682", "e66287b662827da75b244a3", "original")) {
		t.Fatal("namespace not moved")
	}

	// A retry finds the source gone
	status, err = Rename(root, "a", "b")
	if err != nil || status != "missing" {
		t.Fatalf("retry got %q, %v", status, err)
	}
}

func TestRenameRefusesExistingTarget(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")
	// Even an empty directory, which rename(2) would replace
	if err := os.Mkdir(filepath.Join(root, "b"), 0700); err != nil {
		t.Fatal(err)
	}

	if _, err := Rename(root, "a", "b"); !errors.Is(err, ErrExists) {
		t.Fatalf("got %v, want ErrExists", err)
	}
	if !exists(filepath.Join(root, "a")) {
		t.Fatal("source moved")
	}
}

func TestRejectsInvalidNames(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")
	for _, name := range []string{"", ".", "..", "../a", "a/b", "A", ".trash", "tmp", "a b", "a\x00"} {
		if _, err := Rename(root, "a", name); !errors.Is(err, ErrInvalidName) {
			t.Errorf("rename to %q: got %v, want ErrInvalidName", name, err)
		}
		if _, err := Delete(root, name); !errors.Is(err, ErrInvalidName) {
			t.Errorf("delete %q: got %v, want ErrInvalidName", name, err)
		}
	}
	if _, err := Rename(root, "a", "a"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("rename to itself: got %v, want ErrInvalidName", err)
	}
	if !exists(filepath.Join(root, "a")) {
		t.Fatal("namespace touched")
	}
}

func TestDeleteRemovesNamespace(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")
	image(t, root, "b")

	result, err := Delete(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	if result != (Result{Status: "deleted", Images: 1, Files: 2}) {
		t.Fatalf("got %+v", result)
	}
	if exists(filepath.Join(root, "a")) || !exists(filepath.Join(root, "b")) {
		t.Fatal("wrong namespace removed")
	}

	result, err = Delete(root, "a")
	if err != nil || result.Status != "missing" {
		t.Fatalf("retry got %+v, %v", result, err)
	}
}

// A rename waits for requests that hold the namespace, so a variant they
// write moves with it instead of recreating the old directory
func TestRenameWaitsForInFlightWork(t *testing.T) {
	root := t.TempDir()
	dir := image(t, root, "a")

	unlock := ReadLock("a")
	done := make(chan error, 1)
	go func() {
		_, err := Rename(root, "a", "b")
		done <- err
	}()

	select {
	case <-done:
		t.Fatal("rename did not wait for the request")
	case <-time.After(100 * time.Millisecond):
	}
	// The request finishes writing a variant, then releases the namespace
	if err := os.WriteFile(filepath.Join(dir, "w200.jpg"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	unlock()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("rename never finished")
	}
	if exists(filepath.Join(root, "a")) {
		t.Fatal("old namespace directory still exists")
	}
	if !exists(filepath.Join(root, "b", "6e0", "072", "682", "e66287b662827da75b244a3", "w200.jpg")) {
		t.Fatal("variant did not move with the namespace")
	}
}

// Opposite renames lock in name order, so neither deadlocks
func TestOppositeRenamesDoNotDeadlock(t *testing.T) {
	root := t.TempDir()
	image(t, root, "a")

	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			Rename(root, "a", "b")
			Rename(root, "b", "a")
		}
		close(done)
	}()
	for i := 0; i < 200; i++ {
		Rename(root, "b", "a")
		Rename(root, "a", "b")
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock")
	}
	if exists(filepath.Join(root, "a")) == exists(filepath.Join(root, "b")) {
		t.Fatal("want exactly one of a and b")
	}
}
