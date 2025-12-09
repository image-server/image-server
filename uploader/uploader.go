package uploader

import (
	"errors"
	"log"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/uploader/noop"
	"github.com/image-server/image-server/uploader/s3"
)

type Uploader struct {
	BaseDir  string
	Uploader core.Uploader
}

// DefaultUploader returns an uploader that uses an adapter based on the configuration
// on the server configuration.
func DefaultUploader(sc *core.ServerConfiguration) *Uploader {
	u := &Uploader{
		BaseDir:  sc.RemoteBasePath,
		Uploader: &s3.Uploader{},
	}

	if sc.UploaderIsAws() {
		u.Uploader = &s3.Uploader{}
	} else {
		u.Uploader = &noop.Uploader{}
	}
	return u
}

func (u *Uploader) Upload(source string, destination string, contType string) error {
	start := time.Now()

	if contType == "" {
		return errors.New("uploader.Upload is missing content type: " + source)
	}

	err := u.Uploader.Upload(source, destination, contType)
	elapsed := time.Since(start)
	log.Printf("Took %s to upload image: %s", elapsed, destination)
	return err
}

func (u *Uploader) ListDirectory(directory string) ([]string, error) {
	return u.Uploader.ListDirectory(directory)
}

func (u *Uploader) CreateDirectory(path string) error {
	start := time.Now()
	directoryPath := u.Uploader.CreateDirectory(path)
	elapsed := time.Since(start)
	log.Printf("Took %s to generate remote directory: %s", elapsed, path)
	return directoryPath
}

func Initialize(sc *core.ServerConfiguration) error {
	if sc.UploaderIsAws() {
		s3.Initialize(sc.AWSBucket, sc.AWSRegion)
	}
	return nil
}
