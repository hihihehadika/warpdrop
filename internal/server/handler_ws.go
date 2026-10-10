package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"warpdrop/internal/ws"
)

// WSHandler upgrades HTTP request to a persistent WebSocket connection managed by WSHub.
func WSHandler(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hub == nil {
			http.Error(w, "WebSocket hub belum aktif", http.StatusInternalServerError)
			return
		}

		clientID := r.URL.Query().Get("id")
		if clientID == "" {
			randomBytes := make([]byte, 8)
			rand.Read(randomBytes)
			clientID = hex.EncodeToString(randomBytes)
		}

		ws.ServeWs(hub, w, r, clientID)
	}
}
