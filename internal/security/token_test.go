package security

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(16)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	if len(token) != 32 {
		t.Fatalf("expected 32 character hex string for 16 bytes, got %d chars: %s", len(token), token)
	}

	token2, err := GenerateToken(16)
	if err != nil {
		t.Fatalf("unexpected error generating second token: %v", err)
	}

	if token == token2 {
		t.Fatal("subsequent token generation must produce unique values")
	}

	_, errShort := GenerateToken(4)
	if errShort != ErrInvalidTokenLength {
		t.Fatalf("expected ErrInvalidTokenLength, got: %v", errShort)
	}
}

func TestValidateToken(t *testing.T) {
	expectedToken := "4a7c8b9d0e1f2a3b4c5d6e7f8a9b0c1d"

	if !ValidateToken(expectedToken, expectedToken) {
		t.Fatal("expected matching tokens to return true")
	}

	tamperedToken := "4a7c8b9d0e1f2a3b4c5d6e7f8a9b0c1e"
	if ValidateToken(tamperedToken, expectedToken) {
		t.Fatal("expected mismatched tokens to return false")
	}

	if ValidateToken("", expectedToken) {
		t.Fatal("expected empty provided token to return false")
	}

	if ValidateToken(expectedToken, "") {
		t.Fatal("expected empty expected token to return false")
	}

	if ValidateToken("short", expectedToken) {
		t.Fatal("expected length mismatch to return false")
	}
}
