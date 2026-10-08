package logger

import (
	"time"

	"github.com/image-server/image-server/core"
)

var Loggers []core.Logger

func initialize() {
	Loggers = []core.Logger{}
}

func ImagePosted() {
	for _, logger := range Loggers {
		go logger.ImagePosted()
	}
}

func ImagePostingFailed() {
	for _, logger := range Loggers {
		go logger.ImagePostingFailed()
	}
}

func ImageProcessed(ic *core.ImageConfiguration) {
	for _, logger := range Loggers {
		go logger.ImageProcessed(ic)
	}
}

func ImageAlreadyProcessed(ic *core.ImageConfiguration) {
	for _, logger := range Loggers {
		go logger.ImageAlreadyProcessed(ic)
	}
}

func ImageProcessedWithErrors(ic *core.ImageConfiguration) {
	for _, logger := range Loggers {
		go logger.ImageProcessedWithErrors(ic)
	}
}

func AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string) {
	for _, logger := range Loggers {
		go logger.AllImagesAlreadyProcessed(namespace, hash, sourceURL)
	}
}

func SourceDownloaded() {
	for _, logger := range Loggers {
		go logger.SourceDownloaded()
	}
}

func OriginalDownloaded(source string, destination string) {
	for _, logger := range Loggers {
		go logger.OriginalDownloaded(source, destination)
	}
}

func OriginalDownloadFailed(source string) {
	for _, logger := range Loggers {
		go logger.OriginalDownloadFailed(source)
	}
}

func OriginalDownloadSkipped(source string) {
	for _, logger := range Loggers {
		go logger.OriginalDownloadSkipped(source)
	}
}

func RequestLatency(handler string, since time.Time) {
	for _, logger := range Loggers {
		go logger.RequestLatency(handler, since)
	}
}

// NamespaceAdmin records a namespace rename or delete and its result. reason
// is set when the result is "rejected".
func NamespaceAdmin(op string, result string, reason string) {
	for _, logger := range Loggers {
		go logger.NamespaceAdmin(op, result, reason)
	}
}

func OriginalUploaded(props *core.ImageProperties, namespace string) {
	for _, logger := range Loggers {
		go logger.OriginalUploaded(props, namespace)
	}
}
