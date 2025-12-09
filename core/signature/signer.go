package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

// Signer generates signed URLs for the image server
type Signer struct {
	Secret  string
	BaseURL string
}

// NewSigner creates a new Signer with the given secret and base URL
func NewSigner(secret, baseURL string) *Signer {
	return &Signer{
		Secret:  secret,
		BaseURL: baseURL,
	}
}

// SignURL generates a signed URL for the given method and path with the specified TTL
func (s *Signer) SignURL(method, path string, ttl time.Duration) string {
	expires := time.Now().Add(ttl).Unix()
	return s.SignURLWithExpires(method, path, expires)
}

// SignURLWithExpires generates a signed URL with a specific expiration timestamp
func (s *Signer) SignURLWithExpires(method, path string, expiresUnix int64) string {
	signature := s.ComputeSignature(method, path, expiresUnix)

	return fmt.Sprintf("%s%s?X-Expires=%d&X-Path=%s&X-Signature=%s",
		s.BaseURL,
		path,
		expiresUnix,
		url.QueryEscape(path),
		url.QueryEscape(signature),
	)
}

// ComputeSignature computes the HMAC-SHA256 signature for the given parameters
func (s *Signer) ComputeSignature(method, path string, expiresUnix int64) string {
	stringToSign := fmt.Sprintf("%s\n%s\n%d", method, path, expiresUnix)
	return ComputeSignatureWithSecret(stringToSign, s.Secret)
}

// ComputeSignatureWithSecret computes an HMAC-SHA256 signature using the given secret
func ComputeSignatureWithSecret(stringToSign, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// StringToSign builds the canonical string to sign from request components
func StringToSign(method, path string, expires int64) string {
	return fmt.Sprintf("%s\n%s\n%d", method, path, expires)
}
