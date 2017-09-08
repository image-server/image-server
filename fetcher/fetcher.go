package fetcher

import (
	"sync"

	"github.com/image-server/image-server/info"
)

var mu sync.RWMutex // To protect ImageDownloads
var ImageDownloads map[string][]chan FetchResult

func init() {
	ImageDownloads = make(map[string][]chan FetchResult)
}

type FetchResult struct {
	Error        error
	ImageDetails *info.ImageProperties
	Downloaded   bool
}
