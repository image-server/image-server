package webhook

import (
	"time"

	"github.com/image-server/image-server/core"
)

// EventType represents the type of webhook event
type EventType string

const (
	EventImageUploaded        EventType = "image.uploaded"
	EventImageProcessed       EventType = "image.processed"
	EventImageProcessingFailed EventType = "image.processing_failed"
	EventBatchComplete        EventType = "image.batch_complete"
)

// Payload is the base webhook payload structure
type Payload struct {
	Event     EventType   `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// UploadedData contains data for image.uploaded events
type UploadedData struct {
	Namespace   string `json:"namespace"`
	Hash        string `json:"hash"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ContentType string `json:"content_type"`
	RemoteURL   string `json:"remote_url"`
}

// ProcessedData contains data for image.processed events
type ProcessedData struct {
	Namespace string `json:"namespace"`
	Hash      string `json:"hash"`
	Filename  string `json:"filename"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Quality   uint   `json:"quality"`
	RemoteURL string `json:"remote_url"`
}

// ProcessingFailedData contains data for image.processing_failed events
type ProcessingFailedData struct {
	Namespace string `json:"namespace"`
	Hash      string `json:"hash"`
	Filename  string `json:"filename"`
	Format    string `json:"format"`
	Error     string `json:"error,omitempty"`
}

// BatchCompleteData contains data for image.batch_complete events
type BatchCompleteData struct {
	Namespace string `json:"namespace"`
	Hash      string `json:"hash"`
	SourceURL string `json:"source_url,omitempty"`
}

// NewUploadedPayload creates a payload for image.uploaded events
func NewUploadedPayload(props *core.ImageProperties, namespace string, remoteURL string) *Payload {
	return &Payload{
		Event:     EventImageUploaded,
		Timestamp: time.Now().UTC(),
		Data: UploadedData{
			Namespace:   namespace,
			Hash:        props.Hash,
			Width:       props.Width,
			Height:      props.Height,
			ContentType: props.ContentType,
			RemoteURL:   remoteURL,
		},
	}
}

// NewProcessedPayload creates a payload for image.processed events
func NewProcessedPayload(ic *core.ImageConfiguration, remoteURL string) *Payload {
	return &Payload{
		Event:     EventImageProcessed,
		Timestamp: time.Now().UTC(),
		Data: ProcessedData{
			Namespace: ic.Namespace,
			Hash:      ic.ID,
			Filename:  ic.Filename,
			Format:    ic.Format,
			Width:     ic.Width,
			Height:    ic.Height,
			Quality:   ic.Quality,
			RemoteURL: remoteURL,
		},
	}
}

// NewProcessingFailedPayload creates a payload for image.processing_failed events
func NewProcessingFailedPayload(ic *core.ImageConfiguration, errMsg string) *Payload {
	return &Payload{
		Event:     EventImageProcessingFailed,
		Timestamp: time.Now().UTC(),
		Data: ProcessingFailedData{
			Namespace: ic.Namespace,
			Hash:      ic.ID,
			Filename:  ic.Filename,
			Format:    ic.Format,
			Error:     errMsg,
		},
	}
}

// NewBatchCompletePayload creates a payload for image.batch_complete events
func NewBatchCompletePayload(namespace, hash, sourceURL string) *Payload {
	return &Payload{
		Event:     EventBatchComplete,
		Timestamp: time.Now().UTC(),
		Data: BatchCompleteData{
			Namespace: namespace,
			Hash:      hash,
			SourceURL: sourceURL,
		},
	}
}
