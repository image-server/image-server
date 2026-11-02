package namespaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

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

// renameMarker is kept under the target of a remote rename. While the
// rename runs ("running"), it tells a retry of the same rename that the
// target is a partial copy to finish, not a namespace that was already there.
// After it completes ("done"), it lets a retry of the same rename answer
// "renamed" instead of "missing". A later rename of the target removes it,
// with the rest of the source.
const renameMarker = ".rename.json"

const (
	markerRunning = "running"
	markerDone    = "done"
)

// CopyWorkers is how many objects a remote rename copies at once
var CopyWorkers = 16

type marker struct {
	From   string `json:"from"`
	Status string `json:"status"`
	// Count is the number of objects in the target, excluding the marker,
	// when the rename completed
	Count int `json:"count,omitempty"`
}

// remotePrefix is the key prefix of a namespace, ending in a slash so it
// never matches a longer name
func (s Storage) remotePrefix(namespace string) string {
	return path.Join(s.RemoteBase, namespace) + "/"
}

// renameRemote copies from to to, then deletes from. Both write locks are
// held. It is safe to retry after any failure: a running marker lets a
// retry of the same rename resume, and keys already copied are not copied
// again. A retry after completion answers "renamed" from the done marker.
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
	m, err := s.readMarker(markerKey)
	if err != nil {
		return "", err
	}
	ours := m != nil && m.From == from
	// Markers written before the status field existed were all in progress
	running := ours && (m.Status == markerRunning || m.Status == "")
	done := ours && m.Status == markerDone

	// The source's own marker (from an earlier rename into it) is not
	// content: it is never copied, and is deleted with the source
	sourceObjects := 0
	for _, key := range fromKeys {
		if key != fromPrefix+renameMarker {
			sourceObjects++
		}
	}
	copied := map[string]bool{}
	for _, key := range toKeys {
		if key != markerKey {
			copied[strings.TrimPrefix(key, toPrefix)] = true
		}
	}
	localFrom, err := hasFiles(fromDir)
	if err != nil {
		return "", err
	}
	localTo, err := hasFiles(toDir)
	if err != nil {
		return "", err
	}

	if sourceObjects == 0 && !localFrom {
		switch {
		case running:
			// An earlier attempt moved everything but stopped before the end
			if err := s.Remote.DeleteKeys(fromKeys); err != nil {
				return "", err
			}
			if err := s.writeMarker(markerKey, marker{From: from, Status: markerDone, Count: len(copied)}); err != nil {
				return "", err
			}
			return "renamed", nil
		case done && len(copied) >= m.Count:
			// This rename already completed
			return "renamed", nil
		case done:
			// It completed, but objects have gone from the target since
			return "", fmt.Errorf("%w: %s holds %d objects, its rename recorded %d", ErrExists, to, len(copied), m.Count)
		}
		return "missing", nil
	}

	// Anything in the target that is not this rename in progress: never
	// overwrite it. A done marker is not resumed: the source has new content.
	if !running && (len(copied) > 0 || localTo || m != nil) {
		return "", fmt.Errorf("%w: %s", ErrExists, to)
	}

	if !running {
		if err := s.writeMarker(markerKey, marker{From: from, Status: markerRunning}); err != nil {
			return "", err
		}
	}

	var pending []string
	for _, key := range fromKeys {
		rel := strings.TrimPrefix(key, fromPrefix)
		if rel != renameMarker && !copied[rel] {
			pending = append(pending, rel)
		}
	}
	// Copies are independent, and a store takes about as long for one as
	// for several at once
	err = parallel(len(pending), CopyWorkers, func(i int) error {
		rel := pending[i]
		if err := s.Remote.Copy(fromPrefix+rel, toPrefix+rel); err != nil {
			return fmt.Errorf("copy %s: %w", fromPrefix+rel, err)
		}
		return nil
	})
	if err != nil {
		return "", err
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

	count := len(copied) + len(pending)
	if err := s.writeMarker(markerKey, marker{From: from, Status: markerDone, Count: count}); err != nil {
		return "", err
	}
	return "renamed", nil
}

// readMarker returns the rename marker at key, or nil when there is none.
// A marker that cannot be read is treated as foreign: it names no source.
func (s Storage) readMarker(key string) (*marker, error) {
	data, found, err := s.Remote.GetBytes(key)
	if err != nil || !found {
		return nil, err
	}
	var m marker
	if err := json.Unmarshal(data, &m); err != nil {
		return &marker{}, nil
	}
	return &m, nil
}

func (s Storage) writeMarker(key string, m marker) error {
	data, _ := json.Marshal(m)
	return s.Remote.PutBytes(key, data, "application/json")
}

// deleteRemote removes every object of a namespace, then its local cache.
// The write lock is held. A retry after a failure deletes what is left.
func (s Storage) deleteRemote(namespace, nsDir string) (Result, error) {
	prefix := s.remotePrefix(namespace)
	keys, err := s.Remote.ListKeys(prefix)
	if err != nil {
		return Result{}, err
	}
	local, err := hasFiles(nsDir)
	if err != nil {
		return Result{}, err
	}
	if len(keys) == 0 && !local {
		// Clear directories left with no files
		if err := os.RemoveAll(nsDir); err != nil {
			return Result{}, err
		}
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

// parallel calls fn(0..n-1) with at most workers running at once. It stops
// starting new calls after the first error, and returns it.
func parallel(n, workers int, fn func(i int) error) error {
	var mu sync.Mutex
	var firstErr error
	next := 0
	var wg sync.WaitGroup
	for w := 0; w < workers && w < n; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				if firstErr != nil || next >= n {
					mu.Unlock()
					return
				}
				i := next
				next++
				mu.Unlock()
				if err := fn(i); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return firstErr
}
