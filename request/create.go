package request

import (
	"log"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/fetcher"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/logger"
	"github.com/image-server/image-server/uploader"
)

func (r *Request) Create() (*info.ImageProperties, error) {
	f := fetcher.NewSourceFetcher(r.Paths)
	var imageDetails *info.ImageProperties
	var downloaded bool
	var err error

	if r.SourceURL != "" {
		imageDetails, downloaded, err = f.Fetch(r.SourceURL, r.Namespace)
	} else {
		imageDetails, err = f.StoreBinary(r.SourceData, r.Namespace, r.ContentType)
		downloaded = true
	}

	if err != nil {
		return nil, err
	}

	r.Hash = imageDetails.Hash

	if downloaded {
		err = r.UploadOriginal(imageDetails)
		if err != nil {
			return nil, err
		}
	}

	err = r.ProcessMultiple()
	if err != nil {
		return nil, err
	}
	return imageDetails, nil
}

func (r *Request) DownloadOriginal() error {
	localOriginalPath := r.Paths.LocalOriginalPath(r.Namespace, r.Hash)
	remoteOriginalPath := r.Paths.RemoteOriginalURL(r.Namespace, r.Hash)

	// download original image
	f := fetcher.NewUniqueFetcher(remoteOriginalPath, localOriginalPath)
	_, err := f.Fetch()
	return err
}

func (r *Request) UploadOriginal(imageDetails *info.ImageProperties) error {
	uploader := uploader.DefaultUploader(r.ServerConfiguration)
	err := uploader.CreateDirectory(r.Paths.RemoteImageDirectory(r.Namespace, imageDetails.Hash))
	if err != nil {
		return err
	}

	localOriginalPath := r.Paths.LocalOriginalPath(r.Namespace, imageDetails.Hash)

	destination := r.Paths.RemoteOriginalPath(r.Namespace, imageDetails.Hash)

	r.UploadImageDetails(imageDetails, uploader)

	// upload original image
	err = uploader.Upload(localOriginalPath, destination, imageDetails.ContentType)
	if err != nil {
		return err
	}

	// Notify loggers (including webhooks) that original was uploaded
	logger.OriginalUploaded(&core.ImageProperties{
		Hash:        imageDetails.Hash,
		Height:      imageDetails.Height,
		Width:       imageDetails.Width,
		ContentType: imageDetails.ContentType,
	}, r.Namespace)

	return nil
}

func (r *Request) UploadImageDetails(imageDetails *info.ImageProperties, uploader *uploader.Uploader) {
	localInfoPath := r.Paths.LocalInfoPath(r.Namespace, imageDetails.Hash)
	remoteInfoPath := r.Paths.RemoteInfoPath(r.Namespace, imageDetails.Hash)

	err := info.SaveImageDetail(imageDetails, localInfoPath)
	if err != nil {
		log.Println(err)
		return
	}

	// upload info
	err = uploader.Upload(localInfoPath, remoteInfoPath, "application/json")
	if err != nil {
		log.Println(err)
	}
}
