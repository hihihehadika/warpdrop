package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"warpdrop/internal/archive"
	"warpdrop/internal/storage"
)

type downloadAllErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// DownloadAllHandler streams all stored files as a single zip archive directly to client.
func DownloadAllHandler(provider storage.StorageProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metode HTTP tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		files, err := provider.ListFiles()
		if err != nil {
			log.Printf("[ERROR] Gagal membaca berkas untuk arsip ZIP: %v", err)
			http.Error(w, "Gagal membaca berkas penyimpanan", http.StatusInternalServerError)
			return
		}

		if len(files) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(downloadAllErrorResponse{
				Status:  "error",
				Message: "Tidak ada berkas untuk diunduh",
			})
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename=\"warpdrop_all.zip\"; filename*=UTF-8''warpdrop_all.zip")
		w.Header().Set("Content-Type", "application/zip")

		err = archive.StreamZip(w, files, provider)
		if err != nil {
			if errors.Is(err, archive.ErrNoFilesToArchive) {
				return
			}
			log.Printf("[WARN] Transmisi arsip ZIP terputus: %v", err)
		}
	}
}
