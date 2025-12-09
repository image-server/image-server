package signature

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestValidateRequest_ValidSignature(t *testing.T) {
	secret := "test-secret-key"
	config := &Config{
		Enabled: true,
		Secrets: []string{secret},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)
	signer := NewSigner(secret, "")

	expires := time.Now().Add(15 * time.Minute).Unix()
	path := "/photos"
	sig := signer.ComputeSignature("POST", path, expires)

	req := httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("Expected valid signature, got error: %v", err)
	}
}

func TestValidateRequest_ExpiredSignature(t *testing.T) {
	secret := "test-secret-key"
	config := &Config{
		Enabled: true,
		Secrets: []string{secret},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)
	signer := NewSigner(secret, "")

	// Expired 5 minutes ago
	expires := time.Now().Add(-5 * time.Minute).Unix()
	path := "/photos"
	sig := signer.ComputeSignature("POST", path, expires)

	req := httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != ErrSignatureExpired {
		t.Errorf("Expected ErrSignatureExpired, got: %v", err)
	}
}

func TestValidateRequest_InvalidSignature(t *testing.T) {
	config := &Config{
		Enabled: true,
		Secrets: []string{"correct-secret"},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)

	// Sign with wrong secret
	wrongSigner := NewSigner("wrong-secret", "")
	expires := time.Now().Add(15 * time.Minute).Unix()
	path := "/photos"
	sig := wrongSigner.ComputeSignature("POST", path, expires)

	req := httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got: %v", err)
	}
}

func TestValidateRequest_MissingSignature(t *testing.T) {
	config := &Config{
		Enabled: true,
		Secrets: []string{"secret"},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)

	req := httptest.NewRequest("POST", "/photos", nil)

	err := validator.ValidateRequest(req)
	if err != ErrMissingSignature {
		t.Errorf("Expected ErrMissingSignature, got: %v", err)
	}
}

func TestValidateRequest_MissingExpires(t *testing.T) {
	config := &Config{
		Enabled: true,
		Secrets: []string{"secret"},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)

	req := httptest.NewRequest("POST", "/photos?X-Signature=abc123", nil)

	err := validator.ValidateRequest(req)
	if err != ErrMissingSignature {
		t.Errorf("Expected ErrMissingSignature, got: %v", err)
	}
}

func TestValidateRequest_TTLExceeded(t *testing.T) {
	secret := "test-secret-key"
	config := &Config{
		Enabled: true,
		Secrets: []string{secret},
		MaxTTL:  time.Hour, // Max 1 hour
	}
	validator := NewValidator(config)
	signer := NewSigner(secret, "")

	// Expires in 2 hours (exceeds MaxTTL)
	expires := time.Now().Add(2 * time.Hour).Unix()
	path := "/photos"
	sig := signer.ComputeSignature("POST", path, expires)

	req := httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != ErrSignatureTTLExceeded {
		t.Errorf("Expected ErrSignatureTTLExceeded, got: %v", err)
	}
}

func TestValidateRequest_PathPrefix(t *testing.T) {
	secret := "test-secret-key"
	config := &Config{
		Enabled: true,
		Secrets: []string{secret},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)
	signer := NewSigner(secret, "")

	expires := time.Now().Add(15 * time.Minute).Unix()

	// Sign for /photos but access /photos/abc/def/ghi/jkl/image.jpg
	signedPath := "/photos"
	sig := signer.ComputeSignature("POST", signedPath, expires)

	req := httptest.NewRequest("POST", "/photos/abc/def/ghi/jkl/image.jpg?X-Expires="+itoa(expires)+"&X-Path="+signedPath+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("Expected valid signature for path prefix, got error: %v", err)
	}
}

func TestValidateRequest_PathPrefixMismatch(t *testing.T) {
	secret := "test-secret-key"
	config := &Config{
		Enabled: true,
		Secrets: []string{secret},
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)
	signer := NewSigner(secret, "")

	expires := time.Now().Add(15 * time.Minute).Unix()

	// Sign for /photos but try to access /other
	signedPath := "/photos"
	sig := signer.ComputeSignature("POST", signedPath, expires)

	req := httptest.NewRequest("POST", "/other/abc/def?X-Expires="+itoa(expires)+"&X-Path="+signedPath+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath, got: %v", err)
	}
}

func TestValidateRequest_SecretRotation(t *testing.T) {
	oldSecret := "old-secret"
	newSecret := "new-secret"

	config := &Config{
		Enabled: true,
		Secrets: []string{newSecret, oldSecret}, // Both secrets valid
		MaxTTL:  time.Hour,
	}
	validator := NewValidator(config)

	expires := time.Now().Add(15 * time.Minute).Unix()
	path := "/photos"

	// Sign with old secret (should still work)
	oldSigner := NewSigner(oldSecret, "")
	sig := oldSigner.ComputeSignature("POST", path, expires)

	req := httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err := validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("Expected old secret to still be valid, got error: %v", err)
	}

	// Sign with new secret (should also work)
	newSigner := NewSigner(newSecret, "")
	sig = newSigner.ComputeSignature("POST", path, expires)

	req = httptest.NewRequest("POST", "/photos?X-Expires="+itoa(expires)+"&X-Path="+path+"&X-Signature="+sig, nil)

	err = validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("Expected new secret to be valid, got error: %v", err)
	}
}

func TestShouldValidate(t *testing.T) {
	tests := []struct {
		name            string
		enabled         bool
		requireForReads bool
		method          string
		path            string
		expected        bool
	}{
		{"disabled", false, false, "POST", "/photos", false},
		{"POST when enabled", true, false, "POST", "/photos", true},
		{"GET when reads not required", true, false, "GET", "/photos/abc/def/ghi/jkl/image.jpg", false},
		{"GET when reads required", true, true, "GET", "/photos/abc/def/ghi/jkl/image.jpg", true},
		{"status_check skipped", true, false, "GET", "/status_check", false},
		{"probe endpoint skipped", true, true, "GET", "/probe/ready", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Enabled:         tt.enabled,
				RequireForReads: tt.requireForReads,
				Secrets:         []string{"secret"},
			}
			validator := NewValidator(config)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := validator.ShouldValidate(req)

			if result != tt.expected {
				t.Errorf("ShouldValidate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsPathPrefix(t *testing.T) {
	tests := []struct {
		signedPath  string
		requestPath string
		expected    bool
	}{
		{"/photos", "/photos", true},
		{"/photos", "/photos/", true},
		{"/photos/", "/photos", true},
		{"/photos", "/photos/abc", true},
		{"/photos", "/photos/abc/def/ghi", true},
		{"/photos", "/photosxyz", false}, // Not a path boundary
		{"/photos", "/other", false},
		{"/photos/abc", "/photos", false},
		{"/photos/abc/def/ghi/jkl", "/photos/abc/def/ghi/jkl/image.jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.signedPath+"->"+tt.requestPath, func(t *testing.T) {
			result := isPathPrefix(tt.signedPath, tt.requestPath)
			if result != tt.expected {
				t.Errorf("isPathPrefix(%q, %q) = %v, expected %v", tt.signedPath, tt.requestPath, result, tt.expected)
			}
		})
	}
}

func TestSigner_SignURL(t *testing.T) {
	signer := NewSigner("test-secret", "https://images.example.com")

	url := signer.SignURL("POST", "/photos", 15*time.Minute)

	// Check URL contains expected components
	if url == "" {
		t.Error("SignURL returned empty string")
	}

	// URL should start with base URL
	expected := "https://images.example.com/photos?"
	if len(url) < len(expected) || url[:len(expected)] != expected {
		t.Errorf("URL doesn't start with expected prefix: %s", url)
	}
}

// Helper to convert int64 to string
func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
