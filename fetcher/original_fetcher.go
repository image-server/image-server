package fetcher

import (
	"fmt"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
)

// OriginalFetcher is used to download orinal images either from our the image store or from the original source
type OriginalFetcher struct {
	Paths core.Paths
}

// Fetch returns the ImageDetail of downloaded file
// It will handle the following cases
//   - The image has never been uploaded
//   - In this case the URL of the image is required, but the image hash will be empty. The Image will be downloaded from the source
//   - The image has already been uploaded
//   - The image hash is present, the image url is optional. The image will be downloaded from our store
//   - If the image can't be found, and the source url is provided then the image will be downloaded again from the source
//
// Returns downloaded true only if it was downloaded from source
func (f OriginalFetcher) Fetch(namespace string, sourceURL string, imageHash string) (info *info.ImageProperties, downloaded bool, err error) {
	if sourceURL == "" && imageHash == "" {
		return nil, false, fmt.Errorf("Missing Hash & URL")
	}

	if imageHash != "" {
		info, err = f.fetchFromStore(namespace, imageHash)
	}

	if sourceURL != "" && (err != nil || imageHash == "") {
		info, downloaded, err = f.fetchFromSource(namespace, sourceURL)
	}

	return info, downloaded, err
}

func (f OriginalFetcher) fetchFromStore(namespace string, imageHash string) (details *info.ImageProperties, err error) {
	destination := f.Paths.LocalOriginalPath(namespace, imageHash)
	source := f.Paths.RemoteOriginalURL(namespace, imageHash)
	key := f.Paths.RemoteOriginalPath(namespace, imageHash)
	uf := NewRemoteFetcher(source, key, destination)
	_, err = uf.Fetch()

	if err != nil {
		return nil, err
	}

	i := &info.Info{Path: destination}
	details, err = i.ImageDetails()

	return details, err
}

func (f OriginalFetcher) fetchFromSource(namespace string, sourceURL string) (info *info.ImageProperties, downloaded bool, err error) {
	sf := NewSourceFetcher(f.Paths)
	return sf.Fetch(sourceURL, namespace)
}
