package qr

import (
	"fmt"
	"strings"

	"github.com/skip2/go-qrcode"
)

// RenderTerminalQR converts an input URL into an ANSI high-contrast terminal QR code.
// It uses ANSI background color sequences to ensure accurate scanning on both
// dark and light terminal color themes.
func RenderTerminalQR(url string) (string, error) {
	code, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return "", err
	}

	bitmap := code.Bitmap()
	var sb strings.Builder

	// ANSI color block escapes
	const (
		white = "\033[47m  \033[0m"
		black = "\033[40m  \033[0m"
	)

	// Add top quiet zone (border)
	borderSize := 2
	totalWidth := len(bitmap) + (borderSize * 2)
	for i := 0; i < borderSize; i++ {
		for j := 0; j < totalWidth; j++ {
			sb.WriteString(white)
		}
		sb.WriteString("\n")
	}

	// Render QR rows with left and right quiet zone
	for _, row := range bitmap {
		for i := 0; i < borderSize; i++ {
			sb.WriteString(white)
		}

		for _, isBlack := range row {
			if isBlack {
				sb.WriteString(black)
			} else {
				sb.WriteString(white)
			}
		}

		for i := 0; i < borderSize; i++ {
			sb.WriteString(white)
		}
		sb.WriteString("\n")
	}

	// Add bottom quiet zone (border)
	for i := 0; i < borderSize; i++ {
		for j := 0; j < totalWidth; j++ {
			sb.WriteString(white)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FormatBanner generates a formatted status banner containing connection instructions.
func FormatBanner(ip string, port int, token string, storageDir string) string {
	url := fmt.Sprintf("http://%s:%d/?token=%s", ip, port, token)
	qrArt, err := RenderTerminalQR(url)
	if err != nil {
		qrArt = fmt.Sprintf("[WARN] Gagal membuat visual QR Code: %v\n", err)
	}

	separator := strings.Repeat("=", 64)

	var sb strings.Builder
	sb.WriteString("\n" + separator + "\n")
	sb.WriteString("WARPDROP - High-Performance Local P2P File Drop Engine\n")
	sb.WriteString(separator + "\n")
	sb.WriteString(fmt.Sprintf("[STATUS]  Server aktif pada port: %d\n", port))
	sb.WriteString(fmt.Sprintf("[NETWORK] Alamat IP Lokal: %s\n", ip))
	sb.WriteString(fmt.Sprintf("[FOLDER]  Direktori Penyimpanan: %s\n", storageDir))
	sb.WriteString("\nArahkan kamera HP ke QR Code di bawah untuk menghubungkan:\n\n")
	sb.WriteString(qrArt)
	sb.WriteString("\nAtau buka alamat berikut secara manual di browser HP:\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", url))
	sb.WriteString("[PANDUAN OPERASIONAL]\n")
	sb.WriteString("1. Pastikan HP dan Laptop terhubung ke WiFi atau Hotspot yang sama.\n")
	sb.WriteString("2. Jika koneksi gagal, router kemungkinan mengaktifkan AP Isolation.\n")
	sb.WriteString("   Solusi: Aktifkan Mobile Hotspot pada HP atau Laptop, lalu hubungkan.\n")
	sb.WriteString("3. Tekan CTRL+C di terminal ini untuk menghentikan server dengan aman.\n")
	sb.WriteString(separator + "\n\n")

	return sb.String()
}
