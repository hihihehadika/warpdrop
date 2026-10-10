package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"warpdrop/internal/security"
)

const sessionCookieName = "warpdrop_session"

// AuthMiddleware inspects incoming requests for a valid authentication token.
// The token can be supplied via:
// 1. URL Query parameter: '?token=...'
// 2. HTTP Header: 'X-WarpDrop-Token: ...'
// 3. HTTP Cookie: 'warpdrop_session=...'
// If authenticated, it sets a session cookie on the response so subsequent
// requests in the same browser session remain authorized.
func AuthMiddleware(expectedToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("X-WarpDrop-Token")
		}
		if token == "" {
			if cookie, err := r.Cookie(sessionCookieName); err == nil {
				token = cookie.Value
			}
		}

		if !security.ValidateToken(token, expectedToken) {
			clientIP := r.RemoteAddr
			if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
				clientIP = clientIP[:idx]
			}
			log.Printf("[WARN] Akses ilegal ditolak dari IP: %s (path: %s)", clientIP, r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"status":"error","message":"Akses ditolak: Token autentikasi tidak valid"}`))
			return
		}

		// Refresh or set session cookie for seamless browser navigation
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    expectedToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		next.ServeHTTP(w, r)
	})
}

// StaticSessionMiddleware captures ?token=... on static web requests to set the session cookie,
// but does not reject unauthenticated visitors so the client UI can present the 4-digit PIN login modal.
func StaticSessionMiddleware(expectedToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != "" && security.ValidateToken(token, expectedToken) {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    expectedToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(24 * time.Hour),
			})
		}
		next.ServeHTTP(w, r)
	})
}
