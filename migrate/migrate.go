// Package migrate copies a local image tree to remote storage, with the
// layout and content types the server uses when it uploads.
package migrate

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/mime"
)

// Store is the remote storage to copy into
type Store interface {
	ListDirectory(directory string) ([]string, error)
	Upload(source, destination, contentType string) error
}

// Options controls a migration
type Options struct {
	Root       string // local base path
	RemoteBase string // remote base path
	Namespace  string // only this namespace; empty for all
	Variants   bool   // also copy variants, not only originals and info.json
	DryRun     bool   // report what would be uploaded, upload nothing
	Workers    int
	Log        io.Writer
}

// Summary counts the files of a migration
type Summary struct {
	Images   int
	Uploaded int
	Existing int // already in remote storage, not uploaded again
	Skipped  int // variants when not copying them, and unrecognised files
	Failed   int
}

var (
	namespaceName = regexp.MustCompile(`^[a-z0-9_-]+$`)
	hashParts     = []*regexp.Regexp{
		regexp.MustCompile(`^[a-f0-9]{3}$`), regexp.MustCompile(`^[a-f0-9]{3}$`),
		regexp.MustCompile(`^[a-f0-9]{3}$`), regexp.MustCompile(`^[a-f0-9]{23}$`),
	}
)

// Run copies every image directory under Root that is missing remotely.
// Files already in remote storage are left as they are, so it can be run
// again after a failure or while the server keeps uploading.
func Run(opts Options, store Store) (Summary, error) {
	if opts.Workers <= 0 {
		opts.Workers = 8
	}
	if opts.Log == nil {
		opts.Log = io.Discard
	}
	dirs, err := imageDirs(opts.Root, opts.Namespace)
	if err != nil {
		return Summary{}, err
	}

	var mu sync.Mutex
	var summary Summary
	summary.Images = len(dirs)
	logf := func(format string, args ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(opts.Log, format+"\n", args...)
	}

	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				s := migrateImage(opts, store, rel, logf)
				mu.Lock()
				summary.Uploaded += s.Uploaded
				summary.Existing += s.Existing
				summary.Skipped += s.Skipped
				summary.Failed += s.Failed
				mu.Unlock()
			}
		}()
	}
	for _, rel := range dirs {
		jobs <- rel
	}
	close(jobs)
	wg.Wait()
	return summary, nil
}

// imageDirs returns the image directories under root, relative to it:
// {namespace}/{h0-3}/{h3-6}/{h6-9}/{h9-32}. tmp/ and other names that are
// not image paths are ignored.
func imageDirs(root, only string) ([]string, error) {
	namespaces, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, ns := range namespaces {
		name := ns.Name()
		if !ns.IsDir() || name == "tmp" || !namespaceName.MatchString(name) || (only != "" && name != only) {
			continue
		}
		var walk func(rel string, depth int) error
		walk = func(rel string, depth int) error {
			if depth == len(hashParts) {
				dirs = append(dirs, rel)
				return nil
			}
			entries, err := os.ReadDir(filepath.Join(root, rel))
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.IsDir() && hashParts[depth].MatchString(e.Name()) {
					if err := walk(path.Join(rel, e.Name()), depth+1); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if err := walk(name, 0); err != nil {
			return nil, err
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func migrateImage(opts Options, store Store, rel string, logf func(string, ...interface{})) Summary {
	var s Summary
	localDir := filepath.Join(opts.Root, filepath.FromSlash(rel))
	remoteDir := path.Join(opts.RemoteBase, rel)

	entries, err := os.ReadDir(localDir)
	if err != nil {
		logf("FAILED %s: %v", rel, err)
		s.Failed++
		return s
	}
	existing := map[string]bool{}
	names, err := store.ListDirectory(remoteDir)
	if err != nil {
		logf("FAILED %s: listing remote files: %v", rel, err)
		s.Failed++
		return s
	}
	for _, n := range names {
		existing[n] = true
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		local := filepath.Join(localDir, name)
		contentType := ""
		switch {
		case name == "original":
			if existing[name] {
				s.Existing++
				continue
			}
			contentType = originalContentType(localDir)
		case name == "info.json":
			contentType = "application/json"
		case !opts.Variants:
			s.Skipped++
			continue
		default:
			ext := strings.TrimPrefix(path.Ext(name), ".")
			if ext == "" {
				logf("SKIPPED %s/%s: no extension, unknown content type", rel, name)
				s.Skipped++
				continue
			}
			contentType = mime.ExtToContentType(ext)
		}

		if existing[name] {
			s.Existing++
			continue
		}
		remote := path.Join(remoteDir, name)
		if opts.DryRun {
			logf("would upload %s (%s)", remote, contentType)
			s.Uploaded++
			continue
		}
		if err := store.Upload(local, remote, contentType); err != nil {
			logf("FAILED %s: %v", remote, err)
			s.Failed++
			continue
		}
		logf("uploaded %s (%s)", remote, contentType)
		s.Uploaded++
	}
	return s
}

// originalContentType is the content type recorded in info.json, or detected
// from the file when info.json is missing or has none
func originalContentType(localDir string) string {
	if data, err := os.ReadFile(filepath.Join(localDir, "info.json")); err == nil {
		var stored struct {
			ContentType string `json:"content_type"`
		}
		if json.Unmarshal(data, &stored) == nil && stored.ContentType != "" {
			return stored.ContentType
		}
	}
	contentType := "application/octet-stream"
	if details, err := (info.Info{Path: filepath.Join(localDir, "original")}).ImageDetails(); err == nil && details.ContentType != "" {
		contentType = details.ContentType
	}
	log.Printf("migrate: no content type in info.json for %s, detected %s", localDir, contentType)
	return contentType
}
