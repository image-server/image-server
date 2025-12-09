package signature

import (
	"encoding/json"
	"net/http"
)

// Error codes for signature validation failures
const (
	ErrCodeMissingSignature  = "MissingSignature"
	ErrCodeInvalidSignature  = "InvalidSignature"
	ErrCodeSignatureExpired  = "SignatureExpired"
	ErrCodeSignatureTTL      = "SignatureTTLExceeded"
	ErrCodeInvalidPath       = "InvalidPath"
	ErrCodeInvalidExpires    = "InvalidExpires"
)

// Error represents a signature validation error
type Error struct {
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// WriteError writes a signature error as JSON to the response
func WriteError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(err)
}

// Predefined errors
var (
	ErrMissingSignature = &Error{
		Code:    ErrCodeMissingSignature,
		Message: "Request must include X-Signature and X-Expires parameters",
	}

	ErrInvalidSignature = &Error{
		Code:    ErrCodeInvalidSignature,
		Message: "The request signature does not match",
	}

	ErrSignatureExpired = &Error{
		Code:    ErrCodeSignatureExpired,
		Message: "The provided signature has expired",
	}

	ErrSignatureTTLExceeded = &Error{
		Code:    ErrCodeSignatureTTL,
		Message: "Signature expiration exceeds maximum allowed TTL",
	}

	ErrInvalidPath = &Error{
		Code:    ErrCodeInvalidPath,
		Message: "Request path does not match signed path",
	}

	ErrInvalidExpires = &Error{
		Code:    ErrCodeInvalidExpires,
		Message: "Invalid expiration timestamp",
	}
)
