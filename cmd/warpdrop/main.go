package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"warpdrop/internal/auth"
	"warpdrop/internal/network"
	"warpdrop/internal/qr"
	"warpdrop/internal/security"
	"warpdrop/internal/server"
	"warpdrop/internal/storage"
	"warpdrop/internal/ws"
	"warpdrop/web"
)

func main() {
	portFlag := flag.Int("port", 8080, "Port TCP untuk server HTTP WarpDrop")
	dirFlag := flag.String("dir", "./warpdrop_received", "Direktori penyimpanan file yang diterima")
	tokenFlag := flag.String("token", "", "Token autentikasi kustom (opsional)")
	flag.Parse()

	// Inisialisasi penyimpanan disk
	diskStorage, err := storage.NewDiskStorageProvider(*dirFlag)
	if err != nil {
		log.Fatalf("[FATAL] Gagal menginisialisasi direktori penyimpanan '%s': %v", *dirFlag, err)
	}

	// Inisialisasi token autentikasi sesi
	token := *tokenFlag
	if token == "" {
		generatedToken, err := security.GenerateToken(16)
		if err != nil {
			log.Fatalf("[FATAL] Gagal membuat token kriptografi sesi: %v", err)
		}
		token = generatedToken
	}

	// Inisialisasi pengelola 4-digit Quick PIN
	pinMgr := auth.NewPINManager(token, 10*time.Minute)
	activePIN, _ := pinMgr.GetActivePIN()

	// Inisialisasi WebSocket Hub untuk Real-Time Shared Clipboard
	wsHub := ws.NewHub()
	go wsHub.Run()
	defer wsHub.Stop()

	// Deteksi alamat IP lokal aktif
	localIP, err := network.GetLocalIP()
	if err != nil {
		log.Printf("[WARN] Tidak dapat mendeteksi IP otomatis: %v. Menggunakan 127.0.0.1", err)
		localIP = "127.0.0.1"
	}

	// Inisialisasi server HTTP dengan static filesystem yang di-embed
	srv := server.NewServer(server.Config{
		Port:       *portFlag,
		Token:      token,
		PINManager: pinMgr,
		WSHub:      wsHub,
		Storage:    diskStorage,
		StaticFS:   web.GetFileSystem(),
	})

	// Cetak banner informasi, Quick PIN, dan QR Code di terminal
	banner := qr.FormatBanner(localIP, *portFlag, token, activePIN, diskStorage.GetBaseDir())
	fmt.Print(banner)

	// Jalankan server di goroutine terpisah
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] Server WarpDrop mulai mendengarkan pada :%d", *portFlag)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Saluran penangkap sinyal interupsi OS untuk graceful shutdown
	shutdownSig := make(chan os.Signal, 1)
	signal.Notify(shutdownSig, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] Kesalahan fatal saat menjalankan server: %v", err)

	case sig := <-shutdownSig:
		log.Printf("[INFO] Menerima sinyal terminasi (%v). Menghentikan server WarpDrop dengan aman...", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[WARN] Kesalahan saat shutdown server: %v", err)
		} else {
			log.Printf("[INFO] Seluruh koneksi ditutup secara bersih. Sampai jumpa.")
		}
	}
}
