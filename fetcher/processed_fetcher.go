package fetcher

import (
	"errors"
	"os"

	"github.com/image-server/image-server/core"
)

// ProcessedFetcher handles fetching an already processed image
type ProcessedFetcher struct {
	Paths core.Paths
}

// NewProcessedFetcher initializes a ProcessedFetcher
func NewProcessedFetcher(paths core.Paths) *ProcessedFetcher {
	return &ProcessedFetcher{paths}
}

// Fetch downlods an already processed image
func (f *ProcessedFetcher) Fetch(ic *core.ImageConfiguration) (err error) {
	destination := f.Paths.LocalImagePath(ic.Namespace, ic.ID, ic.Filename)
	source := f.Paths.RemoteImageURL(ic.Namespace, ic.ID, ic.Filename)
	key := f.Paths.RemoteImagePath(ic.Namespace, ic.ID, ic.Filename)

	uf := NewRemoteFetcher(source, key, destination)
	_, err = uf.Fetch()

	if err != nil {
		return err
	}

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return errors.New("unable to find already processed image")
	}

	return err
}
