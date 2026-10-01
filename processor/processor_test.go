package processor_test

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/processor"
	. "github.com/image-server/image-server/test"
)

func newProcessor(source, destination string) (*processor.Processor, *processor.ProcessorChannels) {
	channels := &processor.ProcessorChannels{
		ImageProcessed: make(chan *core.ImageConfiguration),
		Skipped:        make(chan string),
	}
	return &processor.Processor{
		Source:             source,
		Destination:        destination,
		ImageConfiguration: &core.ImageConfiguration{Width: 50, Format: "jpg", Quality: 90},
		ImageDetails:       &info.ImageProperties{Width: 800, Height: 600},
		Channels:           channels,
	}, channels
}

// notification returns what a caller gets after a successful CreateImage:
// true for ImageProcessed, false for Skipped
func notification(t *testing.T, channels *processor.ProcessorChannels) bool {
	t.Helper()
	select {
	case <-channels.ImageProcessed:
		return true
	case <-channels.Skipped:
		return false
	case <-time.After(5 * time.Second):
		t.Fatal("no notification after a successful CreateImage")
		return false
	}
}

// waitFor fails instead of hanging when callers deadlock
func waitFor(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("CreateImage callers did not finish")
	}
}

// Concurrent requests for one file all get the result; exactly one of them
// generates it and the rest are told it was skipped
func TestCreateImageConcurrentRequests(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "w50.jpg")

	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := 0
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, channels := newProcessor("../test/images/wine.jpg", destination)
			<-start // release all at once so they overlap
			Ok(t, p.CreateImage())
			if notification(t, channels) {
				mu.Lock()
				processed++
				mu.Unlock()
			}
		}()
	}
	close(start)
	waitFor(t, &wg)

	ExpectFile(t, destination)
	Equals(t, 1, processed)
}

// Callers stop reading the notification channels when CreateImage fails, so
// a failure must not leave goroutines blocked on them
func TestCreateImageErrorDoesNotLeakGoroutines(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "w50.jpg")
	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, _ := newProcessor("../test/images/missing.jpg", destination)
			<-start
			Assert(t, p.CreateImage() != nil, "expected an error for a missing source")
		}()
	}
	close(start)
	waitFor(t, &wg)

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	Assert(t, runtime.NumGoroutine() <= before, "leaked goroutines: %d before, %d after", before, runtime.NumGoroutine())
}

// A failure must not leave the destination marked as in progress
func TestCreateImageCanRetryAfterError(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "w50.jpg")

	failing, _ := newProcessor("../test/images/missing.jpg", destination)
	Assert(t, failing.CreateImage() != nil, "expected an error for a missing source")

	retry, channels := newProcessor("../test/images/wine.jpg", destination)
	done := make(chan error, 1)
	go func() { done <- retry.CreateImage() }()
	select {
	case err := <-done:
		Ok(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("retry is still waiting on the failed attempt")
	}
	Equals(t, true, notification(t, channels))
	ExpectFile(t, destination)
}
