package request

import (
	"log"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/fetcher"
	"github.com/image-server/image-server/info"
	"github.com/image-server/image-server/processor"
)

// Process downloads or processes an image version
func (r *Request) Process(ic *core.ImageConfiguration) error {
	err := r.downloadProcessed(ic)
	if err == nil {
		return nil
	}

	err = r.processImage(ic)
	return err
}

// downloadProcessed download an existing image that has been already processed
func (r *Request) downloadProcessed(ic *core.ImageConfiguration) error {
	f := fetcher.NewProcessedFetcher(r.Paths)
	return f.Fetch(ic)
}

func (r *Request) processImage(ic *core.ImageConfiguration) error {
	localResizedPath := r.Paths.LocalImagePath(r.Namespace, r.Hash, ic.Filename)
	localOriginalPath := r.Paths.LocalOriginalPath(r.Namespace, r.Hash)

	// The original file will be downloaded only once, even when every dimension requests it
	err := r.DownloadOriginal()
	if err != nil {
		return err
	}

	// process image
	pchan := &processor.ProcessorChannels{
		ImageProcessed: make(chan *core.ImageConfiguration),
		Skipped:        make(chan string),
	}

	info := &info.Info{
		Path: localOriginalPath,
	}
	id, err := info.ImageDetails()
	if err != nil {
		return err
	}

	p := processor.Processor{
		Source:             localOriginalPath,
		Destination:        localResizedPath,
		ImageConfiguration: ic,
		ImageDetails:       id,
		Channels:           pchan,
	}

	err = p.CreateImage()
	if err != nil {
		return err
	}

	select {
	case <-pchan.ImageProcessed:
		log.Printf("Processed (resize handler) %s", localResizedPath)
		go r.uploadResizedImage(localResizedPath, ic)

	case <-pchan.Skipped:
		log.Printf("Skipped processing (resize handler) %s", localResizedPath)
	}

	return nil
}

func (r *Request) uploadResizedImage(localResizedPath string, ic *core.ImageConfiguration) (err error) {
	remoteResizedPath := r.Paths.RemoteImagePath(ic.Namespace, ic.ID, ic.Filename)
	err = r.Uploader.Upload(localResizedPath, remoteResizedPath, ic.ToContentType())

	if err != nil {
		log.Println("Unable to upload file", remoteResizedPath, err)
		return err
	}
	return nil
}
