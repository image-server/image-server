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
				Name: "image_server_new_image_request_total",
				Help: "Number of requested images",
			},
			nil,
		),
		imagePostingFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_new_image_request_failed_total",
				Help: "Number of failed requested images",
			},
			nil,
		),
		imageProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_processing_version_ok_total",
				Help: "Number of processed images",
			},
			[]string{"ic_format"},
		),
		imageAlreadyProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_processing_version_noop_total",
				Help: "Number of already processed images",
			},
			[]string{"ic_format"},
		),
		imageProcessedWithErrorsMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_processing_version_failed_total",
				Help: "Number of failed processed images",
			},
			[]string{"ic_format"},
		),
		allImagesAlreadyProcessedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_processing_versions_noop_total",
				Help: "Number of already processed all images",
			},
			[]string{"namespace"},
		),
		sourceDownloadedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_fetch_source_downloaded_total",
				Help: "Number of downloaded source images",
			},
			nil,
		),
		originalDownloadedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_fetch_original_downloaded_total",
				Help: "Number of downloaded original images",
			},
			nil,
		),
		originalDownloadFailedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_fetch_original_unavailable_total",
				Help: "Number of unavailable downloaded original images",
			},
			nil,
		),
		originalDownloadSkippedMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_server_fetch_original_download_skipped_total",
				Help: "Number of skipped downloaded original images",
			},
			nil,
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
