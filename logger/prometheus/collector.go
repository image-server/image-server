package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ImageServerCollector the prometheus collector for the image server
type ImageServerCollector struct {
	imagePostedMetric               *prometheus.CounterVec
	imagePostingFailedMetric        *prometheus.CounterVec
	imageProcessedMetric            *prometheus.CounterVec
	imageAlreadyProcessedMetric     *prometheus.CounterVec
	imageProcessedWithErrorsMetric  *prometheus.CounterVec
	allImagesAlreadyProcessedMetric *prometheus.CounterVec
	sourceDownloadedMetric          *prometheus.CounterVec
	originalDownloadedMetric        *prometheus.CounterVec
	originalDownloadFailedMetric    *prometheus.CounterVec
	originalDownloadSkippedMetric   *prometheus.CounterVec
}

// NewImageServerCollector creates a new image server collector
func NewImageServerCollector() *ImageServerCollector {
	return &ImageServerCollector{
		imagePostedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: imagePosted.Name,
				Help: imagePosted.Description,
			},
			imagePosted.Args,
		),
		imagePostingFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: imagePostingFailed.Name,
				Help: imagePostingFailed.Description,
			},
			imagePostingFailed.Args,
		),
		imageProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: imageProcessed.Name,
				Help: imageProcessed.Description,
			},
			imageProcessed.Args,
		),
		imageAlreadyProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: imageAlreadyProcessed.Name,
				Help: imageAlreadyProcessed.Description,
			},
			imageAlreadyProcessed.Args,
		),
		imageProcessedWithErrorsMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: imageProcessedWithErrors.Name,
				Help: imageProcessedWithErrors.Description,
			},
			imageProcessedWithErrors.Args,
		),
		allImagesAlreadyProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: allImagesAlreadyProcessed.Name,
				Help: allImagesAlreadyProcessed.Description,
			},
			allImagesAlreadyProcessed.Args,
		),
		sourceDownloadedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: sourceDownloaded.Name,
				Help: sourceDownloaded.Description,
			},
			sourceDownloaded.Args,
		),
		originalDownloadedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: originalDownloaded.Name,
				Help: originalDownloaded.Description,
			},
			originalDownloaded.Args,
		),
		originalDownloadFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: originalDownloadFailed.Name,
				Help: originalDownloadFailed.Description,
			},
			originalDownloadFailed.Args,
		),
		originalDownloadSkippedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: originalDownloadSkipped.Name,
				Help: originalDownloadSkipped.Description,
			},
			originalDownloadSkipped.Args,
		),
	}
}

// Describe implements the required describe collector function for all prometheus collectors
func (collector *ImageServerCollector) Describe(ch chan<- *prometheus.Desc) {
	collector.imagePostedMetric.Describe(ch)
	collector.imagePostingFailedMetric.Describe(ch)
	collector.imageProcessedMetric.Describe(ch)
	collector.imageAlreadyProcessedMetric.Describe(ch)
	collector.imageProcessedWithErrorsMetric.Describe(ch)
	collector.allImagesAlreadyProcessedMetric.Describe(ch)
	collector.sourceDownloadedMetric.Describe(ch)
	collector.originalDownloadedMetric.Describe(ch)
	collector.originalDownloadFailedMetric.Describe(ch)
	collector.originalDownloadSkippedMetric.Describe(ch)
}

//Collect implements required collect function for all prometheus collectors
func (collector *ImageServerCollector) Collect(ch chan<- prometheus.Metric) {
	collector.imagePostedMetric.Collect(ch)
	collector.imagePostingFailedMetric.Collect(ch)
	collector.imageProcessedMetric.Collect(ch)
	collector.imageAlreadyProcessedMetric.Collect(ch)
	collector.imageProcessedWithErrorsMetric.Collect(ch)
	collector.allImagesAlreadyProcessedMetric.Collect(ch)
	collector.sourceDownloadedMetric.Collect(ch)
	collector.originalDownloadedMetric.Collect(ch)
	collector.originalDownloadFailedMetric.Collect(ch)
	collector.originalDownloadSkippedMetric.Collect(ch)
}
