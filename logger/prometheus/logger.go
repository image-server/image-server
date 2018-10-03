package prometheus

import (
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
	l.metrics.imagePostedMetric.WithLabelValues().Inc()
}

// ImagePostingFailed posts an image posting failed metric
func (l *Logger) ImagePostingFailed() {
	l.metrics.imagePostingFailedMetric.WithLabelValues().Inc()
}

// ImageProcessed posts an image processed metric
func (l *Logger) ImageProcessed(ic *core.ImageConfiguration) {
	l.metrics.imageProcessedMetric.WithLabelValues(ic.Format).Inc()
}

// ImageAlreadyProcessed posts an image already processed metric
func (l *Logger) ImageAlreadyProcessed(ic *core.ImageConfiguration) {
	l.metrics.imageAlreadyProcessedMetric.WithLabelValues(ic.Format).Inc()
}

// ImageProcessedWithErrors posts an image processed with errors metric
func (l *Logger) ImageProcessedWithErrors(ic *core.ImageConfiguration) {
	l.metrics.imageProcessedWithErrorsMetric.WithLabelValues(ic.Format).Inc()
}

// AllImagesAlreadyProcessed posts an all images already processed metric
func (l *Logger) AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string) {
	l.metrics.allImagesAlreadyProcessedMetric.WithLabelValues(namespace).Inc()
}

// SourceDownloaded posts an source downloaded metric
func (l *Logger) SourceDownloaded() {
	l.metrics.sourceDownloadedMetric.WithLabelValues().Inc()
}

// OriginalDownloaded posts an original downloaded metric
func (l *Logger) OriginalDownloaded(source string, destination string) {
	l.metrics.originalDownloadedMetric.WithLabelValues().Inc()
}

// OriginalDownloadFailed posts an original download failed metric
func (l *Logger) OriginalDownloadFailed(source string) {
	l.metrics.originalDownloadFailedMetric.WithLabelValues().Inc()
}

// OriginalDownloadSkipped posts an original download skipped metric
func (l *Logger) OriginalDownloadSkipped(source string) {
	l.metrics.originalDownloadSkippedMetric.WithLabelValues().Inc()
}
