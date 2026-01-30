package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ValidateAlchemySignature validates the X-Alchemy-Signature header
// Returns true if the signature is valid, false otherwise
func ValidateAlchemySignature(body []byte, signature string, signingKey string) bool {
	if signingKey == "" || signature == "" {
		return false
	}

	// Compute HMAC-SHA256 hash
	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures using constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GenerateAlchemySignature generates a signature for testing purposes
func GenerateAlchemySignature(body []byte, signingKey string) string {
	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// SignatureError represents a webhook signature validation error
type SignatureError struct {
	Message string
}

func (e *SignatureError) Error() string {
	return fmt.Sprintf("webhook signature validation failed: %s", e.Message)
}

// NewSignatureError creates a new signature error
func NewSignatureError(message string) *SignatureError {
	return &SignatureError{Message: message}
}
