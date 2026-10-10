package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"warpdrop/internal/auth"
)

type pinAuthRequest struct {
	PIN string `json:"pin"`
}

type pinAuthResponse struct {
	Status  string `json:"status"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message"`
}

// PINAuthHandler handles PIN verification and exchanges valid PINs for session cookies.
func PINAuthHandler(pinMgr *auth.PINManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metode HTTP tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		clientIP := r.RemoteAddr
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}

		var req pinAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(pinAuthResponse{
				Status:  "error",
				Message: "Format payload JSON tidak valid",
			})
			return
		}

		cleanPIN := strings.TrimSpace(req.PIN)
		ok, err := pinMgr.VerifyPIN(clientIP, cleanPIN)
		if !ok || err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, auth.ErrIPLockedOut) {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(pinAuthResponse{
					Status:  "error",
					Message: "Terlalu banyak percobaan gagal. IP diblokir selama 60 detik.",
				})
				return
			}

			if errors.Is(err, auth.ErrPINExpired) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(pinAuthResponse{
					Status:  "error",
					Message: "Kode PIN telah kedaluwarsa.",
				})
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(pinAuthResponse{
				Status:  "error",
				Message: "Kode PIN salah.",
			})
			return
		}

		// Set authentication session cookie
		sessionToken := pinMgr.GetSessionToken()
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(pinAuthResponse{
			Status:  "success",
			Token:   sessionToken,
			Message: "Autentikasi PIN berhasil",
		})
	}
}
