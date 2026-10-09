@echo off
setlocal

echo ============================================================
echo Memulai WarpDrop - Local P2P File Transfer Engine
echo ============================================================

if not exist "warpdrop.exe" (
    echo [INFO] Mengompilasi binary warpdrop.exe...
    go build -ldflags="-s -w" -o warpdrop.exe ./cmd/warpdrop
    if errorlevel 1 (
        echo [ERROR] Gagal mengompilasi binary!
        pause
        exit /b 1
    )
)

echo [INFO] Menjalankan server...
warpdrop.exe %*

endlocal
