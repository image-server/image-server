package fetcher

import (
	"log"
	"os"
	"path/filepath"

	httpFetcher "github.com/image-server/image-server/fetcher/http"
)

type UniqueFetcher struct {
	Source      string
	Destination string
}

func NewUniqueFetcher(source string, destination string) *UniqueFetcher {
	return &UniqueFetcher{source, destination}
}

// Fetch returns a boolean to denote if the image was downloaded.
// This value is false when the image is already present in the filesystem
func (f *UniqueFetcher) Fetch() (bool, error) {
	c := make(chan FetchResult)
	go f.uniqueFetch(c)
	r := <-c
	return r.Downloaded, r.Error
}

// Even if simultaneous calls request the same image, only the first one will download
// the image, and will then notify all requesters. The channel returns an error object
func (f *UniqueFetcher) uniqueFetch(c chan FetchResult) {
	url := f.Source
	destination := f.Destination
	var err error

	mu.Lock()
	_, present := ImageDownloads[url]
	if present {
		ImageDownloads[url] = append(ImageDownloads[url], c)
		mu.Unlock()
	} else {
		ImageDownloads[url] = []chan FetchResult{c}
		mu.Unlock()

		defer func() {
			mu.Lock()
			delete(ImageDownloads, url)
			mu.Unlock()
		}()

		// only copy image if does not exist
		if _, err = os.Stat(destination); os.IsNotExist(err) {
			dir := filepath.Dir(destination)
			os.MkdirAll(dir, 0700)

			fetcher := &httpFetcher.Fetcher{}
			err = fetcher.Fetch(url, destination)
		}

		mu.Lock()
		if err == nil {
			log.Printf("Notifying download complete for path %s", destination)
			f.notifyDownloadComplete(url)
		} else {
			log.Printf("Unable to download image %s", err)
			f.notifyDownloadFailed(url, err)
		}
		mu.Unlock()
	}
}

func (f *UniqueFetcher) notifyDownloadComplete(url string) {
	for i, cc := range ImageDownloads[url] {
		downloaded := i == 0
		fr := FetchResult{nil, nil, downloaded}
		cc <- fr
		close(cc)
	}
}

func (f *UniqueFetcher) notifyDownloadFailed(url string, err error) {
	for _, cc := range ImageDownloads[url] {
		fr := FetchResult{err, nil, false}
		cc <- fr
		close(cc)
	}
}
