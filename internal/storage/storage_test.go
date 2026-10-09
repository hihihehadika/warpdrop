package storage

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestDiskStorageProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "warpdrop_storage_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider, err := NewDiskStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage provider: %v", err)
	}

	payload := []byte("hello warpdrop stream payload data")
	info, err := provider.SaveStream("sample.txt", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("unexpected error saving stream: %v", err)
	}

	if info.Name != "sample.txt" {
		t.Fatalf("expected name sample.txt, got %s", info.Name)
	}
	if info.Size != int64(len(payload)) {
		t.Fatalf("expected size %d, got %d", len(payload), info.Size)
	}

	// Test ListFiles
	files, err := provider.ListFiles()
	if err != nil {
		t.Fatalf("unexpected error listing files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name != "sample.txt" {
		t.Fatalf("expected file sample.txt, got %s", files[0].Name)
	}

	// Test GetFileReader
	reader, readInfo, err := provider.GetFileReader("sample.txt")
	if err != nil {
		t.Fatalf("unexpected error opening reader: %v", err)
	}
	defer reader.Close()

	readBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read from reader: %v", err)
	}
	if !bytes.Equal(readBytes, payload) {
		t.Fatal("content read back does not match original payload")
	}
	if readInfo.Size != int64(len(payload)) {
		t.Fatalf("expected size %d, got %d", len(payload), readInfo.Size)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, c := range cases {
		got := FormatBytes(c.bytes)
		if got != c.expected {
			t.Errorf("FormatBytes(%d) = %q; expected %q", c.bytes, got, c.expected)
		}
	}
}
