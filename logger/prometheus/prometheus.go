package prometheus

import (
	"fmt"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/logger"
)

// Logger prometheus logger for metrics
type Logger struct {
	metrics *Metrics
}

// Enable enables the prometheus collector
func Enable() {
	metrics := CreateAndRegisterMetrics()
	l := &Logger{
		metrics: metrics,
	}
	logger.Loggers = append(logger.Loggers, l)
}

// ImagePosted posts an image posted metric
func (l *Logger) ImagePosted() {
	l.metrics.imagePostedMetric.Inc()
}

// ImagePostingFailed posts an image posting failed metric
func (l *Logger) ImagePostingFailed() {
	l.metrics.imagePostingFailedMetric.Inc()
}

// ImageProcessed posts an image processed metric
func (l *Logger) ImageProcessed(ic *core.ImageConfiguration) {
	l.metrics.imageProcessedMetric.WithLabelValues(ic.Namespace, ic.Format, fmt.Sprint(ic.Quality)).Inc()
}

// ImageAlreadyProcessed posts an image already processed metric
func (l *Logger) ImageAlreadyProcessed(ic *core.ImageConfiguration) {
	l.metrics.imageAlreadyProcessedMetric.WithLabelValues(ic.Namespace, ic.Format, fmt.Sprint(ic.Quality)).Inc()
}

// ImageProcessedWithErrors posts an image processed with errors metric
func (l *Logger) ImageProcessedWithErrors(ic *core.ImageConfiguration) {
	l.metrics.imageProcessedWithErrorsMetric.WithLabelValues(ic.Namespace, ic.Format, fmt.Sprint(ic.Quality)).Inc()
}

// AllImagesAlreadyProcessed posts an all images already processed metric
func (l *Logger) AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string) {
	l.metrics.allImagesAlreadyProcessedMetric.WithLabelValues(namespace).Inc()
}

// SourceDownloaded posts an source downloaded metric
func (l *Logger) SourceDownloaded() {
	l.metrics.sourceDownloadedMetric.Inc()
}

// OriginalDownloaded posts an original downloaded metric
func (l *Logger) OriginalDownloaded(source string, destination string) {
	l.metrics.originalDownloadedMetric.Inc()
}

// OriginalDownloadFailed posts an original download failed metric
func (l *Logger) OriginalDownloadFailed(source string) {
	l.metrics.originalDownloadFailedMetric.Inc()
}

// OriginalDownloadSkipped posts an original download skipped metric
func (l *Logger) OriginalDownloadSkipped(source string) {
	l.metrics.originalDownloadSkippedMetric.Inc()
}

// RequestLatency adds the latency for a request
func (l *Logger) RequestLatency(handler string, since time.Time) {
	l.metrics.requestLatency.WithLabelValues(handler).Observe(time.Since(since).Seconds())
}

// OriginalUploaded is called when an original image is uploaded
func (l *Logger) OriginalUploaded(props *core.ImageProperties, namespace string) {
	// Prometheus doesn't need to track individual uploads
	// Could add a counter here if desired
}
