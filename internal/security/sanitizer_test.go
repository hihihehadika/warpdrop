package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"normal.pdf", "normal.pdf"},
		{"../../evil.exe", "evil.exe"},
		{"..\\..\\Windows\\System32\\cmd.exe", "cmd.exe"},
		{"file:with*invalid?chars.txt", "file_with_invalid_chars.txt"},
		{"   spaced name.jpg   ", "spaced name.jpg"},
		{"", "file_unnamed"},
		{".", "file_unnamed"},
		{"..", "file_unnamed"},
		{"....", "file_unnamed"},
		{"/root/secret.key", "secret.key"},
	}

	for _, c := range cases {
		got := SanitizeFilename(c.input)
		if got != c.expected {
			t.Errorf("SanitizeFilename(%q) = %q; expected %q", c.input, got, c.expected)
		}
	}
}

func TestGetSafeDestinationPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "warpdrop_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test normal path resolution
	safePath, err := GetSafeDestinationPath(tempDir, "document.pdf")
	if err != nil {
		t.Fatalf("expected safe path, got error: %v", err)
	}
	expectedBase, _ := filepath.Abs(tempDir)
	if !strings.HasPrefix(safePath, expectedBase) {
		t.Fatalf("expected safePath to be inside %s, got: %s", expectedBase, safePath)
	}

	// Test collision resolution
	existingFile := filepath.Join(expectedBase, "document.pdf")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	collisionPath, err := GetSafeDestinationPath(tempDir, "document.pdf")
	if err != nil {
		t.Fatalf("unexpected error on collision: %v", err)
	}
	expectedCollision := filepath.Join(expectedBase, "document (1).pdf")
	if collisionPath != expectedCollision {
		t.Fatalf("expected %s on collision, got %s", expectedCollision, collisionPath)
	}

	// Test traversal attempt
	traversalPath, err := GetSafeDestinationPath(tempDir, "../../../outside.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(traversalPath) != expectedBase {
		t.Fatalf("traversal attack failed to be contained: got %s, expected parent %s", traversalPath, expectedBase)
	}
}
