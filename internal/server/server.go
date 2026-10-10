package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"warpdrop/internal/auth"
	"warpdrop/internal/storage"
	"warpdrop/internal/ws"
)

// Config holds runtime configuration options for WarpDrop HTTP server.
type Config struct {
	Port       int
	Token      string
	PINManager *auth.PINManager
	WSHub      *ws.Hub
	Storage    storage.StorageProvider
	StaticFS   http.FileSystem
}

// Server encapsulates http.Server and lifecycle management.
type Server struct {
	httpServer *http.Server
	config     Config
}

// NewServer initializes HTTP multiplexer, middlewares, and routes.
func NewServer(cfg Config) *Server {
	mux := http.NewServeMux()

	// Public health check route
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"online","version":"1.0.0"}`))
	})

	// Public PIN authentication exchange route
	if cfg.PINManager != nil {
		mux.HandleFunc("/api/v1/auth/pin", PINAuthHandler(cfg.PINManager))
	}

	// Protected API routes
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/v1/upload", UploadHandler(cfg.Storage))
	apiMux.HandleFunc("/api/v1/files", ListFilesHandler(cfg.Storage))
	apiMux.HandleFunc("/api/v1/download", DownloadHandler(cfg.Storage))
	apiMux.HandleFunc("/api/v1/download-all", DownloadAllHandler(cfg.Storage))
	apiMux.HandleFunc("/api/v1/ws", WSHandler(cfg.WSHub))

	// Mount protected API routes with AuthMiddleware
	mux.Handle("/api/v1/upload", AuthMiddleware(cfg.Token, apiMux))
	mux.Handle("/api/v1/files", AuthMiddleware(cfg.Token, apiMux))
	mux.Handle("/api/v1/download", AuthMiddleware(cfg.Token, apiMux))
	mux.Handle("/api/v1/download-all", AuthMiddleware(cfg.Token, apiMux))
	mux.Handle("/api/v1/ws", AuthMiddleware(cfg.Token, apiMux))

	// Static web assets handler (if provided)
	if cfg.StaticFS != nil {
		fileServer := http.FileServer(cfg.StaticFS)
		mux.Handle("/", StaticSessionMiddleware(cfg.Token, fileServer))
	}

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Minute,
		WriteTimeout: 30 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpSrv,
		config:     cfg,
	}
}

// Start begins listening on configured TCP port.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server, waiting for active connections to drain.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// GetHandler returns the underlying http.Handler for testing purposes.
func (s *Server) GetHandler() http.Handler {
	return s.httpServer.Handler
}
