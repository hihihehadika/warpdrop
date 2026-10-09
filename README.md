# WarpDrop

A high-speed local file sharing tool built with Go. Transfer files between laptop and mobile devices over local Wi-Fi without internet or app installation.

---

## How It Works

1. Run `warpdrop.exe` on your laptop.
2. The terminal displays a QR Code and a local URL (e.g., `http://192.168.1.15:8080`).
3. Scan the QR Code with your phone's camera or open the URL in any browser.
4. Send files from phone to laptop or download files from laptop to phone — all at full Wi-Fi speed with zero internet required.

---

## Features

- **100% Offline:** Works entirely over local Wi-Fi or mobile hotspot. No internet, no cloud, no subscription.
- **Zero-Install on Mobile:** No app needed on the phone. Any browser works.
- **High-Speed Streaming:** File data is streamed directly to disk using buffered I/O. RAM usage stays flat regardless of file size.
- **Session Security:** Every server session generates a cryptographic token embedded in the QR Code URL. Unauthorized access attempts are rejected with HTTP 403.
- **Batch Upload:** Select and send multiple files at once from the browser.
- **Real-Time Progress:** Transfer speed (MB/s) and time remaining (ETA) are displayed in the browser during every transfer.
- **Path Traversal Protection:** All incoming filenames are sanitized before being written to disk.
- **Graceful Shutdown:** Press CTRL+C and the server closes all connections cleanly before exiting.

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

---

## File Locations

| Direction | Where the file goes |
|-----------|---------------------|
| Phone to Laptop | `./warpdrop_received/` (folder created automatically) |
| Laptop to Phone | Browser `Download` folder on the phone |

---

## Security Notes

- The session token is re-generated every time the server starts.
- Only devices that scan your QR Code (or receive the URL directly from you) can access the server.
- All file paths are sanitized on the server side. Malicious filenames like `../../system/file` are rejected automatically.
- The server only binds to your local network interface. It is not accessible from the internet.

---

## Architecture

```
warpdrop/
├── cmd/warpdrop/         Entry point (CLI bootstrapper)
├── internal/
│   ├── network/          LAN IP detection
│   ├── security/         Token generation and file path sanitization
│   ├── storage/          Extensible storage interface and disk implementation
│   ├── server/           HTTP handlers, middleware, routing
│   └── qr/               ASCII QR Code terminal renderer
└── web/static/           Embedded browser UI (HTML, CSS, JS)
```
