package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"warpdrop/internal/storage"
)

func setupTestServer(t *testing.T) (*Server, string, string) {
	tempDir, err := os.MkdirTemp("", "warpdrop_server_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	diskStorage, err := storage.NewDiskStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create disk storage: %v", err)
	}

	testToken := "abcdef0123456789abcdef0123456789"
	srv := NewServer(Config{
		Port:    8080,
		Token:   testToken,
		Storage: diskStorage,
	})

	return srv, testToken, tempDir
}

func TestServerHealth(t *testing.T) {
	srv, _, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	srv.GetHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
}

func TestServerUploadAndDownloadFlow(t *testing.T) {
	srv, token, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Step 1: Upload multipart file
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("files", "test_doc.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	originalContent := []byte("this is verified warpdrop upload test data")
	fileWriter.Write(originalContent)
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/api/v1/upload?token="+token, &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	srv.GetHandler().ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload failed, expected 200 OK, got %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("failed to decode upload response: %v", err)
	}
	if uploadResp.Status != "success" || len(uploadResp.Files) != 1 {
		t.Fatalf("unexpected upload response payload: %+v", uploadResp)
	}
	savedFilename := uploadResp.Files[0].Name

	// Step 2: List files
	listReq := httptest.NewRequest("GET", "/api/v1/files?token="+token, nil)
	listRec := httptest.NewRecorder()
	srv.GetHandler().ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list files failed, expected 200 OK, got: %d", listRec.Code)
	}

	var files []storage.FileInfo
	if err := json.Unmarshal(listRec.Body.Bytes(), &files); err != nil {
		t.Fatalf("failed to decode file list: %v", err)
	}
	if len(files) != 1 || files[0].Name != savedFilename {
		t.Fatalf("expected 1 file with name %s, got: %+v", savedFilename, files)
	}

	// Step 3: Download file
	downloadReq := httptest.NewRequest("GET", "/api/v1/download?token="+token+"&file="+savedFilename, nil)
	downloadRec := httptest.NewRecorder()
	srv.GetHandler().ServeHTTP(downloadRec, downloadReq)

	if downloadRec.Code != http.StatusOK {
		t.Fatalf("download failed, expected 200 OK, got: %d", downloadRec.Code)
	}

	downloadedBytes, err := io.ReadAll(downloadRec.Body)
	if err != nil {
		t.Fatalf("failed to read downloaded body: %v", err)
	}
	if !bytes.Equal(downloadedBytes, originalContent) {
		t.Fatalf("downloaded content mismatch, expected '%s', got '%s'", originalContent, downloadedBytes)
	}

	// Step 4: Ensure unauthorized download is rejected
	unauthReq := httptest.NewRequest("GET", "/api/v1/download?file="+savedFilename, nil)
	unauthRec := httptest.NewRecorder()
	srv.GetHandler().ServeHTTP(unauthRec, unauthReq)

	if unauthRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on unauthenticated download, got: %d", unauthRec.Code)
	}
}
