package web

import (
	"io"
	"strings"
	"testing"
)

func TestGetFileSystem(t *testing.T) {
	fsys := GetFileSystem()

	// Verify index.html exists in embedded filesystem
	file, err := fsys.Open("index.html")
	if err != nil {
		t.Fatalf("failed to open embedded index.html: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read embedded index.html: %v", err)
	}

	if !strings.Contains(string(content), "WARPDROP") {
		t.Fatal("expected index.html to contain brand title WARPDROP")
	}

	// Verify style.css exists
	cssFile, err := fsys.Open("style.css")
	if err != nil {
		t.Fatalf("failed to open embedded style.css: %v", err)
	}
	cssFile.Close()

	// Verify app.js exists
	jsFile, err := fsys.Open("app.js")
	if err != nil {
		t.Fatalf("failed to open embedded app.js: %v", err)
	}
	jsFile.Close()
}
