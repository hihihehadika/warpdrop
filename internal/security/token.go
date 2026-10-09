package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

// ErrInvalidTokenLength returned when requested token length is less than 8 bytes.
var ErrInvalidTokenLength = errors.New("token length must be at least 8 bytes")

// GenerateToken generates a cryptographically secure random hexadecimal token.
// The length parameter specifies the number of random bytes (resulting hex string is 2x length).
func GenerateToken(length int) (string, error) {
	if length < 8 {
		return "", ErrInvalidTokenLength
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// ValidateToken compares two token strings in constant time to mitigate timing attacks.
func ValidateToken(provided, expected string) bool {
	if len(provided) == 0 || len(expected) == 0 {
		return false
	}

	if len(provided) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
