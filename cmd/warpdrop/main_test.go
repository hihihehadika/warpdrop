package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"warpdrop/internal/security"
	"warpdrop/internal/server"
	"warpdrop/internal/storage"
	"warpdrop/web"
)

func TestAppBootstrapAndGracefulShutdown(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "warpdrop_main_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	diskStorage, err := storage.NewDiskStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	token, err := security.GenerateToken(16)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Use port 0 or a high test port to avoid collision
	srv := server.NewServer(server.Config{
		Port:     18080,
		Token:    token,
		Storage:  diskStorage,
		StaticFS: web.GetFileSystem(),
	})

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Give server brief moment to start
	time.Sleep(100 * time.Millisecond)

	// Test reaching health endpoint
	resp, err := http.Get("http://127.0.0.1:18080/api/v1/health")
	if err != nil {
		t.Fatalf("failed to reach health endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from health, got: %d", resp.StatusCode)
	}

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("server failed to shutdown cleanly: %v", err)
	}
}
