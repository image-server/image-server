package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// SignPayload signs a webhook payload using HMAC-SHA256
// The signature is computed over: timestamp.body
// This matches the common pattern used by Stripe, GitHub, etc.
func SignPayload(secret string, timestamp time.Time, body []byte) string {
	ts := strconv.FormatInt(timestamp.Unix(), 10)
	message := fmt.Sprintf("%s.%s", ts, string(body))

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))

	return hex.EncodeToString(h.Sum(nil))
}

// Headers returns the signature headers to include in the webhook request
type Headers struct {
	Signature string
	Timestamp int64
}

// GenerateHeaders creates the webhook signature headers
func GenerateHeaders(secret string, body []byte) *Headers {
	timestamp := time.Now().UTC()
	signature := SignPayload(secret, timestamp, body)

	return &Headers{
		Signature: fmt.Sprintf("sha256=%s", signature),
		Timestamp: timestamp.Unix(),
	}
}

// VerifySignature verifies a webhook signature (useful for testing)
func VerifySignature(secret string, timestamp int64, body []byte, signature string) bool {
	ts := time.Unix(timestamp, 0)
	expected := fmt.Sprintf("sha256=%s", SignPayload(secret, ts, body))

	return hmac.Equal([]byte(expected), []byte(signature))
}
