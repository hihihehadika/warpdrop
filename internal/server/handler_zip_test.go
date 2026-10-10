package server

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"warpdrop/internal/storage"
)

func TestDownloadAllHandler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "warpdrop_zip_handler_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	diskStorage, err := storage.NewDiskStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create disk storage: %v", err)
	}

	handler := DownloadAllHandler(diskStorage)

	// Case 1: Empty storage returns 400 Bad Request
	reqEmpty := httptest.NewRequest("GET", "/api/v1/download-all", nil)
	recEmpty := httptest.NewRecorder()
	handler.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request on empty storage, got: %d", recEmpty.Code)
	}

	// Case 2: Storage with files returns 200 OK and valid ZIP attachment
	diskStorage.SaveStream("file1.txt", bytes.NewReader([]byte("content of file 1")))
	diskStorage.SaveStream("file2.txt", bytes.NewReader([]byte("content of file 2")))

	reqValid := httptest.NewRequest("GET", "/api/v1/download-all", nil)
	recValid := httptest.NewRecorder()
	handler.ServeHTTP(recValid, reqValid)

	if recValid.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on valid download-all, got: %d", recValid.Code)
	}

	contentType := recValid.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Fatalf("expected Content-Type application/zip, got: %s", contentType)
	}

	// Verify valid zip content
	bodyBytes := recValid.Body.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		t.Fatalf("failed to parse returned zip stream: %v", err)
	}

	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 zipped files, got: %d", len(zipReader.File))
	}
}
