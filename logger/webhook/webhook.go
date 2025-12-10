package webhook

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/logger"
)

const (
	maxRetries     = 3
	headerSignature = "X-Webhook-Signature"
	headerTimestamp = "X-Webhook-Timestamp"
	userAgent       = "image-server-webhook/1.0"
)

// Config holds the webhook configuration
type Config struct {
	URL     string
	Secret  string
	Timeout time.Duration
	Events  map[EventType]bool
}

// Logger implements core.Logger for webhook notifications
type Logger struct {
	config *Config
	client *http.Client
	paths  core.Paths
}

// Enable registers the webhook logger if a URL is configured
func Enable(cfg *Config, paths core.Paths) {
	if cfg == nil || cfg.URL == "" {
		return
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	// Default to all events if none specified
	if cfg.Events == nil || len(cfg.Events) == 0 {
		cfg.Events = map[EventType]bool{
			EventImageUploaded:         true,
			EventImageProcessed:        true,
			EventImageProcessingFailed: true,
			EventBatchComplete:         true,
		}
	}

	l := &Logger{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		paths:  paths,
	}

	logger.Loggers = append(logger.Loggers, l)
	log.Printf("Webhook logger enabled: url=%s, events=%v", cfg.URL, cfg.Events)
}

// send dispatches a webhook payload with retries
func (l *Logger) send(payload *Payload) {
	if !l.config.Events[payload.Event] {
		return
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("webhook: failed to marshal payload: %v", err)
			return
		}

		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				// Exponential backoff: 1s, 4s
				backoff := time.Duration(attempt*attempt) * time.Second
				time.Sleep(backoff)
			}

			if err := l.doSend(body); err != nil {
				lastErr = err
				log.Printf("webhook: attempt %d failed: %v", attempt+1, err)
				continue
			}
			return // success
		}

		log.Printf("webhook: failed after %d attempts: %v", maxRetries, lastErr)
	}()
}

// doSend performs a single webhook HTTP request
func (l *Logger) doSend(body []byte) error {
	req, err := http.NewRequest(http.MethodPost, l.config.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	// Add signature headers if secret is configured
	if l.config.Secret != "" {
		headers := GenerateHeaders(l.config.Secret, body)
		req.Header.Set(headerSignature, headers.Signature)
		req.Header.Set(headerTimestamp, strconv.FormatInt(headers.Timestamp, 10))
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return &WebhookError{StatusCode: resp.StatusCode}
}

// WebhookError represents a non-2xx response from the webhook endpoint
type WebhookError struct {
	StatusCode int
}

func (e *WebhookError) Error() string {
	return "webhook returned status " + strconv.Itoa(e.StatusCode)
}

// --- core.Logger interface implementation ---

// ImagePosted is called when an image upload request is received
// We don't send a webhook here because we don't have image details yet
func (l *Logger) ImagePosted() {
	// No-op: wait for OriginalUploaded which has full image details
}

// ImagePostingFailed is called when an image upload fails
func (l *Logger) ImagePostingFailed() {
	// Could potentially add a generic upload_failed event
}

// ImageProcessed is called when a single image variant is processed
func (l *Logger) ImageProcessed(ic *core.ImageConfiguration) {
	remoteURL := l.paths.RemoteImageURL(ic.Namespace, ic.ID, ic.Filename)
	payload := NewProcessedPayload(ic, remoteURL)
	l.send(payload)
}

// ImageAlreadyProcessed is called when an image variant already exists
func (l *Logger) ImageAlreadyProcessed(ic *core.ImageConfiguration) {
	// Don't send webhook for cached/existing images
}

// ImageProcessedWithErrors is called when image processing fails
func (l *Logger) ImageProcessedWithErrors(ic *core.ImageConfiguration) {
	payload := NewProcessingFailedPayload(ic, "processing failed")
	l.send(payload)
}

// AllImagesAlreadyProcessed is called when all variants already exist
func (l *Logger) AllImagesAlreadyProcessed(namespace string, hash string, sourceURL string) {
	payload := NewBatchCompletePayload(namespace, hash, sourceURL)
	l.send(payload)
}

// SourceDownloaded is called when source image is downloaded
func (l *Logger) SourceDownloaded() {
	// No-op
}

// OriginalDownloaded is called when original image is downloaded from remote
func (l *Logger) OriginalDownloaded(source string, destination string) {
	// No-op
}

// OriginalDownloadFailed is called when original download fails
func (l *Logger) OriginalDownloadFailed(source string) {
	// No-op
}

// OriginalDownloadSkipped is called when original download is skipped
func (l *Logger) OriginalDownloadSkipped(source string) {
	// No-op
}

// RequestLatency is called to record request latency
func (l *Logger) RequestLatency(handler string, since time.Time) {
	// No-op: latency metrics don't need webhooks
}

// OriginalUploaded is called when the original image is uploaded to storage
// This is a new method added to support webhooks with full image details
func (l *Logger) OriginalUploaded(props *core.ImageProperties, namespace string) {
	remoteURL := l.paths.RemoteOriginalURL(namespace, props.Hash)
	payload := NewUploadedPayload(props, namespace, remoteURL)
	l.send(payload)
}
