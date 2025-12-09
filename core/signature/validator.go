package signature

import (
	"crypto/hmac"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Validator validates request signatures
type Validator struct {
	config *Config
}

// NewValidator creates a new signature validator with the given configuration
func NewValidator(config *Config) *Validator {
	return &Validator{config: config}
}

// ValidateRequest validates the signature on an HTTP request
// Returns nil if valid, or an Error if validation fails
func (v *Validator) ValidateRequest(r *http.Request) *Error {
	// Extract signature parameters
	query := r.URL.Query()
	signature := query.Get("X-Signature")
	expiresStr := query.Get("X-Expires")
	signedPath := query.Get("X-Path")

	// Check required parameters
	if signature == "" || expiresStr == "" {
		return ErrMissingSignature
	}

	// Parse expiration
	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return ErrInvalidExpires
	}

	// Check if signature has expired
	now := time.Now().Unix()
	if now > expires {
		return ErrSignatureExpired
	}

	// Check TTL isn't too far in the future
	if v.config.MaxTTL > 0 {
		maxExpires := now + int64(v.config.MaxTTL.Seconds())
		if expires > maxExpires {
			return ErrSignatureTTLExceeded
		}
	}

	// Use signed path if provided, otherwise use request path
	if signedPath == "" {
		signedPath = r.URL.Path
	}

	// Validate path prefix - signed path must be a prefix of the request path
	if !isPathPrefix(signedPath, r.URL.Path) {
		return ErrInvalidPath
	}

	// Build the string to sign
	stringToSign := StringToSign(r.Method, signedPath, expires)

	// Try each secret (supports rotation)
	for _, secret := range v.config.Secrets {
		expectedSig := ComputeSignatureWithSecret(stringToSign, secret)
		if hmac.Equal([]byte(signature), []byte(expectedSig)) {
			return nil // Valid signature
		}
	}

	return ErrInvalidSignature
}

// ShouldValidate returns true if the request should be validated based on configuration
func (v *Validator) ShouldValidate(r *http.Request) bool {
	if !v.config.Enabled {
		return false
	}

	// Skip health check endpoints
	if r.URL.Path == "/status_check" || strings.HasPrefix(r.URL.Path, "/probe/") {
		return false
	}

	// For GET requests, only validate if RequireForReads is enabled
	if r.Method == http.MethodGet && !v.config.RequireForReads {
		return false
	}

	return true
}

// isPathPrefix checks if signedPath is a prefix of requestPath
// Handles path normalization (trailing slashes, etc.)
func isPathPrefix(signedPath, requestPath string) bool {
	// Normalize paths - remove trailing slashes for comparison
	signedPath = strings.TrimSuffix(signedPath, "/")
	requestPath = strings.TrimSuffix(requestPath, "/")

	// Exact match
	if signedPath == requestPath {
		return true
	}

	// Check if signed path is a prefix
	// Must match at path segment boundary (e.g., /photos matches /photos/abc but not /photosxyz)
	if strings.HasPrefix(requestPath, signedPath) {
		// Check that the next character after the prefix is a slash
		if len(requestPath) > len(signedPath) && requestPath[len(signedPath)] == '/' {
			return true
		}
	}

	return false
}
