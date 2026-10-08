package prometheus

import (
	"runtime"

	"github.com/image-server/image-server/core"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics is the structure that holds a reference to all the image-server metrics
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
	requestLatency                  *prometheus.HistogramVec
	namespaceAdminMetric            *prometheus.CounterVec
}

// CreateAndRegisterMetrics creates a struct of Metrics
func CreateAndRegisterMetrics() *Metrics {
	metrics := Metrics{}

	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "image_server_build_info",
			Help: "Build information",
		},
		[]string{"go_version", "version", "git_hash", "build_timestamp"},
	)
	prometheus.MustRegister(buildInfo)
	buildInfo.WithLabelValues(runtime.Version(), core.VERSION, core.GitHash, core.BuildTimestamp).Set(1)

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
		[]string{"namespace", "format", "quality"},
	)
	prometheus.MustRegister(metrics.imageProcessedMetric)

	metrics.imageAlreadyProcessedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_version_noop_total",
			Help: "Number of already processed images",
		},
		[]string{"namespace", "format", "quality"},
	)
	prometheus.MustRegister(metrics.imageAlreadyProcessedMetric)

	metrics.imageProcessedWithErrorsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_processing_version_failed_total",
			Help: "Number of failed processed images",
		},
		[]string{"namespace", "format", "quality"},
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

	metrics.requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "image_server_request_latency_seconds",
			Help: "Latency for requests made towards the api",
		},
		[]string{"handler"},
	)
	prometheus.MustRegister(metrics.requestLatency)

	metrics.namespaceAdminMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "image_server_namespace_admin_total",
			Help: "Namespace renames and deletes by op (rename, delete), result, and reason when rejected",
		},
		[]string{"op", "result", "reason"},
	)
	prometheus.MustRegister(metrics.namespaceAdminMetric)

	return &metrics
}
