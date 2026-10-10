package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"testing"

	"warpdrop/internal/storage"
)

func TestStreamZip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "warpdrop_zip_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider, err := storage.NewDiskStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to init storage provider: %v", err)
	}

	// Test Empty Files
	var emptyOut bytes.Buffer
	errEmpty := StreamZip(&emptyOut, []storage.FileInfo{}, provider)
	if errEmpty != ErrNoFilesToArchive {
		t.Fatalf("expected ErrNoFilesToArchive, got: %v", errEmpty)
	}

	// Prepare 2 files
	payload1 := []byte("first archive test document content")
	payload2 := []byte("second archive test document content data")

	info1, err := provider.SaveStream("doc1.txt", bytes.NewReader(payload1))
	if err != nil {
		t.Fatalf("failed to save doc1: %v", err)
	}

	info2, err := provider.SaveStream("doc2.txt", bytes.NewReader(payload2))
	if err != nil {
		t.Fatalf("failed to save doc2: %v", err)
	}

	// Stream to in-memory buffer
	var zipBuf bytes.Buffer
	err = StreamZip(&zipBuf, []storage.FileInfo{info1, info2}, provider)
	if err != nil {
		t.Fatalf("unexpected error streaming zip: %v", err)
	}

	// Read and verify zip structure
	zipReader, err := zip.NewReader(bytes.NewReader(zipBuf.Bytes()), int64(zipBuf.Len()))
	if err != nil {
		t.Fatalf("failed to open generated zip archive: %v", err)
	}

	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 files in zip archive, got: %d", len(zipReader.File))
	}

	for _, zf := range zipReader.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("failed to open zipped file %s: %v", zf.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read content of %s: %v", zf.Name, err)
		}

		if zf.Name == "doc1.txt" && !bytes.Equal(content, payload1) {
			t.Fatalf("content mismatch for doc1.txt: got %s", content)
		}
		if zf.Name == "doc2.txt" && !bytes.Equal(content, payload2) {
			t.Fatalf("content mismatch for doc2.txt: got %s", content)
		}
	}
}
