package vips

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
)

// Decoding the original is most of the work for a variant, and a new
// original is usually followed by a burst of variants (one GET per crop or
// size). sourceCache keeps each original decoded and upright for a while, so
// every variant in the burst works from the same pixels: the image is decoded
// once, by whichever variant needs pixels first. Images are read-only once
// cached, and each variant works on its own Copy.
var sources = newSourceCache(time.Minute, 512<<20)

type sourceCache struct {
	mu       sync.Mutex
	entries  map[string]*sourceEntry
	idleTTL  time.Duration // evict this long after the last variant finished
	maxBytes int64         // decoded size kept for idle entries
	bytes    int64
}

type sourceEntry struct {
	key       string
	ready     chan struct{} // closed once image or err is set
	image     *vips.ImageRef
	err       error
	size      int64
	refs      int
	lastUsed  time.Time
	forgotten bool // file deleted: drop once the last variant is done
}

func newSourceCache(idleTTL time.Duration, maxBytes int64) *sourceCache {
	c := &sourceCache{entries: map[string]*sourceEntry{}, idleTTL: idleTTL, maxBytes: maxBytes}
	go func() {
		for range time.Tick(idleTTL / 4) {
			c.evict(time.Now())
		}
	}()
	return c
}

// loadSource is replaced in tests to count decodes
var loadSource = loadUpright

// acquire returns a private copy of the decoded, auto-rotated original.
// Close it when done.
func (c *sourceCache) acquire(path string) (*vips.ImageRef, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("vips failed to load image: %w", err)
	}
	// A rewritten file gets a new entry; the stale one ages out
	key := fmt.Sprintf("%s\x00%d\x00%d", path, stat.Size(), stat.ModTime().UnixNano())

	c.mu.Lock()
	e, ok := c.entries[key]
	if !ok {
		e = &sourceEntry{key: key, ready: make(chan struct{})}
		c.entries[key] = e
	}
	e.refs++
	c.mu.Unlock()

	if !ok {
		image, err := loadSource(path)
		c.mu.Lock()
		e.image, e.err = image, err
		if e.err != nil {
			// Not cached: the next request tries again
			delete(c.entries, key)
		} else {
			e.size = int64(e.image.Width()) * int64(e.image.Height()) * int64(e.image.Bands())
			c.bytes += e.size
		}
		c.mu.Unlock()
		close(e.ready)
	}
	<-e.ready

	defer c.release(e)
	if e.err != nil {
		return nil, e.err
	}
	return e.image.Copy()
}

func (c *sourceCache) release(e *sourceEntry) {
	c.mu.Lock()
	e.refs--
	e.lastUsed = time.Now()
	if e.forgotten && e.refs == 0 && e.image != nil && c.entries[e.key] == e {
		c.remove(e.key, e)
	}
	c.mu.Unlock()
	c.evict(time.Now())
}

// evict closes idle entries past their TTL, then the least recently used idle
// entries while over the size budget
func (c *sourceCache) evict(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		var oldestKey string
		var oldest *sourceEntry
		for key, e := range c.entries {
			if e.refs > 0 || e.image == nil {
				continue
			}
			if now.Sub(e.lastUsed) >= c.idleTTL {
				c.remove(key, e)
				continue
			}
			if oldest == nil || e.lastUsed.Before(oldest.lastUsed) {
				oldestKey, oldest = key, e
			}
		}
		if oldest == nil || c.bytes <= c.maxBytes {
			return
		}
		c.remove(oldestKey, oldest)
	}
}

func (c *sourceCache) remove(key string, e *sourceEntry) {
	delete(c.entries, key)
	c.bytes -= e.size
	// Copies handed out hold their own libvips references, so closing here
	// never frees pixels a variant is still using
	e.image.Close()
}

// Forget drops the decoded originals of files under dir, which was deleted
func Forget(dir string) {
	sources.forget(dir)
}

func (c *sourceCache) forget(dir string) {
	prefix := filepath.Clean(dir) + string(filepath.Separator)
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, e := range c.entries {
		if !strings.HasPrefix(filepath.Clean(key), prefix) {
			continue
		}
		if e.refs > 0 || e.image == nil {
			e.forgotten = true
			continue
		}
		c.remove(key, e)
	}
}

func loadUpright(path string) (*vips.ImageRef, error) {
	image, err := vips.NewImageFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("vips failed to load image: %w", err)
	}
	// Bake EXIF orientation into pixels before resize/crop: metadata is
	// stripped on export, so a tag-only rotation would otherwise be lost.
	if err := image.AutoRotate(); err != nil {
		image.Close()
		return nil, fmt.Errorf("vips failed to auto-rotate image: %w", err)
	}
	return image, nil
}
