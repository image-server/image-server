package statsd

import (
	"fmt"
	"log"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/logger"
	"github.com/quipo/statsd"
)

type Logger struct {
	Host   string
	Port   int
	Prefix string
	statsd *statsd.StatsdBuffer
}

func Enable(host string, port int, prefix string) {
	l := &Logger{Host: host, Port: port, Prefix: prefix}
	l.initializeStatsd()

	logger.Loggers = append(logger.Loggers, l)
}

func (l *Logger) ImagePosted() {
	l.track("new_image.request")
}

func (l *Logger) ImagePostingFailed() {
	l.track("new_image.request_failed")
}

func (l *Logger) ImageProcessed(ic *core.ImageConfiguration) {
	l.track("processing.version.ok")
	l.track("processing.version.ok." + ic.Format)
}

func (l *Logger) ImageAlreadyProcessed(ic *core.ImageConfiguration) {
	l.track("processing.version.noop")
	l.track("processing.version.noop." + ic.Format)
}

func (l *Logger) ImageProcessedWithErrors(ic *core.ImageConfiguration) {
	l.track("processing.version.failed")
	l.track("processing.version.failed." + ic.Format)
}

func (l *Logger) AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string) {
	l.track("processing.versions.noop")
}

func (l *Logger) SourceDownloaded() {
	l.track("fetch.source_downloaded")
}

func (l *Logger) OriginalDownloaded(source string, destination string) {
	l.track("fetch.original_downloaded")
}

func (l *Logger) OriginalDownloadFailed(source string) {
	l.track("fetch.original_unavailable")
}

func (l *Logger) OriginalDownloadSkipped(source string) {
	l.track("fetch.original_download_skipped")
}

func (l *Logger) RequestLatency(handler string, since time.Time) {
	l.statsd.Timing(fmt.Sprintf("%s.request_latency", handler), int64(time.Since(since).Seconds()))
}

func (l *Logger) OriginalUploaded(props *core.ImageProperties, namespace string) {
	l.track("upload.original")
}

func (l *Logger) track(name string) {
	metric := fmt.Sprintf("%s_count", name)
	l.statsd.Incr(metric, 1)
}

func (l *Logger) initializeStatsd() {
	server := fmt.Sprintf("%v:%v", l.Host, l.Port)
	statsdclient := statsd.NewStatsdClient(server, l.Prefix)
	statsdclient.CreateSocket()
	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	l.statsd = statsd.NewStatsdBuffer(interval, statsdclient)
	// defer stats.Close()

	log.Println("Loaded Statsd connection:", server)
}
