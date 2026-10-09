package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ErrPathTraversal detected when resolved destination escapes base directory boundary.
var ErrPathTraversal = errors.New("potential path traversal attempt detected")

// SanitizeFilename strips directory components, control characters, and unsafe symbols.
func SanitizeFilename(raw string) string {
	cleaned := filepath.Base(raw)

	// Strip drive letter prefixes in Windows paths if present
	if len(cleaned) >= 2 && cleaned[1] == ':' {
		cleaned = cleaned[2:]
	}

	var sb strings.Builder
	for _, r := range cleaned {
		if r == 0 || unicode.IsControl(r) {
			continue
		}
		// Disallow path separators and characters invalid in Windows filenames
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			sb.WriteRune('_')
			continue
		}
		sb.WriteRune(r)
	}

	result := strings.TrimSpace(sb.String())
	result = strings.Trim(result, ".")

	if result == "" || result == "." || result == ".." {
		return "file_unnamed"
	}

	return result
}

// GetSafeDestinationPath validates that target file path resides strictly inside baseDir.
// If a file with the same name already exists in baseDir, it resolves collision by appending
// sequential index like 'name (1).ext'.
func GetSafeDestinationPath(baseDir, filename string) (string, error) {
	safeName := SanitizeFilename(filename)

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(absBase, safeName)
	cleanTarget := filepath.Clean(targetPath)

	// Ensure destination does not escape base boundary
	rel, err := filepath.Rel(absBase, cleanTarget)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." && safeName != "." {
		return "", ErrPathTraversal
	}

	// Resolve collision if file already exists
	if _, err := os.Stat(cleanTarget); err == nil {
		ext := filepath.Ext(safeName)
		nameWithoutExt := strings.TrimSuffix(safeName, ext)

		for counter := 1; counter <= 10000; counter++ {
			candidateName := fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
			candidatePath := filepath.Join(absBase, candidateName)
			if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
				return candidatePath, nil
			}
		}
	}

	return cleanTarget, nil
}
