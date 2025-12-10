package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/image-server/image-server/core"
)

// mockPaths implements core.Paths for testing
type mockPaths struct{}

func (m *mockPaths) LocalInfoPath(namespace, hash string) string {
	return "/local/info/" + namespace + "/" + hash + ".json"
}
func (m *mockPaths) RemoteInfoPath(namespace, hash string) string {
	return "/remote/info/" + namespace + "/" + hash + ".json"
}
func (m *mockPaths) TempImagePath(s string) string                        { return "/tmp/" + s }
func (m *mockPaths) RandomTempPath() string                               { return "/tmp/random" }
func (m *mockPaths) LocalOriginalPath(namespace, hash string) string      { return "/local/" + namespace + "/" + hash + "/original" }
func (m *mockPaths) LocalImagePath(namespace, md5, imageName string) string {
	return "/local/" + namespace + "/" + md5 + "/" + imageName
}
func (m *mockPaths) LocalImageDirectory(namespace, md5 string) string {
	return "/local/" + namespace + "/" + md5
}
func (m *mockPaths) RemoteImageDirectory(namespace, md5 string) string {
	return "/remote/" + namespace + "/" + md5
}
func (m *mockPaths) RemoteOriginalPath(namespace, hash string) string {
	return "/remote/" + namespace + "/" + hash + "/original"
}
func (m *mockPaths) RemoteOriginalURL(namespace, hash string) string {
	return "https://cdn.example.com/" + namespace + "/" + hash + "/original"
}
func (m *mockPaths) RemoteImagePath(namespace, md5, imageName string) string {
	return "/remote/" + namespace + "/" + md5 + "/" + imageName
}
func (m *mockPaths) RemoteImageURL(namespace, md5, imageName string) string {
	return "https://cdn.example.com/" + namespace + "/" + md5 + "/" + imageName
}

func TestSignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"image.uploaded"}`)
	timestamp := time.Unix(1702135800, 0)

	sig := SignPayload(secret, timestamp, body)

	// Verify the signature is consistent
	sig2 := SignPayload(secret, timestamp, body)
	if sig != sig2 {
		t.Error("Signature should be deterministic")
	}

	// Verify different secret produces different signature
	sig3 := SignPayload("different-secret", timestamp, body)
	if sig == sig3 {
		t.Error("Different secret should produce different signature")
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"image.uploaded"}`)

	headers := GenerateHeaders(secret, body)

	if !VerifySignature(secret, headers.Timestamp, body, headers.Signature) {
		t.Error("Valid signature should verify")
	}

	if VerifySignature("wrong-secret", headers.Timestamp, body, headers.Signature) {
		t.Error("Invalid secret should not verify")
	}

	if VerifySignature(secret, headers.Timestamp, []byte("tampered"), headers.Signature) {
		t.Error("Tampered body should not verify")
	}
}

func TestWebhookPayloads(t *testing.T) {
	t.Run("UploadedPayload", func(t *testing.T) {
		props := &core.ImageProperties{
			Hash:        "abc123",
			Width:       1920,
			Height:      1080,
			ContentType: "image/jpeg",
		}
		payload := NewUploadedPayload(props, "avatars", "https://cdn.example.com/avatars/abc123/original")

		if payload.Event != EventImageUploaded {
			t.Errorf("Expected event %s, got %s", EventImageUploaded, payload.Event)
		}

		data := payload.Data.(UploadedData)
		if data.Namespace != "avatars" {
			t.Errorf("Expected namespace avatars, got %s", data.Namespace)
		}
		if data.Hash != "abc123" {
			t.Errorf("Expected hash abc123, got %s", data.Hash)
		}
		if data.Width != 1920 {
			t.Errorf("Expected width 1920, got %d", data.Width)
		}
	})

	t.Run("ProcessedPayload", func(t *testing.T) {
		ic := &core.ImageConfiguration{
			ID:        "abc123",
			Namespace: "avatars",
			Filename:  "x300.webp",
			Format:    "webp",
			Width:     300,
			Height:    169,
			Quality:   75,
		}
		payload := NewProcessedPayload(ic, "https://cdn.example.com/avatars/abc123/x300.webp")

		if payload.Event != EventImageProcessed {
			t.Errorf("Expected event %s, got %s", EventImageProcessed, payload.Event)
		}

		data := payload.Data.(ProcessedData)
		if data.Filename != "x300.webp" {
			t.Errorf("Expected filename x300.webp, got %s", data.Filename)
		}
	})

	t.Run("BatchCompletePayload", func(t *testing.T) {
		payload := NewBatchCompletePayload("avatars", "abc123", "https://source.example.com/image.jpg")

		if payload.Event != EventBatchComplete {
			t.Errorf("Expected event %s, got %s", EventBatchComplete, payload.Event)
		}

		data := payload.Data.(BatchCompleteData)
		if data.SourceURL != "https://source.example.com/image.jpg" {
			t.Errorf("Expected source URL, got %s", data.SourceURL)
		}
	})
}

func TestWebhookDelivery(t *testing.T) {
	var mu sync.Mutex
	var receivedPayloads []Payload
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		receivedHeaders = r.Header.Clone()

		body, _ := io.ReadAll(r.Body)
		var payload Payload
		json.Unmarshal(body, &payload)
		receivedPayloads = append(receivedPayloads, payload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &Logger{
		config: &Config{
			URL:     server.URL,
			Secret:  "test-secret",
			Timeout: 5 * time.Second,
			Events: map[EventType]bool{
				EventImageUploaded:  true,
				EventImageProcessed: true,
			},
		},
		client: &http.Client{Timeout: 5 * time.Second},
		paths:  &mockPaths{},
	}

	// Test OriginalUploaded
	props := &core.ImageProperties{
		Hash:        "abc123",
		Width:       1920,
		Height:      1080,
		ContentType: "image/jpeg",
	}
	logger.OriginalUploaded(props, "avatars")

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(receivedPayloads) != 1 {
		t.Errorf("Expected 1 payload, got %d", len(receivedPayloads))
	}

	if receivedHeaders.Get(headerSignature) == "" {
		t.Error("Expected signature header")
	}
	if receivedHeaders.Get(headerTimestamp) == "" {
		t.Error("Expected timestamp header")
	}
	mu.Unlock()
}

func TestWebhookEventFiltering(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Only enable uploaded events
	logger := &Logger{
		config: &Config{
			URL:     server.URL,
			Timeout: 5 * time.Second,
			Events: map[EventType]bool{
				EventImageUploaded: true,
				// EventImageProcessed is NOT enabled
			},
		},
		client: &http.Client{Timeout: 5 * time.Second},
		paths:  &mockPaths{},
	}

	// This should send a webhook
	logger.OriginalUploaded(&core.ImageProperties{Hash: "abc123"}, "test")

	// This should NOT send a webhook (event not enabled)
	logger.ImageProcessed(&core.ImageConfiguration{ID: "abc123", Namespace: "test", Filename: "x300.jpg"})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if requestCount != 1 {
		t.Errorf("Expected 1 request (only uploaded), got %d", requestCount)
	}
	mu.Unlock()
}

func TestWebhookRetry(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()

		// Fail first 2 requests, succeed on 3rd
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &Logger{
		config: &Config{
			URL:     server.URL,
			Timeout: 5 * time.Second,
			Events: map[EventType]bool{
				EventImageUploaded: true,
			},
		},
		client: &http.Client{Timeout: 5 * time.Second},
		paths:  &mockPaths{},
	}

	logger.OriginalUploaded(&core.ImageProperties{Hash: "abc123"}, "test")

	// Wait for retries (1s + 4s backoff + processing time)
	time.Sleep(6 * time.Second)

	mu.Lock()
	if requestCount != 3 {
		t.Errorf("Expected 3 requests (2 failures + 1 success), got %d", requestCount)
	}
	mu.Unlock()
}

func TestWebhookNoSecret(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &Logger{
		config: &Config{
			URL:     server.URL,
			Secret:  "", // No secret
			Timeout: 5 * time.Second,
			Events: map[EventType]bool{
				EventImageUploaded: true,
			},
		},
		client: &http.Client{Timeout: 5 * time.Second},
		paths:  &mockPaths{},
	}

	logger.OriginalUploaded(&core.ImageProperties{Hash: "abc123"}, "test")

	time.Sleep(100 * time.Millisecond)

	if receivedHeaders.Get(headerSignature) != "" {
		t.Error("Should not have signature header when no secret configured")
	}
}
