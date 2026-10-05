package vips

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
)

// countLoads wraps loadSource for the duration of a test
func countLoads(t *testing.T) *int32 {
	var n int32
	original := loadSource
	loadSource = func(path string) (*vips.ImageRef, error) {
		atomic.AddInt32(&n, 1)
		return original(path)
	}
	t.Cleanup(func() { loadSource = original })
	return &n
}

func copyFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../test/images/wine.jpg")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "original")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSourceCacheDecodesOnceForConcurrentVariants(t *testing.T) {
	if !Available {
		t.Skip("vips not available, skipping tests")
	}
	loads := countLoads(t)
	c := newSourceCache(time.Minute, 512<<20)
	path := copyFixture(t)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			img, err := c.acquire(path)
			if err != nil {
				t.Error(err)
				return
			}
			defer img.Close()
			if img.Width() != 800 || img.Height() != 600 {
				t.Errorf("got %dx%d", img.Width(), img.Height())
			}
		}()
	}
	wg.Wait()

	if n := atomic.LoadInt32(loads); n != 1 {
		t.Fatalf("decoded %d times, want 1", n)
	}
}

func TestSourceCacheEvictsIdleEntries(t *testing.T) {
	if !Available {
		t.Skip("vips not available, skipping tests")
	}
	loads := countLoads(t)
	c := newSourceCache(time.Hour, 512<<20)
	path := copyFixture(t)

	img, err := c.acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	img.Close()

	c.evict(time.Now().Add(2 * time.Hour))
	if len(c.entries) != 0 || c.bytes != 0 {
		t.Fatalf("entries=%d bytes=%d after TTL, want none", len(c.entries), c.bytes)
	}

	img, err = c.acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	img.Close()
	if n := atomic.LoadInt32(loads); n != 2 {
		t.Fatalf("decoded %d times, want 2", n)
	}
}

func TestSourceCacheKeepsEntriesInUse(t *testing.T) {
	if !Available {
		t.Skip("vips not available, skipping tests")
	}
	// A zero budget evicts every idle entry at once
	c := newSourceCache(time.Hour, 0)
	path := copyFixture(t)

	img, err := c.acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()

	// Evicted once released, but the copy still reads its pixels
	if len(c.entries) != 0 {
		t.Fatalf("entries=%d, want idle entry evicted over budget", len(c.entries))
	}
	if _, _, err := img.ExportJpeg(vips.NewJpegExportParams()); err != nil {
		t.Fatalf("copy unusable after eviction: %v", err)
	}
}

func TestSourceCacheDoesNotCacheErrors(t *testing.T) {
	if !Available {
		t.Skip("vips not available, skipping tests")
	}
	c := newSourceCache(time.Minute, 512<<20)
	path := filepath.Join(t.TempDir(), "original")
	if err := os.WriteFile(path, []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := c.acquire(path); err == nil {
		t.Fatal("expected an error for a broken original")
	}
	if len(c.entries) != 0 {
		t.Fatalf("entries=%d, want failed load not cached", len(c.entries))
	}
}

func TestSourceCacheReloadsRewrittenFile(t *testing.T) {
	if !Available {
		t.Skip("vips not available, skipping tests")
	}
	c := newSourceCache(time.Minute, 512<<20)
	path := copyFixture(t)

	img, err := c.acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	img.Close()

	png, err := os.ReadFile("../../test/images/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, png, 0644); err != nil {
		t.Fatal(err)
	}

	img, err = c.acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	if img.Width() == 800 && img.Height() == 600 {
		t.Fatal("served the cached decode of the old file")
	}
}
