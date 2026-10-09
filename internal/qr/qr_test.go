package qr

import (
	"strings"
	"testing"
)

func TestRenderTerminalQR(t *testing.T) {
	testURL := "http://192.168.1.15:8080/?token=testtoken12345"
	rendered, err := RenderTerminalQR(testURL)
	if err != nil {
		t.Fatalf("unexpected error rendering QR code: %v", err)
	}

	if len(rendered) == 0 {
		t.Fatal("expected non-empty rendered QR string")
	}

	// Must contain ANSI color reset sequence and multiple rows
	if !strings.Contains(rendered, "\033[0m") {
		t.Fatal("expected rendered QR to contain ANSI reset sequences")
	}

	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	if len(lines) < 20 {
		t.Fatalf("expected QR matrix to have at least 20 rows, got: %d", len(lines))
	}
}

func TestFormatBanner(t *testing.T) {
	ip := "192.168.43.15"
	port := 8080
	token := "abcdef0123456789"
	storageDir := "./warpdrop_received"

	banner := FormatBanner(ip, port, token, storageDir)

	expectedSubstrings := []string{
		"WARPDROP",
		"192.168.43.15",
		"8080",
		"abcdef0123456789",
		storageDir,
		"[PANDUAN OPERASIONAL]",
		"AP Isolation",
		"CTRL+C",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(banner, substr) {
			t.Errorf("expected banner to contain %q, but was missing", substr)
		}
	}
}
