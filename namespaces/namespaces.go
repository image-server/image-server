// Package namespaces renames and deletes whole namespaces, in local storage
// and in remote storage when it is configured.
package namespaces

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	vipsadapter "github.com/image-server/image-server/processor/vips"
)

var (
	// ErrInvalidName is returned for names that are not a plain namespace
	ErrInvalidName = errors.New("invalid namespace")
	// ErrExists is returned when the rename target already exists
	ErrExists = errors.New("namespace already exists")
)

// Result describes what a delete removed
type Result struct {
	Status string // "deleted" or "missing"
	Images int
	Files  int
}

// Rename moves a namespace in local storage. See Storage.Rename.
func Rename(root, from, to string) (string, error) {
	return Storage{Root: root}.Rename(from, to)
}

// Delete removes a namespace from local storage. See Storage.Delete.
func Delete(root, namespace string) (Result, error) {
	return Storage{Root: root}.Delete(namespace)
}

// Rename moves namespace from to to. It returns "renamed", or "missing" when
// from does not exist, so a retry after a rename that succeeded is safe. An
// existing target is ErrExists. With remote storage, an interrupted rename
// is finished by retrying it.
func (s Storage) Rename(from, to string) (string, error) {
	root := s.Root
	fromDir, err := dir(root, from)
	if err != nil {
		return "", err
	}
	toDir, err := dir(root, to)
	if err != nil {
		return "", err
	}
	if from == to {
		return "", fmt.Errorf("%w: the new name is the same as the old one", ErrInvalidName)
	}

	// Always lock in name order, so two renames cannot deadlock
	first, second := from, to
	if second < first {
		first, second = second, first
	}
	unlockFirst := lock(first, true)
	defer unlockFirst()
	unlockSecond := lock(second, true)
	defer unlockSecond()

	if s.Remote != nil {
		return s.renameRemote(from, to, fromDir, toDir)
	}

	fromHasFiles, err := hasFiles(fromDir)
	if err != nil {
		return "", err
	}
	if !fromHasFiles {
		return "missing", nil
	}
	toHasFiles, err := hasFiles(toDir)
	if err != nil {
		return "", err
	}
	if toHasFiles {
		return "", fmt.Errorf("%w: %s", ErrExists, to)
	}
	// Directories with no files (e.g. left by a request for an image that
	// did not exist) hold nothing; rename(2) cannot replace them
	if err := os.RemoveAll(toDir); err != nil {
		return "", err
	}

	if err := os.Rename(fromDir, toDir); err != nil {
		return "", err
	}
	vipsadapter.Forget(fromDir)
	vipsadapter.Forget(toDir)
	return "renamed", nil
}

// Delete removes a namespace with every image in it. Deleting a namespace
// that does not exist returns status "missing".
func (s Storage) Delete(namespace string) (Result, error) {
	nsDir, err := dir(s.Root, namespace)
	if err != nil {
		return Result{}, err
	}
	unlock := lock(namespace, true)
	defer unlock()

	if s.Remote != nil {
		return s.deleteRemote(namespace, nsDir)
	}

	info, err := os.Lstat(nsDir)
	if os.IsNotExist(err) {
		return Result{Status: "missing"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("%s is not a directory", namespace)
	}
	if found, err := hasFiles(nsDir); err != nil {
		return Result{}, err
	} else if !found {
		// Only empty directories: nothing to delete, but clear them
		if err := os.RemoveAll(nsDir); err != nil {
			return Result{}, err
		}
		return Result{Status: "missing"}, nil
	}

	images, files, err := count(nsDir)
	if err != nil {
		return Result{}, err
	}
	if err := os.RemoveAll(nsDir); err != nil {
		return Result{}, err
	}
	vipsadapter.Forget(nsDir)
	return Result{Status: "deleted", Images: images, Files: files}, nil
}

// dir returns the namespace directory, refusing anything but a plain name
// directly under root, and the directories the server keeps there itself
func dir(root, namespace string) (string, error) {
	if namespace == "" || strings.Trim(namespace, "abcdefghijklmnopqrstuvwxyz0123456789_-") != "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, namespace)
	}
	if namespace == "tmp" {
		return "", fmt.Errorf("%w: %s is reserved", ErrInvalidName, namespace)
	}
	d := filepath.Join(root, namespace)
	// Defence in depth: the name is validated, but never leave the root
	if rel, err := filepath.Rel(root, d); err != nil || rel != namespace {
		return "", fmt.Errorf("%w: %q is outside the storage root", ErrInvalidName, namespace)
	}
	return d, nil
}

// hasFiles is true when dir holds at least one file. A request for an image
// that does not exist can leave empty directories behind; they are not a
// namespace.
func hasFiles(dir string) (bool, error) {
	found := false
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if os.IsNotExist(err) {
		return false, nil
	}
	return found, err
}

// count returns the images (directories four levels down, one per hash) and
// files in a namespace directory
func count(nsDir string) (images, files int, err error) {
	err = filepath.Walk(nsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files++
			return nil
		}
		rel, _ := filepath.Rel(nsDir, path)
		if rel != "." && strings.Count(rel, string(filepath.Separator)) == 3 {
			images++
		}
		return nil
	})
	return images, files, err
}

// CompileDeletePattern compiles a --delete-namespace-pattern. It must match
// the whole name: a partial match must not allow deleting a namespace.
func CompileDeletePattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("^(?:" + pattern + ")$")
}
