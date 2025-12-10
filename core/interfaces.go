package core

import (
	"time"
)

type Adapters struct {
	Fetcher Fetcher
	Paths   Paths
}

type Fetcher interface {
	Fetch(string, string) error
}

type Logger interface {
	ImagePosted()
	ImagePostingFailed()
	ImageProcessed(ic *ImageConfiguration)
	ImageAlreadyProcessed(ic *ImageConfiguration)
	ImageProcessedWithErrors(ic *ImageConfiguration)
	AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string)
	SourceDownloaded()
	OriginalDownloaded(source string, destination string)
	OriginalDownloadFailed(source string)
	OriginalDownloadSkipped(source string)
	RequestLatency(handler string, since time.Time)
	OriginalUploaded(props *ImageProperties, namespace string)
}

// Processor
type Processor interface {
	CreateImage() error
}

// Paths
type Paths interface {
	LocalInfoPath(string, string) string
	RemoteInfoPath(string, string) string
	TempImagePath(string) string
	RandomTempPath() string
	LocalOriginalPath(string, string) string
	LocalImagePath(namespace string, md5 string, imageName string) string
	LocalImageDirectory(namespace string, md5 string) string
	RemoteImageDirectory(namespace string, md5 string) string
	RemoteOriginalPath(string, string) string
	RemoteOriginalURL(string, string) string
	RemoteImagePath(namespace string, md5 string, imageName string) string
	RemoteImageURL(namespace string, md5 string, imageName string) string
}

// SourceMapper
type SourceMapper interface {
	RemoteImageURL(*ImageConfiguration) string
}

// Uploader
type Uploader interface {
	CreateDirectory(string) error
	Upload(string, string, string) error
	ListDirectory(string) ([]string, error)
}
