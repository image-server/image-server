package namespaces

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	vipsadapter "github.com/image-server/image-server/processor/vips"
)

// ErrInvalidHash is returned for an image hash that is not 32 lowercase hex
// characters
var ErrInvalidHash = errors.New("invalid image hash")

// ImageResult describes what an image delete removed
type ImageResult struct {
	Status string // "deleted" or "missing"
	// Files is the number of files deleted: remote objects when remote
	// storage is set (local files are a cache of them), else local files
	Files int
}

// DeleteImage removes one image from local storage. See Storage.DeleteImage.
func DeleteImage(root, namespace, hash string) (ImageResult, error) {
	return Storage{Root: root}.DeleteImage(namespace, hash)
}

// DeleteImage removes one image, its original with every variant and info
// file, in local storage and in remote storage when it is set. It holds the
// namespace write lock, so it waits for requests and background uploads in
// the namespace, and serializes with renames and deletes. Deleting an image
// that does not exist returns status "missing"; a retry after a failure
// deletes what is left.
func (s Storage) DeleteImage(namespace, hash string) (ImageResult, error) {
	nsDir, err := dir(s.Root, namespace)
	if err != nil {
		return ImageResult{}, err
	}
	parts, err := hashParts(hash)
	if err != nil {
		return ImageResult{}, err
	}
	imgDir := filepath.Join(append([]string{nsDir}, parts...)...)

	unlock := lock(namespace, true)
	defer unlock()

	var keys []string
	if s.Remote != nil {
		// Ends in a slash, so it never matches a longer path
		prefix := path.Join(append([]string{s.RemoteBase, namespace}, parts...)...) + "/"
		keys, err = s.Remote.ListKeys(prefix)
		if err != nil {
			return ImageResult{}, err
		}
	}
	local, err := countFiles(imgDir)
	if err != nil {
		return ImageResult{}, err
	}

	if len(keys) > 0 {
		if err := s.Remote.DeleteKeys(keys); err != nil {
			return ImageResult{}, err
		}
	}
	if err := os.RemoveAll(imgDir); err != nil {
		return ImageResult{}, err
	}
	vipsadapter.Forget(imgDir)
	removeEmptyParents(filepath.Dir(imgDir), nsDir)

	files := len(keys)
	if files == 0 {
		files = local
	}
	if files == 0 {
		return ImageResult{Status: "missing"}, nil
	}
	return ImageResult{Status: "deleted", Files: files}, nil
}

// hashParts splits an image hash into its directories:
// {h0-3}/{h3-6}/{h6-9}/{h9-32}
func hashParts(hash string) ([]string, error) {
	if len(hash) != 32 || strings.Trim(hash, "0123456789abcdef") != "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidHash, hash)
	}
	return []string{hash[0:3], hash[3:6], hash[6:9], hash[9:32]}, nil
}

// countFiles returns the number of files under dir; 0 when it does not exist
func countFiles(dir string) (int, error) {
	files := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return files, err
}

// removeEmptyParents removes dir and its parents while they are empty, up
// to but not including stop
func removeEmptyParents(dir, stop string) {
	for dir != stop && strings.HasPrefix(dir, stop+string(filepath.Separator)) {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
