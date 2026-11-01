package namespaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	vipsadapter "github.com/image-server/image-server/processor/vips"
)

// Remote is remote storage that holds every namespace, with local files only
// a cache of it. S3 has no rename, so a rename copies every object.
type Remote interface {
	ListKeys(prefix string) ([]string, error)
	Copy(src, dst string) error
	DeleteKeys(keys []string) error
	PutBytes(key string, data []byte, contentType string) error
	GetBytes(key string) (data []byte, found bool, err error)
}

// Storage is where namespaces live: local files under Root, plus Remote
// under RemoteBase when set
type Storage struct {
	Root       string
	Remote     Remote
	RemoteBase string
}

// renameMarker is written under the target while a remote rename is in
// progress. It tells a retry of the same rename that the target is a partial
// copy to finish, not a namespace that was already there.
const renameMarker = ".rename.json"

type marker struct {
	From string `json:"from"`
}

// remotePrefix is the key prefix of a namespace, ending in a slash so it
// never matches a longer name
func (s Storage) remotePrefix(namespace string) string {
	return path.Join(s.RemoteBase, namespace) + "/"
}

// renameRemote copies from to to, then deletes from. Both write locks are
// held. It is safe to retry after any failure: the marker lets a retry of
// the same rename resume, and keys already copied are not copied again.
func (s Storage) renameRemote(from, to, fromDir, toDir string) (string, error) {
	fromPrefix, toPrefix := s.remotePrefix(from), s.remotePrefix(to)
	markerKey := toPrefix + renameMarker

	fromKeys, err := s.Remote.ListKeys(fromPrefix)
	if err != nil {
		return "", err
	}
	toKeys, err := s.Remote.ListKeys(toPrefix)
	if err != nil {
		return "", err
	}
	resuming, err := s.ownsMarker(markerKey, from)
	if err != nil {
		return "", err
	}
	_, localFromErr := os.Lstat(fromDir)
	localFrom := localFromErr == nil

	if len(fromKeys) == 0 && !localFrom {
		if !resuming {
			return "missing", nil
		}
		// An earlier attempt moved everything but stopped before the end
		if err := s.Remote.DeleteKeys([]string{markerKey}); err != nil {
			return "", err
		}
		return "renamed", nil
	}

	copied := map[string]bool{}
	for _, key := range toKeys {
		if key != markerKey {
			copied[strings.TrimPrefix(key, toPrefix)] = true
		}
	}
	_, localToErr := os.Lstat(toDir)
	if !resuming && (len(copied) > 0 || localToErr == nil || hasKey(toKeys, markerKey)) {
		return "", fmt.Errorf("%w: %s", ErrExists, to)
	}

	if !resuming {
		data, _ := json.Marshal(marker{From: from})
		if err := s.Remote.PutBytes(markerKey, data, "application/json"); err != nil {
			return "", err
		}
	}

	for _, key := range fromKeys {
		rel := strings.TrimPrefix(key, fromPrefix)
		if rel == renameMarker || copied[rel] {
			continue
		}
		if err := s.Remote.Copy(key, toPrefix+rel); err != nil {
			return "", fmt.Errorf("copy %s: %w", key, err)
		}
	}
	if err := s.Remote.DeleteKeys(fromKeys); err != nil {
		return "", err
	}

	// Local files are a cache of the bucket: drop the old ones
	if err := os.RemoveAll(fromDir); err != nil {
		return "", err
	}
	vipsadapter.Forget(fromDir)
	vipsadapter.Forget(toDir)

	if err := s.Remote.DeleteKeys([]string{markerKey}); err != nil {
		return "", err
	}
	return "renamed", nil
}

// ownsMarker is true when the target's rename marker names from
func (s Storage) ownsMarker(markerKey, from string) (bool, error) {
	data, found, err := s.Remote.GetBytes(markerKey)
	if err != nil || !found {
		return false, err
	}
	var m marker
	if err := json.Unmarshal(data, &m); err != nil {
		return false, nil
	}
	return m.From == from, nil
}

// deleteRemote removes every object of a namespace, then its local cache.
// The write lock is held. A retry after a failure deletes what is left.
func (s Storage) deleteRemote(namespace, nsDir string) (Result, error) {
	prefix := s.remotePrefix(namespace)
	keys, err := s.Remote.ListKeys(prefix)
	if err != nil {
		return Result{}, err
	}
	_, localErr := os.Lstat(nsDir)
	if len(keys) == 0 && localErr != nil {
		return Result{Status: "missing"}, nil
	}

	images := map[string]bool{}
	files := 0
	for _, key := range keys {
		rel := strings.TrimPrefix(key, prefix)
		if rel == renameMarker {
			continue
		}
		files++
		// {h0-3}/{h3-6}/{h6-9}/{h9-32}/{file}
		if parts := strings.Split(rel, "/"); len(parts) == 5 {
			images[strings.Join(parts[:4], "/")] = true
		}
	}

	if err := s.Remote.DeleteKeys(keys); err != nil {
		return Result{}, err
	}
	if err := os.RemoveAll(nsDir); err != nil {
		return Result{}, err
	}
	vipsadapter.Forget(nsDir)
	return Result{Status: "deleted", Images: len(images), Files: files}, nil
}

func hasKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}
