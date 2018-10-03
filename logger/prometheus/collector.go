package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ImageServerCollector the prometheus collector for the image server
type Metrics struct {
	imagePostedMetric               prometheus.Counter
	imagePostingFailedMetric        prometheus.Counter
	imageProcessedMetric            *prometheus.CounterVec
	imageAlreadyProcessedMetric     *prometheus.CounterVec
	imageProcessedWithErrorsMetric  *prometheus.CounterVec
	allImagesAlreadyProcessedMetric *prometheus.CounterVec
	sourceDownloadedMetric          prometheus.Counter
	originalDownloadedMetric        prometheus.Counter
	originalDownloadFailedMetric    prometheus.Counter
	originalDownloadSkippedMetric   prometheus.Counter
}

// CreateAndRegisterMetrics creates a struct of Prometheus Metrics
func CreateAndRegisterMetrics() *Metrics {
	metrics := Metrics{}

	metrics.imagePostedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_new_image_request_total",
			Help: "Number of requested images",
		},
	)
	prometheus.MustRegister(metrics.imagePostedMetric)

	metrics.imagePostingFailedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_new_image_request_failed_total",
			Help: "Number of failed requested images",
		},
	)
	prometheus.MustRegister(metrics.imagePostingFailedMetric)

	metrics.imageProcessedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_version_ok_total",
			Help: "Number of processed images",
		},
		[]string{"ic_format"},
	)
	prometheus.MustRegister(metrics.imageProcessedMetric)

	metrics.imageAlreadyProcessedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_version_noop_total",
			Help: "Number of already processed images",
		},
		[]string{"ic_format"},
	)
	prometheus.MustRegister(metrics.imageAlreadyProcessedMetric)

	metrics.imageProcessedWithErrorsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_version_failed_total",
			Help: "Number of failed processed images",
		},
		[]string{"ic_format"},
	)
	prometheus.MustRegister(metrics.imageProcessedWithErrorsMetric)

	metrics.allImagesAlreadyProcessedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_versions_noop_total",
			Help: "Number of already processed all images",
		},
		[]string{"namespace"},
	)
	prometheus.MustRegister(metrics.allImagesAlreadyProcessedMetric)

	metrics.sourceDownloadedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_fetch_source_downloaded_total",
			Help: "Number of downloaded source images",
		},
	)
	prometheus.MustRegister(metrics.sourceDownloadedMetric)

	metrics.originalDownloadedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_fetch_original_downloaded_total",
			Help: "Number of downloaded original images",
		},
	)
	prometheus.MustRegister(metrics.originalDownloadedMetric)

	metrics.originalDownloadFailedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_fetch_original_unavailable_total",
			Help: "Number of unavailable downloaded original images",
		},
	)
	prometheus.MustRegister(metrics.originalDownloadFailedMetric)

	metrics.originalDownloadSkippedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "image_server_fetch_original_download_skipped_total",
			Help: "Number of skipped downloaded original images",
		},
	)
	prometheus.MustRegister(metrics.originalDownloadSkippedMetric)

	return &metrics
}
