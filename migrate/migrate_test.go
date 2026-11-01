package migrate

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

const img = "6e0/072/682/e66287b662827da75b244a3"

// fakeStore records uploads and serves listings from them
type fakeStore struct {
	mu      sync.Mutex
	objects map[string]string // key -> content type
	fail    map[string]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string]string{}, fail: map[string]bool{}}
}

func (s *fakeStore) ListDirectory(dir string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var names []string
	for k := range s.objects {
		if strings.HasPrefix(k, dir+"/") {
			names = append(names, strings.TrimPrefix(k, dir+"/"))
		}
	}
	return names, nil
}

func (s *fakeStore) Upload(source, destination, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail[destination] {
		return errors.New("upload failed")
	}
	if _, err := os.Stat(source); err != nil {
		return err
	}
	s.objects[destination] = contentType
	return nil
}

func (s *fakeStore) keys() []string {
	var keys []string
	for k := range s.objects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// tree has two images in two namespaces, a temp file and stray files
func tree(t *testing.T) string {
	root := t.TempDir()
	write(t, root, "acct/"+img+"/original", "heic bytes")
	write(t, root, "acct/"+img+"/info.json", `{"hash":"x","content_type":"image/heic"}`)
	write(t, root, "acct/"+img+"/w300.jpg", "jpg")
	write(t, root, "acct/"+img+"/x200.webp", "webp")
	write(t, root, "acct/"+img+"/.download-123", "partial")
	write(t, root, "other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/original", "png bytes")
	write(t, root, "other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/info.json", `{"content_type":"image/png"}`)
	write(t, root, "tmp/abc", "temporary")
	write(t, root, "acct/not-a-hash/original", "stray")
	return root
}

func equal(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got %v\nwant %v", got, want)
	}
}

func TestMigrateOriginalsAndInfo(t *testing.T) {
	root := tree(t)
	store := newFakeStore()

	summary, err := Run(Options{Root: root, RemoteBase: "images"}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary, Summary{Images: 2, Uploaded: 4, Skipped: 2})
	equal(t, store.objects, map[string]string{
		"images/acct/" + img + "/original":                           "image/heic",
		"images/acct/" + img + "/info.json":                          "application/json",
		"images/other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/original":  "image/png",
		"images/other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/info.json": "application/json",
	})

	// Running again uploads nothing
	summary, err = Run(Options{Root: root, RemoteBase: "images"}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary, Summary{Images: 2, Existing: 4, Skipped: 2})
}

func TestMigrateVariants(t *testing.T) {
	store := newFakeStore()
	summary, err := Run(Options{Root: tree(t), Variants: true}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary, Summary{Images: 2, Uploaded: 6})
	equal(t, store.objects["acct/"+img+"/w300.jpg"], "image/jpeg")
	equal(t, store.objects["acct/"+img+"/x200.webp"], "image/webp")
}

func TestMigrateDryRun(t *testing.T) {
	store := newFakeStore()
	var out bytes.Buffer
	summary, err := Run(Options{Root: tree(t), RemoteBase: "images", DryRun: true, Log: &out}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary.Uploaded, 4)
	equal(t, len(store.objects), 0)
	if !strings.Contains(out.String(), "would upload images/acct/"+img+"/original (image/heic)") {
		t.Fatalf("dry run output:\n%s", out.String())
	}
}

func TestMigrateOneNamespace(t *testing.T) {
	store := newFakeStore()
	summary, err := Run(Options{Root: tree(t), Namespace: "other"}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary.Images, 1)
	equal(t, store.keys(), []string{
		"other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/info.json",
		"other/31e/8b3/187/a9f63f26d58c88bf09a7bbd/original",
	})
}

// A failed upload is counted and retried on the next run
func TestMigrateCountsFailures(t *testing.T) {
	root := tree(t)
	store := newFakeStore()
	store.fail["acct/"+img+"/original"] = true

	summary, err := Run(Options{Root: root}, store)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, summary.Failed, 1)
	equal(t, summary.Uploaded, 3)

	delete(store.fail, "acct/"+img+"/original")
	summary, _ = Run(Options{Root: root}, store)
	equal(t, summary, Summary{Images: 2, Uploaded: 1, Existing: 3, Skipped: 2})
}

// Without a content type in info.json, the original's type is detected
func TestMigrateDetectsContentTypeWithoutInfo(t *testing.T) {
	root := t.TempDir()
	data, err := os.ReadFile("../test/images/a.png")
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "ns/"+img+"/original", string(data))
	store := newFakeStore()
	if _, err := Run(Options{Root: root}, store); err != nil {
		t.Fatal(err)
	}
	equal(t, store.objects["ns/"+img+"/original"], "image/png")
}
