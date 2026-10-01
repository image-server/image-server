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

// receive reads the notification a caller gets after a successful CreateImage
func receive(t *testing.T, channels *processor.ProcessorChannels) {
	t.Helper()
	select {
	case <-channels.ImageProcessed:
	case <-channels.Skipped:
	case <-time.After(5 * time.Second):
		t.Fatal("no notification after a successful CreateImage")
	}
}

// Every concurrent request for the same file gets the result, and each
// successful one gets a notification
func TestCreateImageConcurrentRequests(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "w50.jpg")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, channels := newProcessor("../test/images/wine.jpg", destination)
			Ok(t, p.CreateImage())
			receive(t, channels)
		}()
	}
	wg.Wait()
	ExpectFile(t, destination)
}

// Callers stop reading the notification channels when CreateImage fails, so
// a failure must not leave goroutines blocked on them
func TestCreateImageErrorDoesNotLeakGoroutines(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "w50.jpg")
	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, _ := newProcessor("../test/images/missing.jpg", destination)
			Assert(t, p.CreateImage() != nil, "expected an error for a missing source")
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	Assert(t, runtime.NumGoroutine() <= before, "leaked goroutines: %d before, %d after", before, runtime.NumGoroutine())
}
