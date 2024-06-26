package fetcher

import (
	"io"
	"os"
	"path/filepath"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
)

// SourceFetcher handles fetching source original images
// Used to download images from an external site
type SourceFetcher struct {
	Paths core.Paths
}

// NewSourceFetcher initializes a SourceFetcher
func NewSourceFetcher(paths core.Paths) *SourceFetcher {
	return &SourceFetcher{paths}
}

// Fetch returns ImageDetails of downloaded file
// It will only download the image once, even if multiple concurrent requests to the same url are made
// downloaded is false when the file was already present locally
func (f *SourceFetcher) Fetch(url string, namespace string) (*info.ImageProperties, bool, error) {
	c := make(chan FetchResult)
	go f.uniqueFetchSource(c, url, namespace)
	r := <-c
	return r.ImageDetails, r.Downloaded, r.Error
}

func (f *SourceFetcher) StoreBinary(body io.ReadCloser, namespace string, contentType string) (*info.ImageProperties, error) {
	tmpOriginalPath := f.Paths.RandomTempPath()
	defer body.Close()

	dir := filepath.Dir(tmpOriginalPath)
	os.MkdirAll(dir, 0700)

	out, err := os.Create(tmpOriginalPath)
	if err != nil {
		return nil, err
	}

	defer out.Close()

	_, err = io.Copy(out, body)
	if err != nil {
		return nil, err
	}

	i := info.Info{Path: tmpOriginalPath}

	md5, err := i.FileHash()
	if err != nil {
		return nil, err
	}

	destination := f.Paths.LocalOriginalPath(namespace, md5)
	ensureDestinationDirectory(destination)
	err = os.Rename(tmpOriginalPath, destination)
	i.Path = destination
	i.ContentType = contentType

	return i.ImageDetails()
}

// Even if simultaneous calls request the same image, only the first one will download
// the image, and will then notify all requesters. The channel returns an error object
func (f *SourceFetcher) uniqueFetchSource(c chan FetchResult, url string, namespace string) {
	// download temp source
	tmpOriginalPath, downloaded, err := f.downloadTempSource(url)
	if err != nil {
		f.notifyDownloadSourceFailed(c, err)
		return
	}

	// file hash the image url
	md5, err := info.Info{Path: tmpOriginalPath}.FileHash()
	if err != nil {
		f.notifyDownloadSourceFailed(c, err)
		return
	}

	// move file to destination
	destination := f.Paths.LocalOriginalPath(namespace, md5)
	err = f.copyImageFromTmp(tmpOriginalPath, destination)
	if err != nil {
		f.notifyDownloadSourceFailed(c, err)
		return
	}

	// generate image details
	imageDetails, err := info.Info{Path: destination}.ImageDetails()
	if err != nil {
		f.notifyDownloadSourceFailed(c, err)
		return
	}

	c <- FetchResult{nil, imageDetails, downloaded}
	close(c)
}

func (f *SourceFetcher) copyImageFromTmp(tmpOriginalPath string, destination string) error {
	// only copy image if does not exist
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		ensureDestinationDirectory(destination)
		err := os.Link(tmpOriginalPath, destination)

		if err != nil {
			return err
		}
	}
	return nil
}

func ensureDestinationDirectory(path string) {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0700)
}

// downloadedTempSource returns the path of the downloaded source
// downloaded is false when the file was already present locally
func (f *SourceFetcher) downloadTempSource(url string) (string, bool, error) {
	tmpOriginalPath := f.Paths.TempImagePath(url)
	fetcher := NewUniqueFetcher(url, tmpOriginalPath)
	downloaded, err := fetcher.Fetch()
	return tmpOriginalPath, downloaded, err
}

func (f *SourceFetcher) notifyDownloadSourceFailed(c chan FetchResult, err error) {
	c <- FetchResult{err, nil, false}
	close(c)
}
