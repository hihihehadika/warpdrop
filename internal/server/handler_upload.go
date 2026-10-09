package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"warpdrop/internal/storage"
)

// UploadResponse represents JSON response sent after file upload processing.
type UploadResponse struct {
	Status  string             `json:"status"`
	Files   []storage.FileInfo `json:"files"`
	Message string             `json:"message"`
}

// UploadHandler processes incoming multipart file streams and persists them via StorageProvider.
// It uses r.MultipartReader to stream directly to disk without loading file bytes into RAM.
func UploadHandler(provider storage.StorageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metode HTTP tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		reader, err := r.MultipartReader()
		if err != nil {
			log.Printf("[ERROR] Gagal menginisialisasi multipart reader: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Status:  "error",
				Message: "Format payload multipart tidak valid",
			})
			return
		}

		var savedFiles []storage.FileInfo
		startTime := time.Now()

		for {
			part, err := reader.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Printf("[ERROR] Terjadi gangguan saat membaca multipart stream: %v", err)
				break
			}

			filename := part.FileName()
			if filename == "" {
				part.Close()
				continue
			}

			partStartTime := time.Now()
			info, err := provider.SaveStream(filename, part)
			part.Close()

			if err != nil {
				log.Printf("[ERROR] Gagal menyimpan berkas '%s': %v", filename, err)
				continue
			}

			partDuration := time.Since(partStartTime)
			var speedMBps float64
			if partDuration.Seconds() > 0 {
				speedMBps = (float64(info.Size) / (1024 * 1024)) / partDuration.Seconds()
			}

			log.Printf("[SUCCESS] File diterima: %s (%s) dalam %.2fs | Kecepatan: %.2f MB/s",
				info.Name, info.HumanSize, partDuration.Seconds(), speedMBps)

			savedFiles = append(savedFiles, info)
		}

		if len(savedFiles) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(UploadResponse{
				Status:  "error",
				Message: "Tidak ada berkas yang berhasil disimpan",
			})
			return
		}

		totalDuration := time.Since(startTime)
		log.Printf("[INFO] Selesai memproses %d berkas dalam %.2fs", len(savedFiles), totalDuration.Seconds())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UploadResponse{
			Status:  "success",
			Files:   savedFiles,
			Message: "File berhasil diterima dan disimpan di laptop",
		})
	}
}
