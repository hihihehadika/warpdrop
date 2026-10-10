# WarpDrop

A high-speed local file sharing and text synchronization tool built with Go. Transfer files and sync clipboard contents between laptop and mobile devices over local Wi-Fi without internet or app installation.

---

## How It Works

1. Run `warpdrop.exe` on your laptop.
2. The terminal displays a QR Code, an active 4-digit Quick PIN, and a local URL (e.g., `http://192.168.1.15:8080`).
3. Connect instantly using either method:
   - Scan the QR Code with your phone camera for zero-click automatic authentication.
   - Or open the URL manually in any browser and enter the 4-digit Quick PIN.
4. Send files, sync clipboard text in real time, or download files individually and as a bundled ZIP archive.

---

## Features

- **100% Offline:** Works entirely over local Wi-Fi or mobile hotspot. No internet, no cloud, no subscription.
- **Zero-Install on Mobile:** No application required on the phone. Compatible with all modern mobile browsers.
- **Real-Time Shared Clipboard:** Instantly synchronize text, notes, and URLs bidirectionally via persistent WebSockets with 1-click clipboard copy.
- **4-Digit Quick PIN Authentication:** Authenticate without camera scanners using an ephemeral 4-digit numeric code with automated brute-force IP lockout protection.
- **On-The-Fly Streaming ZIP ("Download All"):** Streamline downloads of all stored files in a single compressed ZIP archive with flat memory consumption (zero RAM spikes).
- **High-Speed Streaming:** File uploads and downloads are streamed chunk-by-chunk using buffered I/O. RAM usage stays low regardless of file size.
- **DoS Body Size Limits:** Enforces a 10 GB payload ceiling with `http.MaxBytesReader` to protect host storage from disk exhaustion attacks.
- **Session Security:** Every server session generates a cryptographic token. Sensitive tokens are stripped from browser address bars and hidden in terminal logs.
- **Real-Time Speedometer:** Live transfer speed (MB/s), sent progress, and estimated time of arrival (ETA) displayed in the browser.
- **Path Traversal Protection:** All filenames are sanitized against directory traversal attacks.
- **Graceful Shutdown:** Intercepts OS signals (`SIGINT`, `SIGTERM`) to cleanly drain active network connections.

---

## Requirements

- Go 1.22 or newer (for building from source)
- Windows (primary target), Linux and macOS supported

---

## Quick Start

### Run from Source

```bash
git clone https://github.com/hihihehadika/warpdrop.git
cd warpdrop
go run ./cmd/warpdrop
```

### Build Standalone Executable

```bash
go build -ldflags="-s -w" -o warpdrop.exe ./cmd/warpdrop
./warpdrop.exe
```

### Optional Flags

```bash
warpdrop.exe -port 9090 -dir "C:\Users\user\Desktop\received"
```

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8080` | HTTP server port |
| `-dir` | `./warpdrop_received` | Directory where uploaded files are saved |
| `-token` | `""` | Optional fixed authentication token |

---

## File Locations

| Direction | Where the file goes |
|-----------|---------------------|
| Phone to Laptop | `./warpdrop_received/` (folder created automatically) |
| Laptop to Phone | Browser `Download` folder on the phone |

---

## Security Architecture

- **Ephemeral Quick PIN:** 4-digit numeric code expires every 10 minutes. Consecutive failed attempts trigger an automated 60-second IP lockout (HTTP 429).
- **Constant-Time Verification:** All token and PIN comparisons use `crypto/subtle.ConstantTimeCompare` to eliminate timing side-channel attacks.
- **Token URL Stripping:** The client web application cleans `?token=...` query parameters from the browser history upon initial load.
- **Streaming Boundaries:** The ZIP streamer and multipart uploader process files in 64 KB buffers, preventing memory exhaustion and denial-of-service attempts.

---

## Architecture

```
warpdrop/
├── cmd/warpdrop/         Entry point (CLI bootstrapper and lifecycle management)
├── internal/
│   ├── auth/             Ephemeral 4-digit PIN manager and brute-force lockout
│   ├── ws/               WebSocket connection pool and real-time broadcast hub
│   ├── archive/          On-the-fly streaming ZIP compressor
│   ├── network/          LAN IP address autodetection
│   ├── security/         Cryptographic token generation and filename sanitization
│   ├── storage/          Extensible storage interface and disk implementation
│   ├── server/           HTTP endpoints, handlers, and session authentication middleware
│   └── qr/               ANSI QR Code terminal renderer and banner formatter
└── web/static/           Embedded single-page application (HTML, CSS, JavaScript)
```
