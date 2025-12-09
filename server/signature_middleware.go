package server

import (
	"net/http"

	"github.com/image-server/image-server/core/signature"
)

// SignatureMiddleware validates request signatures using HMAC-SHA256
type SignatureMiddleware struct {
	validator *signature.Validator
}

// NewSignatureMiddleware creates a new signature validation middleware
func NewSignatureMiddleware(config *signature.Config) *SignatureMiddleware {
	return &SignatureMiddleware{
		validator: signature.NewValidator(config),
	}
}

// ServeHTTP implements the negroni.Handler interface
func (m *SignatureMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !m.validator.ShouldValidate(r) {
		next(w, r)
		return
	}

	if err := m.validator.ValidateRequest(r); err != nil {
		signature.WriteError(w, err)
		return
	}

	next(w, r)
}
