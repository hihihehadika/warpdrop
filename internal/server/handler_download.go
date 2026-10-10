package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"warpdrop/internal/storage"
)

// ListFilesHandler returns JSON array of all stored files.
func ListFilesHandler(provider storage.StorageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metode HTTP tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		files, err := provider.ListFiles()
		if err != nil {
			log.Printf("[ERROR] Gagal membaca daftar berkas: %v", err)
			http.Error(w, "Gagal membaca daftar berkas di server", http.StatusInternalServerError)
			return
		}

		if files == nil {
			files = []storage.FileInfo{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(files)
	}
}

// DownloadHandler serves a requested file via streaming with attachment disposition headers.
func DownloadHandler(provider storage.StorageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metode HTTP tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		filename := r.URL.Query().Get("file")
		if filename == "" {
			http.Error(w, "Parameter 'file' wajib disertakan", http.StatusBadRequest)
			return
		}

		reader, info, err := provider.GetFileReader(filename)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Berkas tidak ditemukan", http.StatusNotFound)
				return
			}
			log.Printf("[ERROR] Gagal membuka berkas untuk unduhan: %v", err)
			http.Error(w, "Terjadi kesalahan internal saat membuka berkas", http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		safeAscii := strings.ReplaceAll(strings.ReplaceAll(info.Name, "\"", ""), ";", "")
		encodedUtf8 := url.PathEscape(info.Name)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", safeAscii, encodedUtf8))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))

		buf := make([]byte, 64*1024)
		_, err = io.CopyBuffer(w, reader, buf)
		if err != nil {
			log.Printf("[WARN] Transmisi unduhan terputus untuk '%s': %v", info.Name, err)
		}
	}
}
